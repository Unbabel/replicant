// Package transaction implements the main types for defining, running and emitting
// replicant test data.
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
	"context"

	"github.com/Unbabel/replicant/transaction/callback"
)

// Transaction is a test executor. It can be run many times.
type Transaction interface {
	Run(ctx context.Context) (result Result)
	Config() (config Config)
}

// Config is a synthetic test definition
type Config struct {
	Name       string                 `json:"name" yaml:"name"`
	Driver     string                 `json:"driver" yaml:"driver"`
	Schedule   string                 `json:"schedule" yaml:"schedule"`
	Timeout    string                 `json:"timeout" yaml:"timeout"`
	RetryCount int                    `json:"retry_count" yaml:"retry_count"`
	Script     string                 `json:"script" yaml:"script"`
	CallBack   *callback.Config       `json:"callback" yaml:"callback"`
	Inputs     map[string]interface{} `json:"inputs" yaml:"inputs"`
	Metadata   map[string]string      `json:"metadata" yaml:"metadata"`
	Binary     []byte                 `json:"binary" yaml:"binary"`
}
