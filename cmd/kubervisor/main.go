package main

import (
	"context"
	goflag "flag"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/amadeusitgroup/kubervisor/pkg/controller"
	"github.com/amadeusitgroup/kubervisor/pkg/signal"
	"github.com/amadeusitgroup/kubervisor/pkg/utils"
)

func main() {
	utils.BuildInfos()
	runtime.GOMAXPROCS(runtime.NumCPU())

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	config := controller.NewConfig(logger)
	config.AddFlags(pflag.CommandLine)

	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	pflag.Parse()
	if err := goflag.CommandLine.Parse([]string{}); err != nil {
		logger.Sugar().Errorf("unable to Parse command line: %s", err)
		os.Exit(1)
	}

	if config.Debug {
		l, err := zap.NewDevelopment()
		if err != nil {
			logger.Sugar().Errorf("unable Development zap logger, error: %s", err)
		}
		config.SetLogger(l)
	}
	ctrl := controller.New(config)

	if err := run(ctrl); err != nil {
		logger.Sugar().Errorf("run function return an error: %s", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func run(ctrl *controller.Controller) error {
	ctx, cancelFunc := context.WithCancel(context.Background())
	go signal.HandleSignal(cancelFunc)
	defer func() {
		if err := ctrl.Logger.Sync(); err != nil {
			ctrl.Logger.Sugar().Error(err)
		}
	}()
	return ctrl.Run(ctx.Done())
}
