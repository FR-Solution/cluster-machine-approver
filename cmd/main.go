package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	certificatesv1 "k8s.io/api/certificates/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

var kubeconfigPath = `/home/geo/.kube/config`

func main() {
	configBytes, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		log.Fatal(err)
	}

	cfg, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		// MetricsBindAddress:     config.MetricsAddr,
		// HealthProbeBindAddress: config.ProbeAddr,
	})
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		err = mgr.GetCache().Start(context.Background())
		if err != nil {
			log.Fatal(err)
		}
	}()

	time.Sleep(3 * time.Second)

	cli := mgr.GetClient()

	var (
		csr certificatesv1.CertificateSigningRequest
		req = ctrl.Request{
			NamespacedName: types.NamespacedName{},
		}
	)

	if err := cli.Get(context.TODO(), req.NamespacedName, &csr); err != nil {
		fmt.Println(err)
	}
	fmt.Println(csr)
}
