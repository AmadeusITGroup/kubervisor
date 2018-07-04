package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"

	kapiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	kvclient "github.com/amadeusitgroup/kubervisor/pkg/client"
)

func main() {
	namespace := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE")

	kubervisorServicesName := ""
	if val := os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_KS"); val != "" {
		kubervisorServicesName = val
	}

	kubeconfigFilePath := getKubeConfigDefaultPath(getHomePath())
	if len(kubeconfigFilePath) == 0 {
		log.Fatal("error initializing config. The KUBECONFIG environment variable must be defined.")
	}

	config, err := configFromPath(kubeconfigFilePath)
	if err != nil {
		log.Fatalf("error obtaining kubectl config: %v", err)
	}

	rest, err := config.ClientConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}

	kubervisorClient, err := kvclient.NewClient(rest)
	if err != nil {
		glog.Fatalf("Unable to init kubervisor.clientset from kubeconfig:%v", err)
	}

	kvs := &api.KubervisorServiceList{}
	if kubervisorServicesName == "" {
		kvs, err = kubervisorClient.KubervisorV1alpha1().KubervisorServices(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			fmt.Printf("unable to list kubervisorservice err:%v\n", err)
			return
		}
	} else {
		ks, err := kubervisorClient.KubervisorV1alpha1().KubervisorServices(namespace).Get(kubervisorServicesName, meta_v1.GetOptions{})
		if err == nil && ks != nil {
			kvs.Items = append(kvs.Items, *ks)
		}
		if err != nil && !apierrors.IsNotFound(err) {
			fmt.Printf("unable to get kubervisorservice err:%v\n", err)
			os.Exit(1)
		}
	}

	data := computeTableData(kvs)
	if len(data) == 0 {
		resourcesNotFound()
		os.Exit(0)
	}

	table := newTable()
	for _, v := range data {
		table.Append(v)
	}
	table.Render() // Send output

	os.Exit(0)
}

func hasStatus(ks *api.KubervisorService, conditionType api.KubervisorServiceConditionType, status kapiv1.ConditionStatus) bool {
	for _, cond := range ks.Status.Conditions {
		if cond.Type == conditionType && cond.Status == status {
			return true
		}
	}
	return false
}

func resourcesNotFound() {
	fmt.Println("No resources found.")
}

func buildClusterStatus(ks *api.KubervisorService) string {
	status := []string{}

	if hasStatus(ks, api.KubervisorServiceRunning, kapiv1.ConditionTrue) {
		status = append(status, string(api.KubervisorServiceRunning))
	} else if hasStatus(ks, api.KubervisorServiceInitFailed, kapiv1.ConditionTrue) {
		status = append(status, string(api.KubervisorServiceInitFailed))
	} else if hasStatus(ks, api.KubeServiceNotAvailable, kapiv1.ConditionTrue) {
		status = append(status, string(api.KubeServiceNotAvailable))
	} else if hasStatus(ks, api.KubervisorServiceFailed, kapiv1.ConditionTrue) {
		status = append(status, string(api.KubervisorServiceFailed))
	}

	return strings.Join(status, "-")
}

func computeTableData(kvs *api.KubervisorServiceList) [][]string {
	data := [][]string{}
	for _, ks := range kvs.Items {
		status := buildClusterStatus(&ks)
		var nbPodManaged, nbPodPause, nbPodBreaked uint32
		if ks.Status.PodCounts != nil {
			nbPodManaged = ks.Status.PodCounts.NbPodsManaged
			nbPodPause = ks.Status.PodCounts.NbPodsPaused
			nbPodBreaked = ks.Status.PodCounts.NbPodsBreaked
		}
		data = append(data, []string{ks.Name, ks.Namespace, status, fmt.Sprintf("%d", nbPodManaged), fmt.Sprintf("%d", nbPodBreaked), fmt.Sprintf("%d", nbPodPause)})
	}

	return data
}

func newTable() *tablewriter.Table {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Namespace", "Status", "Pods Managed", "Pods Breaked", "Pods Paused"})
	table.SetBorders(tablewriter.Border{Left: false, Top: false, Right: false, Bottom: false})
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetRowLine(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetHeaderLine(false)

	return table
}

func configFromPath(path string) (clientcmd.ClientConfig, error) {
	rules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: path}
	credentials, err := rules.Load()
	if err != nil {
		return nil, fmt.Errorf("the provided credentials %q could not be loaded: %v", path, err)
	}

	overrides := &clientcmd.ConfigOverrides{
		Context: clientcmdapi.Context{
			Namespace: os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_NAMESPACE"),
		},
	}

	context := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONTEXT")
	if len(context) > 0 {
		rules := clientcmd.NewDefaultClientConfigLoadingRules()
		return clientcmd.NewNonInteractiveClientConfig(*credentials, context, overrides, rules), nil
	}
	return clientcmd.NewDefaultClientConfig(*credentials, overrides), nil
}

func getHomePath() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
		if home == "" {
			home = os.Getenv("USERPROFILE")
		}
	}

	return home
}

func getKubeConfigDefaultPath(home string) string {
	kubeconfig := filepath.Join(home, ".kube", "config")

	kubeconfigEnv := os.Getenv("KUBECONFIG")
	if len(kubeconfigEnv) > 0 {
		kubeconfig = kubeconfigEnv
	}

	configFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_CONFIG")
	kubeConfigFile := os.Getenv("KUBECTL_PLUGINS_GLOBAL_FLAG_KUBECONFIG")
	if len(configFile) > 0 {
		kubeconfig = configFile
	} else if len(kubeConfigFile) > 0 {
		kubeconfig = kubeConfigFile
	}

	return kubeconfig
}
