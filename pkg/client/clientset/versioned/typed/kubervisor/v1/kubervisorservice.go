/*
MIT License

Copyright (c) 2018 PodKubervisor

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	v1 "github.com/amadeusitgroup/podkubervisor/pkg/api/kubervisor/v1"
	scheme "github.com/amadeusitgroup/podkubervisor/pkg/client/clientset/versioned/scheme"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// KubervisorServicesGetter has a method to return a KubervisorServiceInterface.
// A group's client should implement this interface.
type KubervisorServicesGetter interface {
	KubervisorServices(namespace string) KubervisorServiceInterface
}

// KubervisorServiceInterface has methods to work with KubervisorService resources.
type KubervisorServiceInterface interface {
	Create(*v1.KubervisorService) (*v1.KubervisorService, error)
	Update(*v1.KubervisorService) (*v1.KubervisorService, error)
	Delete(name string, options *meta_v1.DeleteOptions) error
	DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error
	Get(name string, options meta_v1.GetOptions) (*v1.KubervisorService, error)
	List(opts meta_v1.ListOptions) (*v1.KubervisorServiceList, error)
	Watch(opts meta_v1.ListOptions) (watch.Interface, error)
	Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.KubervisorService, err error)
	KubervisorServiceExpansion
}

// kubervisorServices implements KubervisorServiceInterface
type kubervisorServices struct {
	client rest.Interface
	ns     string
}

// newKubervisorServices returns a KubervisorServices
func newKubervisorServices(c *KubervisorV1Client, namespace string) *kubervisorServices {
	return &kubervisorServices{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the kubervisorService, and returns the corresponding kubervisorService object, and an error if there is any.
func (c *kubervisorServices) Get(name string, options meta_v1.GetOptions) (result *v1.KubervisorService, err error) {
	result = &v1.KubervisorService{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kubervisorservices").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of KubervisorServices that match those selectors.
func (c *kubervisorServices) List(opts meta_v1.ListOptions) (result *v1.KubervisorServiceList, err error) {
	result = &v1.KubervisorServiceList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("kubervisorservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Do().
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested kubervisorServices.
func (c *kubervisorServices) Watch(opts meta_v1.ListOptions) (watch.Interface, error) {
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("kubervisorservices").
		VersionedParams(&opts, scheme.ParameterCodec).
		Watch()
}

// Create takes the representation of a kubervisorService and creates it.  Returns the server's representation of the kubervisorService, and an error, if there is any.
func (c *kubervisorServices) Create(kubervisorService *v1.KubervisorService) (result *v1.KubervisorService, err error) {
	result = &v1.KubervisorService{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("kubervisorservices").
		Body(kubervisorService).
		Do().
		Into(result)
	return
}

// Update takes the representation of a kubervisorService and updates it. Returns the server's representation of the kubervisorService, and an error, if there is any.
func (c *kubervisorServices) Update(kubervisorService *v1.KubervisorService) (result *v1.KubervisorService, err error) {
	result = &v1.KubervisorService{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("kubervisorservices").
		Name(kubervisorService.Name).
		Body(kubervisorService).
		Do().
		Into(result)
	return
}

// Delete takes name of the kubervisorService and deletes it. Returns an error if one occurs.
func (c *kubervisorServices) Delete(name string, options *meta_v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kubervisorservices").
		Name(name).
		Body(options).
		Do().
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *kubervisorServices) DeleteCollection(options *meta_v1.DeleteOptions, listOptions meta_v1.ListOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("kubervisorservices").
		VersionedParams(&listOptions, scheme.ParameterCodec).
		Body(options).
		Do().
		Error()
}

// Patch applies the patch and returns the patched kubervisorService.
func (c *kubervisorServices) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *v1.KubervisorService, err error) {
	result = &v1.KubervisorService{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("kubervisorservices").
		SubResource(subresources...).
		Name(name).
		Body(data).
		Do().
		Into(result)
	return
}
