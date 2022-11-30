package yandex

import (
	"context"
	"fmt"
	"net"

	compute "github.com/yandex-cloud/go-genproto/yandex/cloud/compute/v1"
	ycsdk "github.com/yandex-cloud/go-sdk"
)

type cloud struct {
	folderID string
	sdk      *ycsdk.SDK
}

func ConnectCloud(
	iamJSON []byte,
	folderID string,
) (*cloud, error) {
	creds, err := getCredentials(iamJSON)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(context.Background(), ycsdk.Config{Credentials: creds})
	if err != nil {
		return nil, err
	}

	return &cloud{
		folderID: folderID,
		sdk:      sdk,
	}, nil
}

func (s *cloud) GetInstanceAddresses(ctx context.Context, instanceName string) ([]net.IP, error) {
	result, err := s.sdk.Compute().Instance().List(ctx, &compute.ListInstancesRequest{
		FolderId: s.folderID,
		PageSize: 2,
		Filter:   fmt.Sprintf("name = \"%s\"", instanceName),
	})

	if err != nil {
		return nil, err
	}

	if len(result.Instances) > 1 {
		return nil, fmt.Errorf("more than 1 instances found by the name %q", instanceName)
	}
	if len(result.Instances) == 0 {
		return nil, fmt.Errorf("no than 1 instances found by the name %q", instanceName)
	}

	return extractAddresses(result.Instances[0]), nil
}

func extractAddresses(instance *compute.Instance) []net.IP {
	var nodeAddresses []net.IP

	for _, iface := range instance.NetworkInterfaces {
		if iface.GetPrimaryV4Address() != nil {
			nodeAddresses = append(nodeAddresses, net.ParseIP(iface.GetPrimaryV4Address().Address))
			if iface.GetPrimaryV4Address().GetOneToOneNat() != nil {
				nodeAddresses = append(nodeAddresses, net.ParseIP(iface.GetPrimaryV4Address().GetOneToOneNat().Address))
			}
		}
	}
	return nodeAddresses
}
