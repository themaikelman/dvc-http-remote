package dvc

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/atekoa/dvc-http-remote/pkg/pool"
)

func Parse(azureconnstring string) (map[string]string, error) {
	parts := map[string]string{}
	for _, pair := range strings.Split(string(azureconnstring), ";") {
		if pair == "" {
			continue
		}

		equalDex := strings.IndexByte(pair, '=')
		if equalDex <= 0 {
			return nil, fmt.Errorf("Invalid connection segment %q", pair)
		}

		value := strings.TrimSpace(pair[equalDex+1:])
		key := strings.TrimSpace(pair[:equalDex])
		parts[key] = value
	}

	return parts, nil
}

func LoadAzureConfig(URL string, connectionString string) (*pool.ConnectionConfig, error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, fmt.Errorf("url cannot be parsed, %w", err)
	}
	container := parsed.Host

	connectionParams, err := Parse(connectionString)
	if err != nil {
		return nil, err
	}
	return &pool.ConnectionConfig{
		Type:             pool.ConfigTypeAzure,
		URL:              parsed,
		ConnectionString: connectionString,
		ContainerName:    container,
		AccountKey:       connectionParams["AccountKey"],
		AccountName:      connectionParams["AccountName"],
	}, nil
}
