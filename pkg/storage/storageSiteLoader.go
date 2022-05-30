package storage

import (
	"os"
	"time"

	"github.com/atekoa/dvc-http-remote/pkg/dvc"
	"github.com/atekoa/dvc-http-remote/pkg/pool"
	"github.com/patrickmn/go-cache"
)

type storageSiteLoader struct {
	cache     *cache.Cache
	localPath string
}

func NewStorageSiteLoader(path string) *storageSiteLoader {
	return &storageSiteLoader{
		cache:     cache.New(30*time.Minute, 60*time.Minute),
		localPath: path,
	}
}

func (s *storageSiteLoader) LoadConfig(local bool) (*pool.ConnectionConfig, error) {
	if local {
		return &pool.ConnectionConfig{
			Type:          pool.ConfigTypeHttp,
			ContainerName: s.localPath,
		}, nil
	} else {
		azureUrl := os.Getenv("AZURE_STORAGE_URL")              // "azure://test/"
		azureConnection := os.Getenv("AZURE_CONNECTION_STRING") // "DefaultEndpointsProtocol=https;AccountName=..."
		return dvc.LoadAzureConfig(azureUrl, azureConnection)
	}
}
