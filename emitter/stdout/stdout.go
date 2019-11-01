package stdout

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
	"encoding/json"
	"fmt"

	"github.com/brunotm/replicant/transaction"
)

// Config options for stdout emitter
type Config struct {
	Pretty bool `json:"pretty" yaml:"pretty"`
}

// Emitter stdout
type Emitter struct {
	config Config
}

// New creates a new emitter
func New(c Config) (e *Emitter) {
	return &Emitter{config: c}
}

// Emit results
func (e *Emitter) Emit(result transaction.Result) {
	var buf []byte

	switch e.config.Pretty {
	case false:
		buf, _ = json.Marshal(&result)
	case true:
		buf, _ = json.MarshalIndent(&result, "", "  ")
	}

	fmt.Printf("%s\n", buf)
}
