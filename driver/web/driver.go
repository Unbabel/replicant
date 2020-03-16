// Package web implements a web transaction driver based on chromedp/FQL.
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
	"net"
	"net/url"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/MontFerret/ferret/pkg/compiler"
	"github.com/Unbabel/replicant/log"
	"github.com/Unbabel/replicant/transaction"
)

// Driver for web based transactions using the chrome developer protocol
type Driver struct {
	m      sync.RWMutex
	cmd    *exec.Cmd
	close  chan struct{}
	config Config
}

// Config for web driver
type Config struct {
	// Server URL for the chrome developer protocol to be used for tests
	// can be overridden by the "cdp_server_url" in the transaction config inputs
	ServerURL string `json:"server_url" yaml:"server_url"`

	// Perform DNS discovery with the server hostname
	// The web driver needs to maintain the same cdp server across multiple http
	// requests. This needed when using multiple cdp servers that are load balanced
	// by DNS round robin, like a kubernetes headless service.
	DNSDiscovery bool `json:"dns_discovery" yaml:"dns_discovery"`

	// Path to chrome binary
	BinaryPath string `json:"binary_path" yaml:"binary_path"`

	// Arguments for launching chrome
	BinaryArgs []string `json:"binary_args" yaml:"binary_args"`

	// Interval for recycling chrome processes
	RecycleInterval time.Duration

	// for testing only
	testing bool
}

// New creates a new web driver
func New(c Config) (d *Driver, err error) {
	if c.ServerURL == "" && c.BinaryPath == "" {
		return nil, fmt.Errorf("driver/web: no chrome binary or server URL specified")
	}

	_, err = url.Parse(c.ServerURL)
	if err != nil {
		return nil, fmt.Errorf("driver/web: could not parse chrome server url: %w", err)
	}

	d = &Driver{config: c, close: make(chan struct{})}

	if d.config.BinaryPath != "" {
		if err = d.monitor(); err != nil {
			return nil, err
		}
	}

	return d, nil
}

// Type returns this driver type
func (d *Driver) Type() (t string) {
	return "web"
}

// New creates a web transaction
func (d *Driver) New(config transaction.Config) (tx transaction.Transaction, err error) {
	txn := &Transaction{}
	txn.driver = d

	txn.program, err = compiler.New().Compile(config.Script)
	if err != nil {
		return nil, fmt.Errorf("driver/web: error compiling transaction script: %w", err)
	}

	txn.config = config
	return txn, nil
}

// monitor the running chrome process to service transactions and recycle at every interval
func (d *Driver) monitor() (err error) {

	address := strings.Replace(d.config.ServerURL, "http://", "", 1)

	// start chrome process and set process group id to avoid
	// leaving zombies upon termination
	d.cmd = exec.Command(d.config.BinaryPath, d.config.BinaryArgs...)
	d.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := d.cmd.Start(); err != nil {
		return fmt.Errorf("driver/web: error starting chrome process: %w", err)
	}

	if err := waitForConn(address, 5, time.Second); err != nil {
		panic(fmt.Errorf("driver/web: %w", err))
	}

	log.Info("chrome process created").Int("pid", int64(d.cmd.Process.Pid)).Log()

	// TODO: don't panic and figure a better way to surface the errors below.
	go func() {
		for {
			select {
			case <-time.After(d.config.RecycleInterval):

				log.Info("recycling chrome process").Int("pid", int64(d.cmd.Process.Pid)).Log()
				d.m.Lock()

				// stop chrome process and its children
				err := syscall.Kill(-d.cmd.Process.Pid, syscall.SIGKILL)
				if err != nil {
					panic(fmt.Errorf("driver/web: error stopping chrome process: %w", err))
				}

				// start chrome process and set process group id to avoid
				// leaving zombies upon termination
				d.cmd = exec.Command(d.config.BinaryPath, d.config.BinaryArgs...)
				d.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
				if err := d.cmd.Start(); err != nil {
					panic(fmt.Errorf("driver/web: error starting chrome process: %w", err))
				}

				if err := waitForConn(address, 5, time.Second); err != nil {
					panic(fmt.Errorf("driver/web: %w", err))
				}

				d.m.Unlock()
				log.Info("chrome process created").Int("pid", int64(d.cmd.Process.Pid)).Log()

			case <-d.close:
				return
			}
		}
	}()

	return nil
}

// waitForConn waits for a successful TCP connection to the specified address
// for the given number of retries beetween the given interval.
func waitForConn(address string, retries int, interval time.Duration) (err error) {
	for x := 0; x < retries; x++ {
		log.Debug("driver/web: checking managed chrome process availability").Log()

		time.Sleep(interval)

		var conn net.Conn
		conn, err = net.Dial("tcp", address)
		if err == nil {
			log.Debug("driver/web: successfully connected to chrome process").Log()
			conn.Close()
			return nil
		}
	}

	return fmt.Errorf("error connecting to managed chrome process: %w", err)
}
