package k8s

import (
	"context"
	"os"
	"sync"

	"github.com/fraima/cluster-machine-approver/internal/controller"
	certificatesv1 "k8s.io/api/certificates/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/certificates/v1"
	"k8s.io/client-go/tools/clientcmd"
)

type k8s struct {
	csr v1.CertificateSigningRequestInterface

	lock     sync.Mutex
	watchers []watch.Interface
}

func New(kubeconfigPath string) (*k8s, error) {
	configBytes, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	config, err := clientcmd.RESTConfigFromKubeConfig(configBytes)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	k := &k8s{
		csr: client.CertificatesV1().CertificateSigningRequests(),
	}

	return k, err
}

func (s *k8s) Stop() {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, w := range s.watchers {
		w.Stop()
	}
}

func (s *k8s) CertificateSigningRequestsChan() (<-chan controller.CertificateSigningRequest, error) {
	w, err := s.csr.Watch(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	s.lock.Lock()
	s.watchers = append(s.watchers, w)
	s.lock.Unlock()

	rChan := make(chan controller.CertificateSigningRequest)

	go func() {
		for event := range w.ResultChan() {
			rChan <- controller.CertificateSigningRequest(event.Object.(*certificatesv1.CertificateSigningRequest))
		}
	}()

	return rChan, err
}

func (s *k8s) Apply(ctx context.Context, r *certificatesv1.CertificateSigningRequest) error {
	r.Status.Conditions = append(r.Status.Conditions, certificatesv1.CertificateSigningRequestCondition{
		Type:           certificatesv1.CertificateApproved,
		Reason:         "User activation",
		Message:        "This CSR was approved",
		LastUpdateTime: metav1.Now(),
	})

	_, err := s.csr.UpdateApproval(ctx, r.Name, r, metav1.UpdateOptions{})
	return err
}
