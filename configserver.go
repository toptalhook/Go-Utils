package utils

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Laisky/zap"
	"github.com/pkg/errors"
)

// ConfigSource config item in config-server
type ConfigSource struct {
	Name   string                 `json:"name"`
	Source map[string]interface{} `json:"source"`
}

// Config whole configuation return by config-server
type Config struct {
	Name     string          `json:"name"`
	Profiles []string        `json:"profiles"`
	Label    string          `json:"label"`
	Version  string          `json:"version"`
	Sources  []*ConfigSource `json:"propertySources"`
}

// ConfigSrv can load configuration from Spring-Cloud-Config-Server
type ConfigSrv struct {
	RemoteCfg *Config

	url, // config-server api
	profile, // env
	label, // branch
	app string // app name
}

// NewConfigSrv create ConfigSrv
func NewConfigSrv(url, app, profile, label string) *ConfigSrv {
	return &ConfigSrv{
		RemoteCfg: &Config{},
		url:       url,
		app:       app,
		label:     label,
		profile:   profile,
	}
}

// Fetch load data from config-server
func (c *ConfigSrv) Fetch() error {
	url := strings.Join([]string{c.url, c.app, c.profile, c.label}, "/")
	err := RequestJSONWithClient(httpClient, "get", url, &RequestData{}, c.RemoteCfg)
	if err != nil {
		return errors.Wrap(err, "try to get config got error")
	}

	return nil
}

// Get get `interface{}` from the localcache of config-server
func (c *ConfigSrv) Get(name string) (interface{}, bool) {
	var (
		item string
		val  interface{}
	)
	for _, src := range c.RemoteCfg.Sources {
		for item, val = range src.Source {
			if item == name {
				return val, true
			}
		}
	}

	return nil, false
}

// GetString get `string` from the localcache of config-server
func (c *ConfigSrv) GetString(name string) (string, bool) {
	if val, ok := c.Get(name); ok {
		return val.(string), true
	}

	return "", false
}

// GetInt get `int` from the localcache of config-server
func (c *ConfigSrv) GetInt(name string) (int, bool) {
	if val, ok := c.Get(name); ok {
		if i, err := strconv.ParseInt(fmt.Sprintf("%v", val), 10, 64); err != nil {
			Logger.Error("try to parse int got error", zap.String("val", fmt.Sprintf("%v", val)))
			return 0, false
		} else {
			return int(i), true
		}
	}
	return 0, false
}

// GetBool get `bool` from the localcache of config-server
func (c *ConfigSrv) GetBool(name string) (bool, bool) {
	if val, ok := c.Get(name); ok {
		if ret, err := strconv.ParseBool(fmt.Sprintf("%v", val)); err != nil {
			Logger.Error("try to parse bool got error", zap.String("val", fmt.Sprintf("%v", val)))
			return false, false
		} else {
			return ret, true
		}
	}
	return false, false
}

// Map interate `set(k, v)`
func (c *ConfigSrv) Map(set func(string, interface{})) {
	var (
		key string
		val interface{}
		src *ConfigSource
	)
	for i := 0; i < len(c.RemoteCfg.Sources); i++ {
		src = c.RemoteCfg.Sources[i]
		for key, val = range src.Source {
			Logger.Debug("set settings", zap.String("key", key), zap.String("val", fmt.Sprint(val)))
			set(key, val)
		}
	}
}
