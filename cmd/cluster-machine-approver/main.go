package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/kelseyhightower/envconfig"
	"go.uber.org/zap"

	"github.com/fraima/cluster-machine-approver/internal/cloud/yandex"
	"github.com/fraima/cluster-machine-approver/internal/controller"
	"github.com/fraima/cluster-machine-approver/internal/k8s"
)

var (
	Version = "undefined"
)

type configuration struct {
	KubeHost           string `envconfig:"KUBE_HOST" default:"kubernetes.default"`
	KubeTokenFile      string `envconfig:"KUBE_TOKEN_FILE" default:"/run/secrets/kubernetes.io/serviceaccount/token"`
	AIMJson            []byte `envconfig:"YANDEX_AIM_JSON" required:"true"`
	FolderID           string `envconfig:"YANDEX_FOLDER_ID" required:"true"`
	InstanceNameLayout string `envconfig:"INSTANCE_NAME_LAYOUT" default:"system:node:(.[^ ]*)"`
}

func main() {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level.SetLevel(zap.DebugLevel)
	logger, err := loggerConfig.Build()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)

	var cfg configuration
	if err := envconfig.Process("", &cfg); err != nil {
		zap.L().Panic("init configuration", zap.Error(err))
	}

	zap.L().Debug("configuration", zap.Any("config", cfg), zap.String("version", Version))

	k, err := k8s.Connect(
		cfg.KubeHost,
		cfg.KubeTokenFile,
	)
	if err != nil {
		zap.L().Fatal("connect k8s", zap.Error(err))
	}

	cloud, err := yandex.ConnectCloud(cfg.AIMJson, cfg.FolderID)
	if err != nil {
		zap.L().Fatal("connect yandex cloude", zap.Error(err))
	}

	ctrl, err := controller.New(k, cloud, cfg.InstanceNameLayout)
	if err != nil {
		zap.L().Fatal("init controller", zap.Error(err))
	}

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
