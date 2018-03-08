package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
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

// Controller represent the kubervisor controller
type Controller struct {
	nbWorker uint32
	Logger   *zap.Logger

	recorder record.EventRecorder

	kubeInformerFactory    kubeinformers.SharedInformerFactory
	breakerInformerFactory binformers.SharedInformerFactory

	kubeClient    clientset.Interface
	breakerClient bclient.Interface

	breakerLister blisters.BreakerConfigLister
	BreakerSynced cache.InformerSynced

	podLister corev1listers.PodLister
	PodSynced cache.InformerSynced

	serviceLister corev1listers.ServiceLister
	ServiceSynced cache.InformerSynced

	queue       workqueue.RateLimitingInterface // BreakerConfigs to be synced
	enqueueFunc func(bc *kubervisorapi.BreakerConfig)

	items                 item.BreakerConfigItemStore
	updateHandler         func(*kubervisorapi.BreakerConfig) error
	podControl            pod.ControlInterface
	rootContext           context.Context
	rootContextCancelFunc context.CancelFunc

	// Kubernetes Probes handler
	health healthcheck.Handler

	httpServer *http.Server
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
		sugar.Fatalf("Unable to define BreakerConfig resource:%v", err)
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
	breakerInformer := breakerInformerFactory.Breaker().V1().BreakerConfigs()

	ctx, ctxCancel := context.WithCancel(context.Background())

	ctrl := &Controller{
		nbWorker: cfg.NbWorker,
		Logger:   cfg.Logger,

		kubeInformerFactory:    kubeInformerFactory,
		breakerInformerFactory: breakerInformerFactory,
		podLister:              podInformer.Lister(),
		PodSynced:              podInformer.Informer().HasSynced,
		serviceLister:          serviceInformer.Lister(),
		ServiceSynced:          serviceInformer.Informer().HasSynced,
		breakerLister:          breakerInformer.Lister(),
		BreakerSynced:          breakerInformer.Informer().HasSynced,

		podControl:            pod.NewPodControl(kubeClient),
		rootContext:           ctx,
		rootContextCancelFunc: ctxCancel,

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "breakerconfig"),
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "kubervisor-controller"}),
	}
	ctrl.enqueueFunc = ctrl.enqueue

	breakerInformer.Informer().AddEventHandler(
		cache.ResourceEventHandlerFuncs{
			AddFunc:    ctrl.onAddBreakerConfig,
			UpdateFunc: ctrl.onUpdateBreakerConfig,
			DeleteFunc: ctrl.onDeleteBreakerConfig,
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

	return ctrl
}

// Run start the Controller
func (ctrl *Controller) Run(stop <-chan struct{}) error {
	defer ctrl.rootContextCancelFunc()
	var err error
	ctrl.kubeInformerFactory.Start(stop)
	ctrl.breakerInformerFactory.Start(stop)
	go ctrl.runHTTPServer(stop)
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
		utilruntime.HandleError(fmt.Errorf("Error syncing rediscluster: %v", err))
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
	ctrl.Logger.Sugar().Debug("sync() key: %s, key")
	startTime := time.Now()
	defer func() {
		ctrl.Logger.Sugar().Debug("Finished syncing BreakerConfig %q in %v", key, time.Since(startTime))
		// TODO add prometheus metric here
	}()
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return false, err
	}
	ctrl.Logger.Sugar().Debug("Syncing BreakerConfig %s/%s", namespace, name)
	sharedBreakerConfig, err := ctrl.breakerLister.BreakerConfigs(namespace).Get(name)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to get BreakerConfig %s/%s: %v. Maybe deleted", namespace, name, err)
		return false, nil
	}
	if !kubervisorapi.IsBreakerConfigDefaulted(sharedBreakerConfig) {
		defaultedBreakerConfig := kubervisorapi.DefaultBreakerConfig(sharedBreakerConfig)
		if err = ctrl.updateHandler(defaultedBreakerConfig); err != nil {
			ctrl.Logger.Sugar().Errorf("unable to default BreakerConfig %s/%s, error:%v", namespace, name, err)
			return false, fmt.Errorf("unable to default BreakerConfig %s/%s, error:%s", namespace, name, err)
		}
		ctrl.Logger.Sugar().Debugf("BreakerConfig %s/%s defaulted", namespace, name)
		return false, nil
	}

	// TODO add validation

	// TODO: add test the case of graceful deletion
	if sharedBreakerConfig.DeletionTimestamp != nil {
		return false, nil
	}

	bc := sharedBreakerConfig.DeepCopy()
	return ctrl.syncBreakerConfig(bc)
}

func (ctrl *Controller) syncBreakerConfig(bc *kubervisorapi.BreakerConfig) (bool, error) {
	key := fmt.Sprintf("%s/%s", bc.Namespace, bc.Name)

	associatedSvc, err := ctrl.serviceLister.Services(bc.Namespace).Get(bc.Spec.Service)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			ctrl.Logger.Sugar().Errorf("associated service %s/%s didn't exist, error:", bc.Namespace, bc.Spec.Service)
			return false, err
		}
		return false, err
	}
	obj, exist, err := ctrl.items.GetByKey(key)
	if err != nil {
		return false, err
	}

	var bci item.Interface
	if !exist {
		ctrl.Logger.Sugar().Debugw("item not found for key:%s", key)
		if bci, err = ctrl.newBreakerConfigItem(bc, associatedSvc); err != nil {
			return false, err
		}
		ctrl.items.Add(bci)
	} else {
		var ok bool
		bci, ok = obj.(item.Interface)
		if !ok {
			return false, fmt.Errorf("unable to case the obj to a BreakerConfigItem")
		}
		if IsSpecUpdated(bc, associatedSvc, bci) {
			bci.Stop()
			if bci, err = ctrl.newBreakerConfigItem(bc, associatedSvc); err != nil {
				return false, err
			}
			ctrl.items.Update(bci)
		}
	}

	globalActivity := false
	// check if some pods have been removed from the service selector
	// if it is the case, removed all labels and annotation
	activity, err := ctrl.podsCleaner(bci.Name(), associatedSvc)
	if err != nil {
		return activity, err
	}
	if activity {
		globalActivity = true
	}

	// initialize possible new pods (add labels)
	activity, err = ctrl.initializePods(bci.Name(), associatedSvc)
	if err != nil {
		return activity, err
	}
	if activity {
		globalActivity = true
	}

	return globalActivity, nil
}

// Used to select all currently associated to the BreakerConfig and check if it is still the case
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

func (ctrl *Controller) newBreakerConfigItem(bc *kubervisorapi.BreakerConfig, svc *apiv1.Service) (item.Interface, error) {
	itemConfig := &item.Config{
		Logger:     ctrl.Logger,
		Selector:   labels.Set(svc.Spec.Selector).AsSelectorPreValidated(),
		PodLister:  ctrl.podLister,
		PodControl: ctrl.podControl,
	}
	bci, err := item.New(bc, itemConfig)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("unable to create new BreakerConfigItem, err:%v", err)
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

func (ctrl *Controller) deleteBreakerConfig(ns, name string) error {
	return ctrl.breakerClient.Breaker().BreakerConfigs(ns).Delete(name, &metav1.DeleteOptions{})
}

// enqueue adds key in the controller queue
func (ctrl *Controller) enqueue(bc *kubervisorapi.BreakerConfig) {
	key, err := cache.MetaNamespaceKeyFunc(bc)
	if err != nil {
		ctrl.Logger.Sugar().Errorf("Controller:enqueue: couldn't get key for BreakerConfig %s/%s: %v", bc.Namespace, bc.Name, err)
		return
	}
	ctrl.queue.Add(key)
}
