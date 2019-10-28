package transaction

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
	"errors"
	"fmt"
	"sync"
	"text/template"
	"time"
)

var (
	// ErrInvalidType invalid transaction type
	ErrInvalidType = errors.New("invalid transaction type")
	// ErrTransactionNotFound transaction not found
	ErrTransactionNotFound = errors.New("transaction not found")

	suppliers    map[string]Supplier
	suppliersMtx sync.RWMutex
)

func init() {
	suppliers = make(map[string]Supplier)
}

// Register suppliers for transaction types
func Register(typ string, supplier Supplier) (err error) {
	suppliersMtx.Lock()
	defer suppliersMtx.Unlock()

	if _, ok := suppliers[typ]; ok {
		return fmt.Errorf("duplicate supplier for type %s", typ)
	}

	suppliers[typ] = supplier
	return nil
}

// New creates a transaction from a config
func New(config Config) (tx Transaction, err error) {
	suppliersMtx.RLock()
	supplier, ok := suppliers[config.Type]
	suppliersMtx.RUnlock()

	if !ok {
		return nil, ErrInvalidType
	}

	if config.Inputs != nil {
		if config, err = parseTemplate(config); err != nil {
			return nil, err
		}
	}

	return supplier(config)
}

// Supplier is a Transaction Supplier
type Supplier func(Config) (Transaction, error)

// Transaction is a test executor. It can be run many times.
type Transaction interface {
	Run(ctx context.Context) (result Result)
	Config() (config Config)
}

// Config is a synthetic test definition
type Config struct {
	Name       string                 `json:"name" yaml:"name"`
	Type       string                 `json:"type" yaml:"type"`
	Schedule   string                 `json:"schedule" yaml:"schedule"`
	Timeout    string                 `json:"timeout" yaml:"timeout"`
	RetryCount int                    `json:"retry_count" yaml:"retry_count"`
	Script     string                 `json:"script" yaml:"script"`
	CallBack   *CallBackConfig        `json:"callback" yaml:"callback"`
	Inputs     map[string]interface{} `json:"inputs" yaml:"inputs"`
	Metadata   map[string]string      `json:"metadata" yaml:"metadata"`
}

// CallBackConfig configuration for receiving async transaction responses
type CallBackConfig struct {
	Type   string `json:"type" yaml:"type"`
	Script string `json:"script" yaml:"script"`
}

// Result represents a transaction execution result
type Result struct {
	Name            string            `json:"name" yaml:"name"`
	Type            string            `json:"type" yaml:"type"`
	Failed          bool              `json:"failed" yaml:"failed"`
	Message         string            `json:"message" yaml:"message"`
	Data            string            `json:"data" yaml:"data"`
	Time            time.Time         `json:"time" yaml:"time"`
	Error           error             `json:"-" yaml:"-"`
	Metadata        map[string]string `json:"metadata" yaml:"metadata"`
	RetryCount      int               `json:"retry_count" yaml:"retry_count"`
	WithCallback    bool              `json:"with_callback" yaml:"with_callback"`
	DurationSeconds float64           `json:"duration_seconds" yaml:"duration_seconds"`
}

func parseTemplate(config Config) (c Config, err error) {

	tmpl, err := template.New(config.Name).Parse(config.Script)
	if err != nil {
		return config, err
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, config.Inputs)
	if err != nil {
		return config, err
	}

	config.Script = buf.String()
	return config, nil

}
