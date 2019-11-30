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
	"encoding/json"
	"fmt"
	"time"
)

// Result represents a transaction execution result
type Result struct {
	UUID            string            `json:"uuid" yaml:"uuid"`
	Name            string            `json:"name" yaml:"name"`
	Driver          string            `json:"driver" yaml:"driver"`
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

// MarshalJSON is custom marshaler for result
func (r *Result) MarshalJSON() (data []byte, err error) {
	var stringError string
	if r.Error != nil {
		stringError = r.Error.Error()
	}

	type alias Result
	return json.Marshal(&struct {
		*alias
		Error string `json:"error"`
	}{
		alias: (*alias)(r),
		Error: stringError,
	})
}

// UnmarshalJSON is custom unmarshaler for result
func (r *Result) UnmarshalJSON(data []byte) error {
	type alias Result
	aux := &struct {
		*alias
		Error string `json:"error"`
	}{
		alias: (*alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	if aux.Error != "" {
		r.Error = fmt.Errorf(aux.Error)
	}

	return nil
}
