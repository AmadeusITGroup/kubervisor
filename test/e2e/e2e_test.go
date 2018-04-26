package e2e

import (
	goflag "flag"
	"os"
	"testing"

	"github.com/spf13/pflag"

	"github.com/amadeusitgroup/kubervisor/test/e2e/framework"
)

func TestE2E(t *testing.T) {
	RunE2ETests(t)
}

func TestMain(m *testing.M) {

	pflag.StringVar(&framework.FrameworkContext.KubeConfigPath, "kubeconfig", "", "Path to kubeconfig")
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	os.Exit(m.Run())
}
