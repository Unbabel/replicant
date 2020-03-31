package callback

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
	"sync"
)

var (
	listeners    map[string]Listener
	listenersMtx sync.RWMutex
)

func init() {
	listeners = make(map[string]Listener)
}

// Config configuration for receiving async transaction responses
type Config struct {
	Type   string `json:"type" yaml:"type"`
	Script string `json:"script" yaml:"script"`
}

// Listener listen for async transaction responses.
// Listeners must be safe for concurrent usage.
type Listener interface {
	Listen(ctx context.Context) (handle *Handle, err error)
}

// Handle is a registered async response handle
type Handle struct {
	UUID     string
	Address  string
	Response <-chan Response
}

// Response from an async call
type Response struct {
	Data  []byte
	Error error
}

// Register register listeners for callback async responses
func Register(typ string, listener Listener) (err error) {
	listenersMtx.Lock()
	defer listenersMtx.Unlock()

	if _, ok := listeners[typ]; ok {
		return fmt.Errorf("callback: duplicate listener for type %s", typ)
	}

	listeners[typ] = listener
	return nil
}

// GetListener gets an previously registered handler
func GetListener(typ string) (listener Listener, err error) {
	listenersMtx.Lock()
	defer listenersMtx.Unlock()

	handler, ok := listeners[typ]
	if !ok {
		return nil, fmt.Errorf("callback: no registered listener for type %s", typ)
	}

	return handler, nil
}
