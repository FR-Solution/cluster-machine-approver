package controller

import (
	"context"
	"crypto/x509"
)

type k8s interface {
	CertificateSigningRequestsChan() (<-chan CertificateSigningRequest, error)
	Apply(ctx context.Context, r *CertificateSigningRequest) error
}

type controller struct {
	k8s k8s
}

func (s *controller) Start() error {
	requestChan, err := s.k8s.CertificateSigningRequestsChan()
	if err != nil {
		return err
	}
	for r := range requestChan {
		csr, _ := x509.ParseCertificateRequest(r.Spec.Request)
		csr.IPAddresses
	}

}
