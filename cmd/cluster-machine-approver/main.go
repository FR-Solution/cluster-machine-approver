package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/fraima/cluster-machine-approver/internal/cloud/yandex"
	"github.com/fraima/cluster-machine-approver/internal/config"
	"github.com/fraima/cluster-machine-approver/internal/controller"
	"github.com/fraima/cluster-machine-approver/internal/k8s"
	"go.uber.org/zap"
)

var (
	Version = "undefined"
)

func main() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(zap.DebugLevel)
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	var configPath string
	flag.StringVar(&configPath, "config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Get(configPath)
	if err != nil {
		zap.L().Fatal("read configuration", zap.Error(err))
	}

	zap.L().Debug("configuration", zap.Any("config", cfg), zap.String("version", Version))

	k, err := k8s.Connect(cfg.KubeconfigPath)
	if err != nil {
		zap.L().Fatal("connect k8s", zap.Error(err))
	}

	cloud, err := yandex.ConnectCloud(cfg.AIMJson, cfg.FolderID)
	if err != nil {
		zap.L().Fatal("connect yandex cloude", zap.Error(err))
	}

	ctrl := controller.New(k, cloud, cfg.InstanceNameLayout)

	go func() {
		err := ctrl.Start()
		if err != nil {
			zap.L().Fatal("start controller", zap.Error(err))
		}
	}()

	zap.L().Info("started")

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	ctrl.Stop()

	zap.L().Info("goodbye")
}
