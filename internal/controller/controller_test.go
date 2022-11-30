package controller

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

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
