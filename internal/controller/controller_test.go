package controller_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/certificates/v1"

	"github.com/fraima/cluster-machine-approver/internal/controller"
	"github.com/fraima/cluster-machine-approver/internal/mocks"
)

var (
	nilError error
)

func TestApprove(t *testing.T) {
	testRegexp := "system:node:(.[^ ]*)"
	testVirtualMachineName := "test-virtual-name"
	testCommonName := fmt.Sprintf("system:node:%s", testVirtualMachineName)
	testIPSet := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
		net.ParseIP("126.126.126.126"),
	}
	testCertificateSigningRequest := testCSR(t, testCommonName, testIPSet)

	k8sMock := mocks.NewK8s(t)
	certificateSigningRequestsChan := make(chan controller.Event)
	var certificateSigningRequestsOutChan <-chan controller.Event = certificateSigningRequestsChan

	k8sMock.On("CertificateSigningRequestsChan").
		Return(certificateSigningRequestsOutChan, nilError)

	k8sMock.On("Approve", testCertificateSigningRequest).
		Return(nilError)

	cloudMock := mocks.NewCloud(t)

	testApproveIPSet := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
		net.ParseIP("126.126.126.126"),
	}

	cloudMock.On("GetInstanceAddresses", testVirtualMachineName).
		Return(testApproveIPSet, nilError)

	ctrl, err := controller.New(
		k8sMock,
		cloudMock,
		testRegexp,
	)
	require.NoError(t, err)

	go func() {
		err = ctrl.Start()
		require.NoError(t, err)
	}()
	certificateSigningRequestsChan <- testCertificateSigningRequest

	close(certificateSigningRequestsChan)
}

func TestDeny(t *testing.T) {
	testRegexp := "system:node:(.[^ ]*)"
	testVirtualMachineName := "test-virtual-name"
	testCommonName := fmt.Sprintf("system:node:%s", testVirtualMachineName)
	testIPSet := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
		net.ParseIP("126.126.126.126"),
	}
	testCertificateSigningRequest := testCSR(t, testCommonName, testIPSet)

	k8sMock := mocks.NewK8s(t)
	certificateSigningRequestsChan := make(chan controller.Event)
	var certificateSigningRequestsOutChan <-chan controller.Event = certificateSigningRequestsChan

	k8sMock.On("CertificateSigningRequestsChan").
		Return(certificateSigningRequestsOutChan, nilError)
	k8sMock.On("Deny", testCertificateSigningRequest).
		Return(nilError)

	cloudMock := mocks.NewCloud(t)

	testDenyIPSet := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
	}

	cloudMock.On("GetInstanceAddresses", testVirtualMachineName).
		Return(testDenyIPSet, nilError)

	ctrl, err := controller.New(
		k8sMock,
		cloudMock,
		testRegexp,
	)
	require.NoError(t, err)

	go func() {
		err = ctrl.Start()
		require.NoError(t, err)
	}()
	certificateSigningRequestsChan <- testCertificateSigningRequest

	close(certificateSigningRequestsChan)
}

func TestWorkflow(t *testing.T) {
	testRegexp := "system:node:(.[^ ]*)"

	testIPSetApprove := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
	}

	testVirtualMachineApproveName := "test-approve-1-virtual-name"
	testCertificateSigningRequestApprove := testCSR(t, fmt.Sprintf("system:node:%s", testVirtualMachineApproveName), testIPSetApprove)

	testIPSetDeny := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
		net.ParseIP("126.126.126.126"),
	}

	testVirtualMachineDenyName := "test-deny-virtual-name"
	testCertificateSigningRequestDeny := testCSR(t, fmt.Sprintf("system:node:%s", testVirtualMachineDenyName), testIPSetDeny)

	cloudMock := mocks.NewCloud(t)

	cloudMock.On("GetInstanceAddresses", testVirtualMachineApproveName).
		Return(testIPSetApprove, nilError)

	cloudMock.On("GetInstanceAddresses", testVirtualMachineDenyName).
		Return(testIPSetApprove, nilError)

	k8sMock := mocks.NewK8s(t)
	certificateSigningRequestsChan := make(chan controller.Event)
	var certificateSigningRequestsOutChan <-chan controller.Event = certificateSigningRequestsChan

	k8sMock.On("CertificateSigningRequestsChan").
		Return(certificateSigningRequestsOutChan, nilError)
	k8sMock.On("Deny", testCertificateSigningRequestDeny).
		Return(nilError)
	k8sMock.On("Approve", testCertificateSigningRequestApprove).
		Return(nilError)

	ctrl, err := controller.New(
		k8sMock,
		cloudMock,
		testRegexp,
	)
	require.NoError(t, err)

	go func() {
		err = ctrl.Start()
		require.NoError(t, err)
	}()
	certificateSigningRequestsChan <- testCertificateSigningRequestApprove
	certificateSigningRequestsChan <- testCertificateSigningRequestDeny

	certificateSigningRequestsChan <- testCertificateSigningRequestApprove
	certificateSigningRequestsChan <- testCertificateSigningRequestApprove

	certificateSigningRequestsChan <- testCertificateSigningRequestDeny
	certificateSigningRequestsChan <- testCertificateSigningRequestDeny
	certificateSigningRequestsChan <- testCertificateSigningRequestApprove

	close(certificateSigningRequestsChan)
}

func testCSR(t *testing.T, testCommonName string, testIPSet []net.IP) *v1.CertificateSigningRequest {
	pk, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(t, err)

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: testCommonName,
		},
		IPAddresses: testIPSet,
	}

	csr, err := x509.CreateCertificateRequest(rand.Reader, &template, pk)
	require.NoError(t, err)

	return &v1.CertificateSigningRequest{
		Spec: v1.CertificateSigningRequestSpec{
			Request: pem.EncodeToMemory(
				&pem.Block{
					Type:  "CERTIFICATE REQUEST",
					Bytes: csr,
				},
			),
		},
	}
}
