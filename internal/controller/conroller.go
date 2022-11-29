package controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net"
	"regexp"

	"go.uber.org/zap"
	v1 "k8s.io/api/certificates/v1"
)

type cloud interface {
	GetInstanceAddresses(ctx context.Context, instanceName string) ([]net.IP, error)
}

type k8s interface {
	CertificateSigningRequestsChan() (<-chan *v1.CertificateSigningRequest, error)
	Apply(ctx context.Context, r *v1.CertificateSigningRequest) error
	Stop()
}

type controller struct {
	k8s   k8s
	cloud cloud

	rInstanceName *regexp.Regexp
}

func New(
	k8s k8s,
	cloud cloud,

	instanceNameLayout string,
) *controller {
	return &controller{
		k8s:   k8s,
		cloud: cloud,
	}
}

func (s *controller) Start() error {
	requestChan, err := s.k8s.CertificateSigningRequestsChan()
	if err != nil {
		return err
	}
	for r := range requestChan {
		isVerification, err := s.verification(r)
		if err != nil {
			zap.L().Error("verification request", zap.Error(err))
		}
		if isVerification {
			err := s.k8s.Apply(context.TODO(), r)
			if err != nil {
				zap.L().Error("apply request", zap.Error(err))
			}
		}
	}
	return nil
}

func (s *controller) Stop() {
	s.k8s.Stop()
}

func (s *controller) verification(req *v1.CertificateSigningRequest) (bool, error) {
	certRequest, err := parseCertificateRequest(req.Spec.Request)
	if err != nil {
		return false, err
	}

	vmIPs, err := s.cloud.GetInstanceAddresses(context.TODO(), s.getVirtualMachineName(certRequest.Subject.CommonName))
	if err != nil {
		return false, err
	}

	for _, ip := range certRequest.IPAddresses {
		if !ipIsExist(ip, vmIPs) {
			return false, nil
		}
	}
	return true, nil
}

func (s *controller) getVirtualMachineName(commonName string) string {
	return s.rInstanceName.FindString(commonName)
}

func parseCertificateRequest(data []byte) (*x509.CertificateRequest, error) {
	b, _ := pem.Decode(data)
	if b == nil {
		return x509.ParseCertificateRequest(data)
	}
	return x509.ParseCertificateRequest(b.Bytes)
}

func ipIsExist(ip net.IP, ips []net.IP) bool {
	for _, i := range ips {
		if ip.Equal(i) {
			return true
		}
	}
	return false
}
