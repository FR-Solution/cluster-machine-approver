package controller

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
	certificateSigningRequestsChan := make(chan *v1.CertificateSigningRequest)
	var certificateSigningRequestsOutChan <-chan *v1.CertificateSigningRequest = certificateSigningRequestsChan
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

	ctrl, err := New(
		k8sMock,
		cloudMock,
		testRegexp,
	)
	require.NoError(t, err)

	t.Run("with approve", func(t *testing.T) {
		go func() {
			err = ctrl.Start()
			require.NoError(t, err)
		}()
		certificateSigningRequestsChan <- testCertificateSigningRequest

		close(certificateSigningRequestsChan)
	})
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
	certificateSigningRequestsChan := make(chan *v1.CertificateSigningRequest)
	var certificateSigningRequestsOutChan <-chan *v1.CertificateSigningRequest = certificateSigningRequestsChan
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

	ctrl, err := New(
		k8sMock,
		cloudMock,
		testRegexp,
	)
	require.NoError(t, err)

	t.Run("with deny", func(t *testing.T) {
		go func() {
			err = ctrl.Start()
			require.NoError(t, err)
		}()
		certificateSigningRequestsChan <- testCertificateSigningRequest

		close(certificateSigningRequestsChan)
	})
}

func TestGetVirtualMachineName(t *testing.T) {
	testRegexp := "system:node:(.[^ ]*)"

	ctrl, err := New(
		nil,
		nil,
		testRegexp,
	)
	require.NoError(t, err)

	t.Run("success test", func(t *testing.T) {
		testStr := "system:node:worker-1-cluster-2"
		expectedName := "worker-1-cluster-2"

		actualName, err := ctrl.getVirtualMachineName(testStr)
		require.NoError(t, err)
		require.Equal(t, expectedName, actualName)
	})

	t.Run("failed test", func(t *testing.T) {
		testStr := "string:without-name"
		expectedName := ""
		expectedErr := fmt.Errorf("virtual machine name in %s not found", testStr)

		actualName, actualErr := ctrl.getVirtualMachineName(testStr)
		require.Equal(t, expectedErr, actualErr)
		require.Equal(t, expectedName, actualName)
	})
}

func TestIpIsExist(t *testing.T) {
	ipSet := []net.IP{
		net.ParseIP("123.123.123.123"),
		net.ParseIP("124.124.124.124"),
		net.ParseIP("125.125.125.125"),
		net.ParseIP("126.126.126.126"),
	}

	t.Run("success test", func(t *testing.T) {
		testIP := net.ParseIP("123.123.123.123")

		isExist := ipIsExist(testIP, ipSet)
		require.True(t, isExist)
	})

	t.Run("failed test", func(t *testing.T) {
		testIP := net.ParseIP("128.128.128.128")

		isExist := ipIsExist(testIP, ipSet)
		require.False(t, isExist)
	})
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
