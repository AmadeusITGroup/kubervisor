package controller

import (
	"fmt"
	"net/http"
	"time"

	"github.com/heptiolabs/healthcheck"
	"go.uber.org/zap"

	apiv1 "k8s.io/api/core/v1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	kubervisorclient "github.com/amadeusitgroup/podkubervisor/pkg/client"
	bclient "github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned"
	binformers "github.com/amadeusitgroup/podkubervisor/pkg/client/informers/externalversions"
	blisters "github.com/amadeusitgroup/podkubervisor/pkg/client/listers/kubervisor/v1"
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

	queue workqueue.RateLimitingInterface // BreakerConfigs to be synced

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
	breakerInformer := breakerInformerFactory.Breaker().V1().BreakerConfigs()

	ctrl := &Controller{
		nbWorker: cfg.NbWorker,
		Logger:   cfg.Logger,

		kubeInformerFactory:    kubeInformerFactory,
		breakerInformerFactory: breakerInformerFactory,
		podLister:              podInformer.Lister(),
		PodSynced:              podInformer.Informer().HasSynced,
		breakerLister:          breakerInformer.Lister(),
		BreakerSynced:          breakerInformer.Informer().HasSynced,

		queue:    workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "breakerconfig"),
		recorder: eventBroadcaster.NewRecorder(scheme.Scheme, apiv1.EventSource{Component: "kubervisor-controller"}),
	}
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

	return ctrl
}

// Run start the Controller
func (ctrl *Controller) Run(stop <-chan struct{}) error {
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
	/*
		for c.processNextItem() {
		}
	*/
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

func (c *Controller) sync(key string) (bool, error) {
	return true, nil
}

func initKubeConfig(c *Config) (*rest.Config, error) {
	if len(c.KubeConfigFile) > 0 {
		return clientcmd.BuildConfigFromFlags(c.Master, c.KubeConfigFile) // out of cluster config
	}
	return rest.InClusterConfig()
}
