// SPDX-License-Identifier: GPL-3.0-or-later

package windows

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/netdata/netdata/go/plugins/plugin/go.d/agent/module"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/confopt"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/prometheus"
	"github.com/netdata/netdata/go/plugins/plugin/go.d/pkg/web"
)

//go:embed "config_schema.json"
var configSchema string

func init() {
	module.Register("windows", module.Creator{
		JobConfigSchema: configSchema,
		Defaults: module.Defaults{
			UpdateEvery: 5,
		},
		Create: func() module.Module { return New() },
		Config: func() any { return &Config{} },
	})
}

func New() *Collector {
	return &Collector{
		Config: Config{
			HTTPConfig: web.HTTPConfig{
				ClientConfig: web.ClientConfig{
					Timeout: confopt.Duration(time.Second * 5),
				},
			},
		},
		cache: cache{
			collection:                  make(map[string]bool),
			collectors:                  make(map[string]bool),
			cores:                       make(map[string]bool),
			nics:                        make(map[string]bool),
			volumes:                     make(map[string]bool),
			thermalZones:                make(map[string]bool),
			processes:                   make(map[string]bool),
			iis:                         make(map[string]bool),
			adcs:                        make(map[string]bool),
			services:                    make(map[string]bool),
			netFrameworkCLRExceptions:   make(map[string]bool),
			netFrameworkCLRInterops:     make(map[string]bool),
			netFrameworkCLRJIT:          make(map[string]bool),
			netFrameworkCLRLoading:      make(map[string]bool),
			netFrameworkCLRLocksThreads: make(map[string]bool),
			netFrameworkCLRMemory:       make(map[string]bool),
			netFrameworkCLRRemoting:     make(map[string]bool),
			netFrameworkCLRSecurity:     make(map[string]bool),
			mssqlInstances:              make(map[string]bool),
			mssqlDBs:                    make(map[string]bool),
			exchangeWorkload:            make(map[string]bool),
			exchangeLDAP:                make(map[string]bool),
			exchangeHTTPProxy:           make(map[string]bool),
			hypervVMMem:                 make(map[string]bool),
			hypervVMDevices:             make(map[string]bool),
			hypervVMInterfaces:          make(map[string]bool),
			hypervVswitch:               make(map[string]bool),
		},
		charts: &module.Charts{},
	}
}

type Config struct {
	UpdateEvery    int `yaml:"update_every,omitempty" json:"update_every"`
	web.HTTPConfig `yaml:",inline" json:""`
	Vnode          string `yaml:"vnode,omitempty" json:"vnode"`
}

type (
	Collector struct {
		module.Base
		Config `yaml:",inline" json:""`

		charts *module.Charts

		prom prometheus.Prometheus

		cache cache
	}
	cache struct {
		cores                       map[string]bool
		volumes                     map[string]bool
		nics                        map[string]bool
		thermalZones                map[string]bool
		processes                   map[string]bool
		iis                         map[string]bool
		adcs                        map[string]bool
		mssqlInstances              map[string]bool
		mssqlDBs                    map[string]bool
		services                    map[string]bool
		netFrameworkCLRExceptions   map[string]bool
		netFrameworkCLRInterops     map[string]bool
		netFrameworkCLRJIT          map[string]bool
		netFrameworkCLRLoading      map[string]bool
		netFrameworkCLRLocksThreads map[string]bool
		netFrameworkCLRMemory       map[string]bool
		netFrameworkCLRRemoting     map[string]bool
		netFrameworkCLRSecurity     map[string]bool
		collectors                  map[string]bool
		collection                  map[string]bool
		exchangeWorkload            map[string]bool
		exchangeLDAP                map[string]bool
		exchangeHTTPProxy           map[string]bool
		hypervVMMem                 map[string]bool
		hypervVMDevices             map[string]bool
		hypervVMInterfaces          map[string]bool
		hypervVswitch               map[string]bool
	}
)

func (c *Collector) Configuration() any {
	return c.Config
}

func (c *Collector) Init(context.Context) error {
	if err := c.validateConfig(); err != nil {
		return fmt.Errorf("config validation: %v", err)
	}

	prom, err := c.initPrometheusClient()
	if err != nil {
		return fmt.Errorf("init prometheus clients: %v", err)
	}
	c.prom = prom

	return nil
}

func (c *Collector) Check(context.Context) error {
	mx, err := c.collect()
	if err != nil {
		return err
	}
	if len(mx) == 0 {
		return errors.New("no metrics collected")
	}
	return nil
}

func (c *Collector) Charts() *module.Charts {
	return c.charts
}

func (c *Collector) Collect(context.Context) map[string]int64 {
	ms, err := c.collect()
	if err != nil {
		c.Error(err)
	}

	if len(ms) == 0 {
		return nil
	}
	return ms
}

func (c *Collector) Cleanup(context.Context) {
	if c.prom != nil && c.prom.HTTPClient() != nil {
		c.prom.HTTPClient().CloseIdleConnections()
	}
}
