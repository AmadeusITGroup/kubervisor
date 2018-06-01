package client

import (
	"reflect"
	"time"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"

	"github.com/golang/glog"

	kubervisor "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor"
	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	"github.com/amadeusitgroup/kubervisor/pkg/client/clientset/versioned"
)

// DefineKubervisorResources defines the  DefineKubervisor Resources as a k8s CR
func DefineKubervisorResources(clientset apiextensionsclient.Interface) (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	kubervisorClusterResourceName := api.ResourcePlural + "." + kubervisor.GroupName
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: kubervisorClusterResourceName,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   kubervisor.GroupName,
			Version: api.SchemeGroupVersion.Version,
			Scope:   apiextensionsv1beta1.NamespaceScoped,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural:     api.ResourcePlural,
				Singular:   api.ResourceSingular,
				Kind:       reflect.TypeOf(api.KubervisorService{}).Name(),
				ShortNames: []string{"rdc"},
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		return nil, err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(kubervisorClusterResourceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(kubervisorClusterResourceName, nil)
		if deleteErr != nil {
			return nil, errors.NewAggregate([]error{err, deleteErr})
		}
		return nil, err
	}

	return crd, nil
}

// NewClient builds and initializes a Client and a Scheme for RedisCluster CR
func NewClient(cfg *rest.Config) (versioned.Interface, error) {
	scheme := runtime.NewScheme()
	if err := api.AddToScheme(scheme); err != nil {
		return nil, err
	}

	config := *cfg
	config.GroupVersion = &api.SchemeGroupVersion
	config.APIPath = "/apis"
	config.ContentType = runtime.ContentTypeJSON
	config.NegotiatedSerializer = serializer.DirectCodecFactory{CodecFactory: serializer.NewCodecFactory(scheme)}

	cs, err := versioned.NewForConfig(&config)
	if err != nil {
		return nil, err
	}
	return cs, nil
}
