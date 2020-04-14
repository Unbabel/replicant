// Package gd implements a Go transaction driver.
package gbd

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
	"fmt"
	"plugin"
	"time"

	"github.com/Unbabel/replicant/driver"
	"github.com/Unbabel/replicant/transaction"
	"github.com/Unbabel/replicant/volume"
)

// Driver for Go binary based transactions
type Driver struct {
	volume *volume.Volume
}

// New creates a new Go driver for binaries
func New(volume *volume.Volume) (d driver.Driver, err error) {
	return &Driver{volume: volume}, nil
}

// Type returns this driver type
func (d *Driver) Type() (t string) {
	return "go_binary"
}

// New creates a web transaction
func (d *Driver) New(config transaction.Config) (tx transaction.Transaction, err error) {
	txn := &Transaction{}

	if config.Timeout != "" {
		txn.timeout, err = time.ParseDuration(config.Timeout)
		if err != nil {
			return nil, fmt.Errorf("driver/go_binary: error parsing timeout: %w", err)
		}
	}

	location, err := d.volume.Location(config.Name + ".so")
	if err != nil {
		return nil, fmt.Errorf("driver/go_binary: %w", err)
	}

	p, err := plugin.Open(location)
	if err != nil {
		return nil, fmt.Errorf("driver/go_binary: error loading the binary: %w", err)
	}

	symbol, err := p.Lookup("Run")
	if err != nil {
		return nil, fmt.Errorf("driver/go_binary: error locating the entrypoint of the binary: %w", err)
	}

	var ok bool
	txn.transaction, ok = symbol.(func(context.Context) (message, data string, err error))
	if !ok || txn.transaction == nil {
		return nil, fmt.Errorf(
			`driver/go_binary: entrypoint Run doesn't implement "func(context.Context) (message, data string, err error)" signature`)
	}

	txn.config = config
	return txn, nil
}
