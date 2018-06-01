package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/golang/glog"
	"github.com/olekukonko/tablewriter"

	kapiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"

	api "github.com/amadeusitgroup/kubervisor/pkg/api/kubervisor/v1alpha1"
	kvclient "github.com/amadeusitgroup/kubervisor/pkg/client"
)

func main() {
	cmdBin := "kubectl"
	if val := os.Getenv("KUBECTL_PLUGINS_CALLER"); val != "" {
		cmdBin = val
	}

	namespace := "default"
	if val := os.Getenv("KUBECTL_PLUGINS_CURRENT_NAMESPACE"); val != "" {
		namespace = val
	}

	kubervisorServicesName := ""
	if val := os.Getenv("KUBECTL_PLUGINS_LOCAL_FLAG_KS"); val != "" {
		kubervisorServicesName = val
	}

	kubeConfigBytes, err := exec.Command(cmdBin, "config", "view").Output()
	if err != nil {
		log.Fatal(err)
	}

	tmpConf, err := ioutil.TempFile("", "example")
	if err != nil {
		log.Fatal(err)
	}

	defer os.Remove(tmpConf.Name()) // clean up
	if _, err = tmpConf.Write(kubeConfigBytes); err != nil {
		log.Fatal(err)
	}
	// use the current context in kubeconfig
	kubeConfig, err := clientcmd.BuildConfigFromFlags("", tmpConf.Name())
	if err != nil {
		panic(err.Error())
	}

	kubervisorClient, err := kvclient.NewClient(kubeConfig)
	if err != nil {
		glog.Fatalf("Unable to init kubervisor.clientset from kubeconfig:%v", err)
	}

	var kvs *api.KubervisorServiceList
	if kubervisorServicesName == "" {
		kvs, err = kubervisorClient.KubervisorV1alpha1().KubervisorServices(namespace).List(meta_v1.ListOptions{})
		if err != nil {
			fmt.Printf("unable to list kubervisorservice err:%v\n", err)
			return
		}
	} else {
		kvs = &api.KubervisorServiceList{}
		ks, err := kubervisorClient.KubervisorV1alpha1().KubervisorServices(namespace).Get(kubervisorServicesName, meta_v1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				fmt.Printf("unable to get kubervisorservice err:%v\n", err)
				os.Exit(1)
			}
		} else {
			kvs.Items = append(kvs.Items, *ks)
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
