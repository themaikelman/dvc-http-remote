package dvc

import (
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/atekoa/dvc-http-remote/pkg/pool"
	"gopkg.in/ini.v1"
)

type DVCConfigParser struct {
	content *ini.File
}

func NewDVCConfig(content io.Reader, others ...interface{}) (*DVCConfigParser, error) {
	config, err := ini.LoadSources(
		ini.LoadOptions{
			IgnoreInlineComment: true,
			Insensitive:         true,
		},
		content,
		others...,
	)
	if err != nil {
		return nil, err
	}
	return &DVCConfigParser{
		content: config,
	}, nil
}

func (config DVCConfigParser) ListRemotes() []string {
	remotes := make([]string, 0)
	for _, i := range config.content.SectionStrings() {
		if strings.Contains(i, "remote") {
			remotes = append(remotes, strings.Split(i, "\"")[1])
		}
	}
	return remotes
}

func (config DVCConfigParser) GetRemote(remote string) (*ini.Section, error) {
	key := "'remote \"" + remote + "\"'"
	return config.content.GetSection(key)
}

func (config DVCConfigParser) GetRemoteType(remote string) (pool.ConfigType, error) {
	section, err := config.GetRemote(remote)
	if err != nil {
		return pool.ConfigTypeUnknown, err
	}
	urlValue, err := section.GetKey("url")
	if err != nil {
		return pool.ConfigTypeUnknown, err
	}
	return GetRemoteType(urlValue.String())
}

func GetRemoteType(URL string) (pool.ConfigType, error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return pool.ConfigTypeUnknown, err
	}
	switch parsed.Scheme {
	case "azure":
		return pool.ConfigTypeAzure, nil
	case "http":
		return pool.ConfigTypeHttp, nil
	default:
		return pool.ConfigTypeUnknown, errors.New("Invalid Scheme")
	}
}
