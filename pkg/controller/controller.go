package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	kubeinformers "k8s.io/client-go/informers"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1core "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	kubervisorapi "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	kubervisorclient "github.com/amadeusitgroup/podkubervisor/pkg/client"
	bclient "github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned"
	binformers "github.com/amadeusitgroup/podkubervisor/pkg/client/informers/externalversions"
	blisters "github.com/amadeusitgroup/podkubervisor/pkg/client/listers/kubervisor/v1"
	"github.com/amadeusitgroup/podkubervisor/pkg/controller/item"
	"github.com/amadeusitgroup/podkubervisor/pkg/labeling"
	"github.com/amadeusitgroup/podkubervisor/pkg/pod"
)

func init() {
	prometheus.MustRegister(kubervisorGauges)
}

var (
	kubervisorGauges = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "kubervisor_breaker_gauge",
			Help: "Display Pod under kubervisor management",
		},
		[]string{"name", "type"}, // type={managed,breaked,paused,unknown}
	)
)

// Controller represent the kubervisor controller
type Controller struct {
	nbWorker uint32
	Logger   *zap.Logger

	recorder record.EventRecorder

	kubeInformerFactory    kubeinformers.SharedInformerFactory
	breakerInformerFactory binformers.SharedInformerFactory

	kubeClient    clientset.Interface
	breakerClient bclient.Interface

	breakerLister blisters.KubervisorServiceLister
	BreakerSynced cache.InformerSynced

	podLister corev1listers.PodLister
	PodSynced cache.InformerSynced

	serviceLister corev1listers.ServiceLister
	ServiceSynced cache.InformerSynced

	queue       workqueue.RateLimitingInterface // KubervisorServices to be synced
	enqueueFunc func(bc *kubervisorapi.KubervisorService)

	items                 item.KubervisorServiceItemStore
	updateHandlerFunc     func(*kubervisorapi.KubervisorService) (*kubervisorapi.KubervisorService, error)
	podControl            pod.ControlInterface
	rootContext           context.Context
	rootContextCancelFunc context.CancelFunc

	// Kubernetes Probes handler
	health healthcheck.Handler

	httpServer *http.Server

	gc *garbageCollector
}

// New returns new Controller instance
func New(cfg *Config) *Controller {
	sugar := cfg.Logger.Sugar()
	kubeConfig, err := initKubeConfig(cfg)
	if err != nil {
		sugar.Fatalf("Unable to init kubervisor controller: %v", err)
	}

	// apiextensionsclientset, err :=
	extClient, err := apiextensionsclient.NewForConfig(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to init clientset from kubeconfig:%v", err)
	}

	_, err = kubervisorclient.DefineKubervisorResources(extClient)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		sugar.Fatalf("Unable to define KubervisorService resource:%v", err)
	}

	kubeClient, err := clientset.NewForConfig(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to initialize kubeClient:%v", err)
	}

	breakerClient, err := kubervisorclient.NewClient(kubeConfig)
	if err != nil {
		sugar.Fatalf("Unable to init kubervisor.clientset from kubeconfig:%v", err)
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	breakerInformerFactory := binformers.NewSharedInformerFactory(breakerClient, time.Second*30)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(sugar.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})

	podInformer := kubeInformerFactory.Core().V1().Pods()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	breakerInformer := breakerInformerFactory.Breaker().V1().KubervisorServices()

	ctx, ctxCancel := context.WithCancel(context.Background())

	ctrl := &Controller{
		nbWorker: cfg.NbWorker,
		Logger:   cfg.Logger,

		kubeInformerFactory:    kubeInformerFactory,
		kubeClient:             kubeClient,
		breakerInformerFactory: breakerInformerFactory,
		breakerClient:          breakerClient,
		podLister:              podInformer.Lister(),
		PodSynced:              podInformer.Informer().HasSynced,
		serviceLister:          serviceInformer.Lister(),
		ServiceSynced:          serviceInformer.Informer().HasSynced,
		breakerLister:          breakerInformer.Lister(),
		BreakerSynced:          breakerInformer.Informer().HasSynced,

		podControl:            pod.NewPodControl(kubeClient),
		rootContext:           ctx,
		rootContextCancelFunc: ctxCancel,

		items: item.NewBreackerConfigItemStore(),

		httpServer: &http.Server{Addr: cfg.ListenAddr},

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "kubervisorservice"),
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "kubervisor-controller"}),
	}
	ctrl.enqueueFunc = ctrl.enqueue
	ctrl.updateHandlerFunc = ctrl.updateHandler
	ctrl.configureHTTPServer()

	breakerInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddKubervisorService,
			UpdateFunc: ctrl.onUpdateKubervisorService,
			DeleteFunc: ctrl.onDeleteKubervisorService,
		},
	)

	podInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddPod,
			UpdateFunc: ctrl.onUpdatePod,
			DeleteFunc: ctrl.onDeletePod,
		},
	)

	serviceInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddService,
			UpdateFunc: ctrl.onUpdateService,
			DeleteFunc: ctrl.onDeleteService,
		},
	)

	ctrl.gc, err = newGarbageCollector(time.Second, ctrl.podControl, ctrl.podLister, ctrl.breakerLister, 2, ctrl.Logger)
	if err != nil {
		sugar.Fatalf("Unable to initialize garbage collector: %v", err)
	}

	return ctrl
}

// Run start the Controller
func (ctrl *Controller) Run(stop <-chan struct{}) error {
	defer ctrl.rootContextCancelFunc()
	var err error
	ctrl.kubeInformerFactory.Start(stop)
	ctrl.breakerInformerFactory.Start(stop)
	go ctrl.runHTTPServer(stop)
	go ctrl.gc.run(stop)
	err = ctrl.run(stop)

	return err
}

func (ctrl *Controller) run(stop <-chan struct{}) error {
	if !cache.WaitForCacheSync(stop, ctrl.PodSynced) {
		return fmt.Errorf("Timed out waiting for caches to sync")
	}

	for i := uint32(0); i < ctrl.nbWorker; i++ {
		go wait.Until(ctrl.runWorker, time.Second, stop)
	}

	<-stop
	return nil
}

func (ctrl *Controller) runWorker() {
	for ctrl.processNextItem() {
	}
}

func (ctrl *Controller) processNextItem() bool {
	key, quit := ctrl.queue.Get()
	if quit {
		return false
	}
	defer ctrl.queue.Done(key)
	needRequeue, err := ctrl.sync(key.(string))
	if err == nil {
		ctrl.queue.Forget(key)
	} else {
		utilruntime.HandleError(fmt.Errorf("Error syncing kubervisorservice: %v", err))
		ctrl.queue.AddRateLimited(key)
		return true
	}

	if needRequeue {
		ctrl.Logger.Sugar().Info("processNextItem: Requeue key:", key)
		ctrl.queue.AddRateLimited(key)
	}

	return true
}

func (ctrl *Controller) sync(key string) (bool, error) {
	ctrl.Logger.Sugar().Debugf("sync() key: %s", key)
	startTime := time.Now()
	defer func() {
		ctrl.Logger.Sugar().Debugf("Finished syncing KubervisorService %q in %v", key, time.Since(startTime))
		// TODO add prometheus metric here
	}()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	ctrl.Logger.Sugar().Debugf("Syncing KubervisorService %s/%s", namespace, name)
	sharedKubervisorService, err := ctrl.breakerLister.KubervisorServices(namespace).Get(name)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to get KubervisorService %s/%s: %v. Maybe deleted", namespace, name, err)
		return false, nil
	}

	if !kubervisorapi.IsKubervisorServiceDefaulted(sharedKubervisorService) {
		ctrl.Logger.Sugar().Debugf("KubervisorService IsKubervisorServiceDefaulted return false for:%s/%s", namespace, name)
		defaultedKubervisorService := kubervisorapi.DefaultKubervisorService(sharedKubervisorService)
		if _, err = ctrl.updateHandlerFunc(defaultedKubervisorService); err != nil {
			ctrl.Logger.Sugar().Errorf("unable to default KubervisorService %s/%s, error:%v", namespace, name, err)
			return false, fmt.Errorf("unable to default KubervisorService %s/%s, error:%s", namespace, name, err)
		}
		ctrl.Logger.Sugar().Debugf("KubervisorService %s/%s defaulted", namespace, name)
		return false, nil
	}

	// TODO add validation

	// TODO: add test the case of graceful deletion
	if sharedKubervisorService.DeletionTimestamp != nil {
		return false, nil
	}

	bc := sharedKubervisorService.DeepCopy()
	retValue, errSync := ctrl.syncKubervisorService(bc)
	if errSync != nil {
		ctrl.Logger.Sugar().Debugf("syncKubervisorService return error: %s", errSync)
	}
	return retValue, errSync
}

func (ctrl *Controller) syncKubervisorService(bc *kubervisorapi.KubervisorService) (bool, error) {
	key := fmt.Sprintf("%s/%s", bc.Namespace, bc.Name)
	ctrl.Logger.Sugar().Debugf("syncKubervisorService %s", key)
	now := metav1.Now()
	// Init status.StartTime
	if bc.Status.StartTime == nil {
		bc.Status.StartTime = &now
		if _, err := ctrl.updateHandlerFunc(bc); err != nil {
			ctrl.Logger.Sugar().Errorf("BreakerService %s/%s: unable init startTime: %v", bc.Namespace, bc.Name, err)
			return false, err
		}
		ctrl.Logger.Sugar().Infof("BreakerService %s/%s: startTime updated", bc.Namespace, bc.Name)
		return true, nil
	}
	ctrl.Logger.Sugar().Debugf("BreakerService %s/%s: startTime already setted", bc.Namespace, bc.Name)

	associatedSvc, err := ctrl.serviceLister.Services(bc.Namespace).Get(bc.Spec.Service)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		msg := fmt.Sprintf("associated service %s/%s didn't exist, error:", bc.Namespace, bc.Spec.Service)
		newStatus, err2 := UpdateStatusConditionServiceError(&bc.Status, msg, now)
		if err2 != nil {
			return false, err2
		}
		bc.Status = *newStatus
		ctrl.Logger.Sugar().Errorf(msg)
		if _, err2 = ctrl.updateHandlerFunc(bc); err2 != nil {
			return false, err2
		}
		return false, err
	}
	obj, exist, err := ctrl.items.GetByKey(key)
	if err != nil {
		return false, err
	}

	var bci item.Interface
	if !exist {
		ctrl.Logger.Sugar().Debugf("item not found for key:%s", key)
		if bci, err = ctrl.createItem(bc, associatedSvc, now); err != nil {
			return false, err
		}
		ctrl.items.Add(bci)
	} else {
		var ok bool
		bci, ok = obj.(item.Interface)
		if !ok {
			return false, fmt.Errorf("unable to case the obj to a KubervisorServiceItem")
		}
		if IsSpecUpdated(bc, associatedSvc, bci) {
			bci.Stop()
			if bci, err = ctrl.createItem(bc, associatedSvc, now); err != nil {
				return false, err
			}
			ctrl.items.Update(bci)
		}
	}

	// check if some pods have been removed from the service selector
	// if it is the case, removed all labels and annotation
	if _, err = ctrl.podsCleaner(bci.Name(), associatedSvc); err != nil {
		ctrl.Logger.Sugar().Errorf("podsCleaner failed: %v", err)
		return false, err
	}

	// initialize possible new pods (add labels)
	if _, err = ctrl.initializePods(bci.Name(), associatedSvc); err != nil {
		ctrl.Logger.Sugar().Errorf("initializePods failed: %v", err)
		return false, err
	}

	newStatus := bci.GetStatus()
	updateGauge(bci.Name(), newStatus)
	if bc.Status.Breaker == nil || !equalBreakerStatusCounters(newStatus, *bc.Status.Breaker) {
		bc.Status.Breaker = &newStatus
		//update status to running
		if newStatus, err := UpdateStatusConditionRunning(&bc.Status, "", now); err == nil {
			bc.Status = *newStatus
		}

		ctrl.Logger.Sugar().Debugf("BreakerService %s/%s: breaker status updated", bc.Namespace, bc.Name)
		if _, err := ctrl.updateHandlerFunc(bc); err != nil {
			return false, err
		}
	}
	return false, nil
}

func updateGauge(name string, status kubervisorapi.BreakerStatus) {
	kubervisorGauges.WithLabelValues(name, "managed").Set(float64(status.NbPodsManaged))
	kubervisorGauges.WithLabelValues(name, "breaked").Set(float64(status.NbPodsBreaked))
	kubervisorGauges.WithLabelValues(name, "paused").Set(float64(status.NbPodsPaused))
	kubervisorGauges.WithLabelValues(name, "unknown").Set(float64(status.NbPodsUnknown))
}

// Used to select all currently associated to the KubervisorService and check if it is still the case
// if they are not manage anymore by the current service label selector, this function remove the added labels.
func (ctrl *Controller) podsCleaner(bciName string, svc *apiv1.Service) (bool, error) {
	selectorSet := labels.Set{labeling.LabelBreakerNameKey: bciName}
	delete(selectorSet, labeling.LabelTrafficKey)
	previousPods, err := ctrl.podLister.List(selectorSet.AsSelectorPreValidated())
	if err != nil {
		return false, err
	}

	svcSelector := svc.DeepCopy().Spec.Selector
	delete(svcSelector, labeling.LabelTrafficKey)
	currentPods, err := ctrl.podLister.List(labels.Set(svcSelector).AsSelectorPreValidated())
	if err != nil {
		return false, err
	}

	discaredPods := pod.ExcludeFromSlice(previousPods, currentPods)
	errs := []error{}
	activity := len(discaredPods) != 0
	for _, pod := range discaredPods {
		if _, err := ctrl.podControl.RemoveBreakerAnnotationAndLabel(pod); err != nil {
			errs = append(errs, err)
		}
	}

	return activity, errors.NewAggregate(errs)
}

func (ctrl *Controller) initializePods(bciName string, svc *apiv1.Service) (bool, error) {
	ctrl.Logger.Sugar().Debugf("initializePods for %s on service %s", bciName, svc.Name)
	pods, err := ctrl.searchNewPods(svc)
	if err != nil {
		return false, err
	}
	if len(pods) == 0 {
		return false, nil
	}
	var errs []error
	for _, p := range pods {
		if _, err := ctrl.podControl.InitBreakerAnnotationAndLabel(bciName, p); err != nil {
			errs = append(errs, err)
		}

	}

	return true, errors.NewAggregate(errs)
}

func (ctrl *Controller) searchNewPods(svc *apiv1.Service) ([]*apiv1.Pod, error) {
	podSelector := svc.DeepCopy().Spec.Selector
	// remove LabelTraffic if it is already present
	delete(podSelector, labeling.LabelTrafficKey)
	pods, err := ctrl.kubeClient.Core().Pods(svc.Namespace).List(metav1.ListOptions{LabelSelector: labels.Set(podSelector).AsSelector().String()})
	if err != nil {
		return nil, err
	}
	outPods := []*apiv1.Pod{}
	for _, p := range pods.Items {
		_, ok1 := p.Labels[labeling.LabelTrafficKey]
		_, ok2 := p.Labels[labeling.LabelBreakerNameKey]
		if ok1 && ok2 {
			continue
		}
		outPods = append(outPods, p.DeepCopy())
	}

	return outPods, nil
}

func (ctrl *Controller) createItem(bc *kubervisorapi.KubervisorService, associatedSvc *apiv1.Service, now metav1.Time) (item.Interface, error) {
	bci, err := ctrl.newKubervisorServiceItem(bc, associatedSvc)
	if err != nil {
		msg := fmt.Sprintf("unable to create KubervisorServiceItem, err:%v", err)
		ctrl.Logger.Sugar().Debugf(msg)
		newStatus, err2 := UpdateStatusConditionInitFailure(&bc.Status, msg, now)
		if err2 != nil {
			return nil, err2
		}
		bc.Status = *newStatus
		ctrl.Logger.Sugar().Errorf(msg)
		if _, err2 = ctrl.updateHandlerFunc(bc); err2 != nil {
			return nil, err2
		}
		return nil, err
	}

	//update status to running
	if newStatus, err := UpdateStatusConditionRunning(&bc.Status, "", now); err == nil {
		bc.Status = *newStatus
		ctrl.updateHandlerFunc(bc)
	}

	return bci, nil
}

func (ctrl *Controller) newKubervisorServiceItem(bc *kubervisorapi.KubervisorService, svc *apiv1.Service) (item.Interface, error) {
	//Purge the service selector from kubervisor traffic labels
	selectorWithoutTrafficKey := labels.NewSelector()
	requirements, _ := labels.Set(svc.Spec.Selector).AsSelectorPreValidated().Requirements()
	for _, r := range requirements {
		if r.Key() != labeling.LabelTrafficKey {
			selectorWithoutTrafficKey.Add(r)
		}
	}

	itemConfig := &item.Config{
		Logger:     ctrl.Logger,
		Selector:   selectorWithoutTrafficKey,
		PodLister:  ctrl.podLister,
		PodControl: ctrl.podControl,
	}
	bci, err := item.New(bc, itemConfig)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to create new KubervisorServiceItem, err:%v", err)
		return nil, err
	}
	bci.Start(ctrl.rootContext)
	return bci, nil
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(c.Master, c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

func (ctrl *Controller) deleteKubervisorService(ns, name string) error {
	return ctrl.breakerClient.Breaker().KubervisorServices(ns).Delete(name, &metav1.DeleteOptions{})
}

func (ctrl *Controller) updateHandler(bc *kubervisorapi.KubervisorService) (*kubervisorapi.KubervisorService, error) {
	return ctrl.breakerClient.Breaker().KubervisorServices(bc.Namespace).Update(bc)
}

// enqueue adds key in the controller queue
func (ctrl *Controller) enqueue(bc *kubervisorapi.KubervisorService) {
	key, err := cache.MetaNamespaceKeyFunc(bc)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Controller:enqueue: couldn't get key for KubervisorService %s/%s: %v", bc.Namespace, bc.Name, err)
		return
	}
	ctrl.queue.Add(key)
}
