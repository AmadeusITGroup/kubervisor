package main

import (
	"context"
	goflag "flag"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/amadeusitgroup/podkubervisor/pkg/controller"
	"github.com/amadeusitgroup/podkubervisor/pkg/signal"
	"github.com/amadeusitgroup/podkubervisor/pkg/utils"
)

func main() {
	utils.BuildInfos()
	runtime.GOMAXPROCS(runtime.NumCPU())

	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	config := controller.NewConfig(logger)
	config.AddFlags(pflag.CommandLine)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	goflag.CommandLine.Parse([]string{})

	ctrl := controller.New(config)

	if err := run(ctrl); err != nil {
		//glog.Errorf("RedisOperator returns an error:%v", err)
		os.Exit(1)
	}

	os.Exit(0)
}

func run(ctrl *controller.Controller) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go signal.HandleSignal(cancelFunc)

	return ctrl.Run(ctx.Done())
}
