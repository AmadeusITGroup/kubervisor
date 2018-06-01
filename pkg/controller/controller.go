package controller

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/heptiolabs/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	apiv1 "k8s.io/api/core/v1"
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

	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	bclient "github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
	binformers "github.com/amadeusitgroup/kubervisor/pkg/client/informers/externalversions"
	"github.com/amadeusitgroup/kubervisor/pkg/client/informers/externalversions/kubervisor/v1alpha1"
	blisters "github.com/amadeusitgroup/kubervisor/pkg/client/listers/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/controller/item"
	"github.com/amadeusitgroup/kubervisor/pkg/labeling"
	election "github.com/amadeusitgroup/kubervisor/pkg/leaderelection"
	"github.com/amadeusitgroup/kubervisor/pkg/pod"
)

func init() {
	prometheus.MustRegister(kubervisorGauges)
}

var (
	// leader election config
	leaseDuration = 15 * time.Second
	renewDuration = 5 * time.Second
	retryPeriod   = 3 * time.Second
)

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

	locker *resourcelock.EndpointsLock

	kubeInformerFactory    kubeinformers.SharedInformerFactory
	breakerInformerFactory binformers.SharedInformerFactory
	breakerInformer        v1alpha1.KubervisorServiceInformer
	kubeClient             clientset.Interface
	breakerClient          bclient.Interface

	breakerLister blisters.KubervisorServiceLister
	BreakerSynced cache.InformerSynced

	podLister corev1listers.PodLister
	PodSynced cache.InformerSynced

	serviceLister corev1listers.ServiceLister
	ServiceSynced cache.InformerSynced

	queue       workqueue.RateLimitingInterface // KubervisorServices to be synced
	enqueueFunc func(bc *api.KubervisorService)

	items                 item.KubervisorServiceItemStore
	updateHandlerFunc     func(*api.KubervisorService) (*api.KubervisorService, error)
	podControl            pod.ControlInterface
	rootContext           context.Context
	rootContextCancelFunc context.CancelFunc

	// Kubernetes Probes handler
	health healthcheck.Handler

	httpServer *http.Server

	gc *garbageCollector
}

//Initializer prepare/return all dependencies for controller creation
type Initializer interface {
	InitClients() (clientset.Interface, bclient.Interface, clientset.Interface, error)
	RegisterAPI() error
	Logger() *zap.Logger
	NbWorker() uint32
	HTTPServer() *http.Server
}

// New returns new Controller instance
func New(initializer Initializer) *Controller {
	sugar := initializer.Logger().Sugar()
	if err := initializer.RegisterAPI(); err != nil {
		return nil
	}
	kubeClient, breakerClient, leaderElectionClient, err := initializer.InitClients()
	if err != nil {
		return nil
	}

	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)
	breakerInformerFactory := binformers.NewSharedInformerFactory(breakerClient, time.Second*30)

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(sugar.Infof)
	eventBroadcaster.StartRecordingToSink(&v1core.EventSinkImpl{Interface: v1core.New(kubeClient.Core().RESTClient()).Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "kubervisor-controller"})

	podInformer := kubeInformerFactory.Core().V1().Pods()
	serviceInformer := kubeInformerFactory.Core().V1().Services()
	breakerInformer := breakerInformerFactory.Kubervisor().V1alpha1().KubervisorServices()

	id, err := os.Hostname()
	if err != nil {
		sugar.Fatalf("Failed to get hostname: %v", err)
	}

	var lockLeader *resourcelock.EndpointsLock
	if leaderElectionClient != nil {
		lockLeader = &resourcelock.EndpointsLock{
			EndpointsMeta: metav1.ObjectMeta{
				Namespace: "default", // TODO retrieve current Namespaces
				Name:      "kubervisor",
			},
			Client: leaderElectionClient.CoreV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity:      id,
				EventRecorder: recorder,
			},
		}
	}

	ctx, ctxCancel := context.WithCancel(context.Background())

	ctrl := &Controller{
		nbWorker: initializer.NbWorker(),
		Logger:   initializer.Logger(),

		kubeInformerFactory:    kubeInformerFactory,
		kubeClient:             kubeClient,
		breakerInformerFactory: breakerInformerFactory,
		breakerInformer:        breakerInformer,
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

		httpServer: initializer.HTTPServer(),

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "kubervisorservice"),
		recorder: recorder,
		locker:   lockLeader,
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
	ctrl.kubeInformerFactory.Start(stop)
	ctrl.breakerInformerFactory.Start(stop)
	go ctrl.runHTTPServer(stop)
	go ctrl.gc.run(stop)

	// Simple run if no leader election
	if ctrl.locker == nil {
		return ctrl.run(stop)
	}

	// Start leader election.
	return election.Run(election.Config{
		Lock:          ctrl.locker,
		LeaseDuration: leaseDuration,
		RenewDeadline: renewDuration,
		RetryPeriod:   retryPeriod,
		Callbacks: election.LeaderCallbacks{
			OnStartedLeading: ctrl.run,
			OnStoppedLeading: func() error {
				ctrl.Logger.Sugar().Errorf("leader election lost")
				return nil
			},
		},
		Logger: ctrl.Logger,
	}, stop)
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

	if !api.IsKubervisorServiceDefaulted(sharedKubervisorService) {
		ctrl.Logger.Sugar().Debugf("KubervisorService IsKubervisorServiceDefaulted return false for:%s/%s", namespace, name)
		defaultedKubervisorService := api.DefaultKubervisorService(sharedKubervisorService)
		if _, err = ctrl.updateHandlerFunc(defaultedKubervisorService); err != nil {
			ctrl.Logger.Sugar().Errorf("unable to default KubervisorService %s/%s, error:%v", namespace, name, err)
			return false, fmt.Errorf("unable to default KubervisorService %s/%s, error:%s", namespace, name, err)
		}
		ctrl.Logger.Sugar().Debugf("KubervisorService %s/%s defaulted", namespace, name)
		return false, nil
	}

	if err := api.ValidateKubervisorServiceSpec(sharedKubervisorService.Spec); err != nil {
		return false, fmt.Errorf("Invalid KubervisorService definition: %v", err)
	}

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

func (ctrl *Controller) syncKubervisorService(bc *api.KubervisorService) (bool, error) {
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
	ctrl.Logger.Sugar().Debugf("BreakerService %s/%s: startTime already set", bc.Namespace, bc.Name)

	var bci item.Interface
	obj, exist, err := ctrl.items.GetByKey(key)
	if err != nil {
		return false, err
	}
	if exist {
		var ok bool
		bci, ok = obj.(item.Interface)
		if !ok {
			return false, fmt.Errorf("unable to case the obj to a KubervisorServiceItem")
		}
	}

	associatedSvc, err := ctrl.serviceLister.Services(bc.Namespace).Get(bc.Spec.Service)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return false, err
		}
		var msg string
		if exist {
			bci.Stop()
			if err2 := ctrl.items.Delete(bci); err != nil {
				return false, err2
			}
			msg = fmt.Sprintf("associated service %s/%s was deleted", bc.Namespace, bc.Spec.Service)
			ctrl.Logger.Sugar().Errorf(msg)
		} else {
			msg = fmt.Sprintf("associated service %s/%s doesn't exist, error:", bc.Namespace, bc.Spec.Service)
			ctrl.Logger.Sugar().Errorf(msg)
		}

		if err2 := ctrl.updateStatusCondition(bc, UpdateStatusConditionServiceError, msg, now); err != nil {
			return false, err2
		}
		return false, nil
	}

	if !exist {
		ctrl.Logger.Sugar().Debugf("item not found for key:%s", key)
		if bci, err = ctrl.createItem(bc, associatedSvc, now); err != nil {
			return false, err
		}
		ctrl.items.Add(bci)
		bci.Start(ctrl.rootContext)
	} else {
		if IsSpecUpdated(bc, associatedSvc, bci) {
			bci.Stop()
			if bci, err = ctrl.createItem(bc, associatedSvc, now); err != nil {
				return false, err
			}
			ctrl.items.Update(bci)
			bci.Start(ctrl.rootContext)
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
	if bc.Status.PodCounts == nil || !equalPodCountStatus(newStatus, *bc.Status.PodCounts) {
		bc.Status.PodCounts = &newStatus
		//update status to running
		ctrl.updateStatusCondition(bc, UpdateStatusConditionRunning, "", now)
	}
	return false, nil
}

func updateGauge(name string, status api.PodCountStatus) {
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

func (ctrl *Controller) updateStatusCondition(bc *api.KubervisorService, statusUdapteFct statusUpdateFunc, msg string, now metav1.Time) error {
	ctrl.Logger.Sugar().Debugf(msg)
	newStatus, err2 := statusUdapteFct(&bc.Status, msg, now)
	if err2 != nil {
		ctrl.Logger.Sugar().Errorf("Unable to update status for CRD %s/%s", bc.Namespace, bc.Name)
		return err2
	}
	bc.Status = *newStatus
	if _, err2 = ctrl.updateHandlerFunc(bc); err2 != nil {
		ctrl.Logger.Sugar().Errorf("Unable to update status for CRD %s/%s", bc.Namespace, bc.Name)
		return err2
	}
	return nil
}

func (ctrl *Controller) createItem(bc *api.KubervisorService, associatedSvc *apiv1.Service, now metav1.Time) (item.Interface, error) {
	bci, err := ctrl.newKubervisorServiceItem(bc, associatedSvc)
	if err != nil {
		ctrl.updateStatusCondition(bc, UpdateStatusConditionInitFailure, fmt.Sprintf("unable to create KubervisorServiceItem, err:%v", err), now)
		return nil, err
	}

	//update status to running
	ctrl.updateStatusCondition(bc, UpdateStatusConditionRunning, "", now)

	return bci, nil
}

func (ctrl *Controller) newKubervisorServiceItem(bc *api.KubervisorService, svc *apiv1.Service) (item.Interface, error) {
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
	return bci, nil
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(c.Master, c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}

func (ctrl *Controller) deleteKubervisorService(ns, name string) error {
	return ctrl.breakerClient.Kubervisor().KubervisorServices(ns).Delete(name, &metav1.DeleteOptions{})
}

func (ctrl *Controller) updateHandler(bc *api.KubervisorService) (*api.KubervisorService, error) {
	return ctrl.breakerClient.Kubervisor().KubervisorServices(bc.Namespace).Update(bc)
}

// enqueue adds key in the controller queue
func (ctrl *Controller) enqueue(bc *api.KubervisorService) {
	key, err := cache.MetaNamespaceKeyFunc(bc)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Controller:enqueue: couldn't get key for KubervisorService %s/%s: %v", bc.Namespace, bc.Name, err)
		return
	}
	ctrl.queue.Add(key)
}
