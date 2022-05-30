package pool

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/Azure/azure-pipeline-go/pipeline"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"gocloud.dev/blob"
	"gocloud.dev/blob/azureblob"
	"gocloud.dev/blob/fileblob"
)

const (
	ConfigTypeUnknown ConfigType = "unknown"
	ConfigTypeAzure   ConfigType = "azure"
	ConfigTypeHttp    ConfigType = "http"
)

type ConfigType string

type ConnectionConfig struct {
	Type ConfigType
	URL  *url.URL

	ContainerName string

	ConnectionString string
	AccountName      string
	AccountKey       string

	RemoteId int
}

type CloudConn struct {
	*blob.Bucket
	remoteId int
	closed   bool
}

func (config *ConnectionConfig) OpenHttp(ctx context.Context) (CloudConn, error) {
	// The directory you pass to fileblob.OpenBucket must exist first.
	// const myDir = "localhost/"
	if err := os.MkdirAll(config.ContainerName, 0777); err != nil {
		log.Fatal(err)
	}

	b, errOpen := fileblob.OpenBucket(config.ContainerName, &fileblob.Options{CreateDir: true})
	if errOpen != nil {
		return CloudConn{nil, 0, true}, fmt.Errorf("Cannot open Bucket, %w", errOpen)
	}
	return CloudConn{b, config.RemoteId, false}, nil
}

func (config *ConnectionConfig) OpenAzure(ctx context.Context) (CloudConn, error) {
	accountName := azureblob.AccountName(config.AccountName)
	accountKey := azureblob.AccountKey(config.AccountKey)
	containerName := config.ContainerName

	credential, err := azureblob.NewCredential(accountName, accountKey)
	if err != nil {
		return CloudConn{nil, 0, true}, fmt.Errorf("Cannot create Azure credentials, %w", err)
	}

	po := azblob.PipelineOptions{
		// Set RetryOptions to control how HTTP request are retried when retryable failures occur
		Retry: azblob.RetryOptions{
			Policy:   azblob.RetryPolicyExponential, // Use exponential backoff as opposed to linear
			MaxTries: 10,                            // Try at most 10 times to perform the operation (set to 1 to disable retries)

			// Per https://godoc.org/github.com/Azure/azure-storage-blob-go/azblob#RetryOptions
			// this should be set to a very nigh number (they claim 60s per MB).
			// That could end up being days so we are limiting this to four hours.
			TryTimeout:    4 * time.Hour,    // Maximum time allowed for any single try
			RetryDelay:    time.Second * 30, // Backoff amount for each retry (exponential or linear)
			MaxRetryDelay: time.Minute * 60, // Max delay between retries
		},

		// Set RequestLogOptions to control how each HTTP request & its response is logged
		RequestLog: azblob.RequestLogOptions{
			LogWarningIfTryOverThreshold: time.Millisecond * 200, // A successful response taking more than this time to arrive is logged as a warning
		},

		// Set LogOptions to control what & where all pipeline log events go
		Log: pipeline.LogOptions{
			Log: func(s pipeline.LogLevel, m string) { // This func is called to log each event
				// This method is not called for filtered-out severities.
				log.Output(2, m) // This example uses Go's standard logger
			},
			ShouldLog: func(level pipeline.LogLevel) bool {
				return level <= pipeline.LogWarning // Log all events from warning to more severe
			},
		},

		// Set HTTPSender to override the default HTTP Sender that sends the request over the network
		HTTPSender: pipeline.FactoryFunc(func(next pipeline.Policy, po *pipeline.PolicyOptions) pipeline.PolicyFunc {
			return func(ctx context.Context, request pipeline.Request) (pipeline.Response, error) {
				// Implement the HTTP client that will override the default sender.
				// For example, below HTTP client uses a transport that is different from http.DefaultTransport
				client := http.Client{
					Transport: &http.Transport{
						Proxy: nil,
						DialContext: (&net.Dialer{
							Timeout:   1 * time.Hour,
							KeepAlive: 1 * time.Hour,
							DualStack: true,
						}).DialContext,
						MaxIdleConns:          1000,
						MaxConnsPerHost:       200,
						IdleConnTimeout:       60 * time.Minute,
						TLSHandshakeTimeout:   60 * time.Minute,
						ExpectContinueTimeout: 60 * time.Minute,
					},
				}

				// Send the request over the network
				resp, err := client.Do(request.WithContext(context.Background()))

				return pipeline.NewHTTPResponse(resp), err
			}
		}),
	}
	pipeline := azureblob.NewPipeline(credential, po)
	b, errOpen := azureblob.OpenBucket(ctx, pipeline, accountName, containerName, &azureblob.Options{Credential: credential})
	if errOpen != nil {
		return CloudConn{nil, 0, true}, fmt.Errorf("Cannot create Azure credentials, %w", err)
	}
	return CloudConn{b, config.RemoteId, false}, nil
}
