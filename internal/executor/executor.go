// Package executor implements the replicant transaction execution service
package executor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Unbabel/replicant/driver"
	godriver "github.com/Unbabel/replicant/driver/go"
	"github.com/Unbabel/replicant/driver/javascript"
	"github.com/Unbabel/replicant/driver/web"
	"github.com/Unbabel/replicant/internal/xz"
	"github.com/Unbabel/replicant/transaction"
)

// Executor is the replicant execution service
type Executor struct {
	config  Config
	drivers *xz.Map
}

// Config for executor
type Config struct {
	Web          web.Config
	ServerURL    string
	AdvertiseURL string
}

// New creates a new executor
func New(c Config) (e *Executor, err error) {
	e = &Executor{}
	e.config = c
	e.drivers = xz.NewMap()

	var drv driver.Driver

	if c.Web.BinaryPath != "" || c.Web.ServerURL != "" {
		drv, err = web.New(c.Web)
		if err != nil {
			return nil, err
		}
		e.drivers.Store(drv.Type(), drv)
	}

	drv, err = javascript.New()
	if err != nil {
		return nil, err
	}
	e.drivers.Store(drv.Type(), drv)

	drv, err = godriver.New()
	if err != nil {
		return nil, err
	}
	e.drivers.Store(drv.Type(), drv)

	return e, err
}

// Run the given transaction
func (e *Executor) Run(uuid string, c transaction.Config) (r transaction.Result, err error) {
	d, ok := e.drivers.Load(c.Driver)
	if !ok {
		return r, fmt.Errorf("replicant-executor: driver %s not found", c.Driver)
	}

	drv := d.(driver.Driver)

	var ctx context.Context
	var cancel context.CancelFunc
	var tx transaction.Transaction

	if tx, err = drv.New(c); err != nil {
		return r, err
	}

	ctx = context.WithValue(context.Background(), "transaction_uuid", uuid)
	switch c.Timeout != "" {
	case true:
		timeout, err := time.ParseDuration(c.Timeout)
		if err != nil {
			return r, err
		}

		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()

	case false:
		ctx, cancel = context.WithCancel(ctx)
		defer cancel()
	}

	// Inject callback proxy
	if c.CallBack != nil {
		listener := &callbackProxy{}
		listener.uuid = uuid
		listener.client = &http.Client{}
		listener.config = c
		listener.serverURL = e.config.ServerURL
		listener.advertiseURL = e.config.AdvertiseURL
		ctx = context.WithValue(ctx, c.CallBack.Type, listener)
	}

	start := time.Now()
	result := tx.Run(ctx)
	result.UUID = uuid
	result.Time = start
	result.DurationSeconds = time.Since(result.Time).Seconds()
	result.Metadata = c.Metadata

	return r, nil
}
