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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/MontFerret/ferret/pkg/runtime/logging"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
)

// Transaction is a pre-compiled replicant transaction for web applications
type Transaction struct {
	config       transaction.Config
	server       *url.URL
	timeout      time.Duration
	program      *runtime.Program
	proxied      bool
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
		result.Message = "failed to handle cdp url"
		result.DurationSeconds = time.Since(result.Time).Seconds()
		return result
	}

	switch t.proxied {
	case false:
		log.Debug("driver/web: chromedp server").String("address", cdpAddr).Log()
		drv := cdp.NewDriver(cdp.WithAddress(cdpAddr))
		defer drv.Close()

		ctx = drivers.WithContext(ctx, drv, drivers.AsDefault())

		// runtime.WithLog runtime.WithLogFields runtime.WithLogLevel
		r, err := t.program.Run(ctx, runtime.WithLogLevel(logging.ErrorLevel))

		if err != nil {
			result.Error = fmt.Errorf("driver/web: error running transaction script: %w", err)
			result.Failed = true
		}

		if len(r) == 0 {
			break
		}

		if err = json.Unmarshal(r, &result); err != nil {
			result.Error = fmt.Errorf("driver/web: error deserializing result data: %w", err)
			result.Failed = true
			result.Data = string(r)
		}

	case true:
		log.Debug("driver/web: chromedp proxied server").String("address", cdpAddr).Log()
		// TODO handle errors
		buf, _ := json.Marshal(t.config)
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, cdpAddr, bytes.NewReader(buf))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			result.Error = err
			result.Failed = true
			break
		}
		defer resp.Body.Close()

		buf, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			result.Error = fmt.Errorf("driver/web: error reading proxied result data: %w", err)
			result.Failed = true
			break
		}

		if err = json.Unmarshal(buf, &result); err != nil {
			result.Error = fmt.Errorf("driver/web: error deserializing proxied result data: %w", err)
			result.Failed = true
			result.Data = string(buf)
		}

	}

	result.DurationSeconds = time.Since(result.Time).Seconds()
	return result
}

// resolveAddr parses the cdp URL and converts the hostname into an ip address
// to ensure we talk with the same cdp server during the whole transaction lifecycle
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
		return "", fmt.Errorf("error resolving cdp_server name: %w", err)
	}

	if len(ips) == 0 {
		return "", fmt.Errorf("could not resolve hostname in %s", serverHostname)
	}

	return strings.Replace(serverURL, serverHostname, ips[0].String(), 1), nil
}
