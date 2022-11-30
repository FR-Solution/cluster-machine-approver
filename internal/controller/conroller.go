package controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
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
	Approve(ctx context.Context, r *v1.CertificateSigningRequest) error
	Deny(ctx context.Context, r *v1.CertificateSigningRequest) error
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
) (ctrl *controller, err error) {
	ctrl = &controller{
		k8s:   k8s,
		cloud: cloud,
	}

	ctrl.rInstanceName, err = regexp.Compile(instanceNameLayout)
	return ctrl, err
}

func (s *controller) Start() error {
	requestChan, err := s.k8s.CertificateSigningRequestsChan()
	if err != nil {
		return err
	}
	for r := range requestChan {
		logger := zap.L().With(zap.String("name", r.Name))
		logger.Debug("new_csr")

		isVerification, err := s.verification(r, logger)
		if err != nil {
			logger.Error("verification request", zap.Error(err))
		}

		if isVerification {
			logger.Debug("approve_request")
			err := s.k8s.Approve(context.TODO(), r)
			if err != nil {
				logger.Error("approve_request", zap.Error(err))
			}
		} else {
			logger.Debug("deny_request", zap.String("name", r.Name))
			err := s.k8s.Deny(context.TODO(), r)
			if err != nil {
				logger.Error("deny_request", zap.Error(err))
			}
		}
	}
	return nil
}

func (s *controller) Stop() {
	s.k8s.Stop()
}

func (s *controller) verification(req *v1.CertificateSigningRequest, logger *zap.Logger) (bool, error) {
	csr, err := parseCertificateRequest(req.Spec.Request)
	if err != nil {
		return false, err
	}

	virtualMachineName, err := s.getVirtualMachineName(csr.Subject.CommonName)
	if err != nil {
		return false, err
	}

	logger.Debug("verification", zap.Any("vm_name", virtualMachineName))

	vmIPs, err := s.cloud.GetInstanceAddresses(context.TODO(), virtualMachineName)
	if err != nil {
		return false, err
	}

	for _, ip := range csr.IPAddresses {
		if !ipIsExist(ip, vmIPs) {
			logger.Debug("ip_check", zap.String("result", "ip is not found in vm ips"), zap.String("ip", ip.String()))
			return false, nil
		}
	}
	return true, nil
}

func (s *controller) getVirtualMachineName(str string) (string, error) {
	submatch := s.rInstanceName.FindStringSubmatch(str)
	if submatch == nil {
		return "", fmt.Errorf("virtual machine name in %s not found", str)
	}
	return submatch[1], nil
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
