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

	"github.com/MontFerret/ferret/pkg/drivers"
	"github.com/MontFerret/ferret/pkg/drivers/cdp"
	"github.com/MontFerret/ferret/pkg/runtime"
	"github.com/MontFerret/ferret/pkg/runtime/logging"
	"github.com/Unbabel/replicant/transaction"
)

// Transaction is a pre-compiled replicant transaction for web applications
type Transaction struct {
	driver  *Driver
	program *runtime.Program
	config  transaction.Config
}

// Config returns the transaction config
func (t *Transaction) Config() (config transaction.Config) {
	return t.config
}

// Run executes the web transaction
func (t *Transaction) Run(ctx context.Context) (result transaction.Result) {
	t.driver.m.RLock()
	defer t.driver.m.RUnlock()

	var err error
	result.Name = t.config.Name
	result.Driver = "web"
	result.Metadata = t.config.Metadata

	// handle browserless mode for testing
	var drv *cdp.Driver
	switch t.driver.config.testing {
	case false:
		drv = cdp.NewDriver(cdp.WithAddress(t.driver.config.ServerURL))
	case true:
		drv = cdp.NewDriver()
	}
	defer drv.Close()

	ctx = drivers.WithContext(ctx, drv, drivers.AsDefault())

	// runtime.WithLog runtime.WithLogFields runtime.WithLogLevel
	r, err := t.program.Run(ctx, runtime.WithLogLevel(logging.ErrorLevel))

	if err != nil {
		result.Error = fmt.Errorf("driver/web: error running transaction script: %w", err)
		result.Failed = true
	}

	if len(r) != 0 {
		if err = json.Unmarshal(r, &result); err != nil {
			result.Error = fmt.Errorf("driver/web: error deserializing result data: %w", err)
			result.Failed = true
			result.Data = string(r)
		}
	}

	return result
}
