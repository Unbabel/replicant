package web

/*
   Copyright 2019 Bruno Moura <brunotm@gmail.com>

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

import (
	"fmt"
	"net/url"
	"time"

	"github.com/MontFerret/ferret/pkg/compiler"
	"github.com/brunotm/replicant/transaction"
)

// Driver for web based transactions using the chrome developer protocol
type Driver struct {
	config Config
}

// Config for web driver
type Config struct {
	// Server URL for the chrome developer protocol to be used for tests
	// can be overridden by the "cdp_server_url" in the transaction config inputs
	ServerURL string `json:"server_url" yaml:"server_url"`

	// Perform DNS discovery with the server hostname
	// The web driver needs to maintain the same cpd server across multiple http
	// requests. This needed when using multiple cdp servers that are load balanced
	// by DNS round robin, like a kubernetes headless service.
	DNSDiscovery bool `json:"dns_discovery" yaml:"dns_discovery"`
}

// New creates a new web driver
func New(c Config) (d *Driver) {
	return &Driver{c}
}

// Type returns this driver type
func (d *Driver) Type() (t string) {
	return "web"
}

// New creates a web transaction
func (d *Driver) New(config transaction.Config) (tx transaction.Transaction, err error) {
	txn := &Transaction{}

	serverURL := d.config.ServerURL
	s, ok := config.Inputs["cdp_address"]
	if ok {
		serverURL, ok = s.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected value for cdp_address: %#v", s)
		}
	}

	if serverURL == "" {
		return nil, fmt.Errorf("no default server and no input cpd_server specified")
	}

	txn.server, err = url.Parse(serverURL)
	if err != nil {
		return nil, fmt.Errorf("could not parse server URL: %w", err)
	}

	if config.Timeout != "" {
		txn.timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, err
		}
	}

	txn.dnsDiscovery = d.config.DNSDiscovery

	txn.program, err = compiler.New().Compile(config.Script)
	if err != nil {
		return nil, err
	}

	txn.config = config
	return txn, nil
}
