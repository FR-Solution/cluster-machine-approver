package controller

import (
	"context"

	"go.uber.org/zap"
)

type k8s interface {
	CertificateSigningRequestsChan() (<-chan *CertificateSigningRequest, error)
	Apply(ctx context.Context, r *CertificateSigningRequest) error
}

type controller struct {
	k8s k8s
}

func New(k8s k8s) *controller {
	return &controller{
		k8s: k8s,
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

func (s *controller) verification(req *CertificateSigningRequest) (bool, error) {
	return true, nil
}
