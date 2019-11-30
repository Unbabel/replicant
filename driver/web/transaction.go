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
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/MontFerret/ferret/pkg/runtime/logging"
	"github.com/brunotm/replicant/transaction"
)

// Transaction is a pre-compiled replicant transaction for web applications
type Transaction struct {
	config       transaction.Config
	server       *url.URL
	timeout      time.Duration
	program      *runtime.Program
	dnsDiscovery bool
}

// Config returns the transaction config
func (t *Transaction) Config() (config transaction.Config) {
	return t.config
}

// Run executes the web transaction
func (t *Transaction) Run(ctx context.Context) (result transaction.Result) {

	var err error

	if t.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t.timeout)
		defer cancel()
	}

	result.Name = t.config.Name
	result.Driver = "web"
	result.Time = time.Now()
	result.Metadata = t.config.Metadata

	cdpAddr, err := t.resolveAddr()
	if err != nil {
		result.Error = fmt.Errorf("driver/web: %w", err)
		result.Message = "failed to handle cpd url"
		return result
	}

	ctx = drivers.WithContext(
		ctx, cdp.NewDriver(cdp.WithAddress(cdpAddr)),
		drivers.AsDefault())

	// runtime.WithLog runtime.WithLogFields runtime.WithLogLevel
	r, err := t.program.Run(ctx, runtime.WithLogLevel(logging.ErrorLevel))
	result.DurationSeconds = time.Since(result.Time).Seconds()

	if err != nil {
		result.Error = fmt.Errorf("driver/web: error running transaction script: %w", err)
		result.Failed = true
	}

	if len(r) == 0 {
		return result
	}

	if err = json.Unmarshal(r, &result); err != nil {
		result.Error = fmt.Errorf("driver/web: error deserializing result data: %w", err)
		result.Data = string(r)
	}

	return result
}

// resolveAddr parses the cdp URL and converts the hostname into an ip address
// to ensure we talk with the same cpd server during the whole transaction lifecycle
// and avoid invalid sessions when dealing with DNS load balacing (e.g. headless k8s services)
func (t *Transaction) resolveAddr() (a string, err error) {
	serverURL := t.server.String()
	serverHostname := t.server.Hostname()

	if !t.dnsDiscovery {
		return serverURL, nil
	}

	// check if address is already an ip address
	if ip := net.ParseIP(serverHostname); ip != nil {
		return serverURL, nil
	}

	// resolve server ip addr
	ips, err := net.LookupIP(serverHostname)
	if err != nil {
		return "", fmt.Errorf("error resolving cpd_server name: %w", err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("could not resolve hostname in %s", serverHostname)
	}

	ip := ips[0]
	return strings.Replace(serverURL, serverHostname, ip.String(), 1), nil

}
