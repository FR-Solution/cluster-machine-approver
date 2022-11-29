package yandex

import (
	"encoding/json"

	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

func getCredentials(iamJSON []byte) (ycsdk.Credentials, error) {
	var iamKey iamkey.Key
	err := json.Unmarshal(iamJSON, &iamKey)
	if err != nil {
		return nil, err
	}
	credentials, err := ycsdk.ServiceAccountKey(&iamKey)
	if err != nil {
		return nil, err
	}

	return credentials, nil
}
