package webhook

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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/julienschmidt/httprouter"
	"github.com/brunotm/replicant/replicant/transaction/callback"
)

// Listener implements a replicant callback.Listener for http webhooks
type Listener struct {
	mtx     sync.Mutex
	path    string
	url     string
	router  *httprouter.Router
	handles map[string]handle
}

// New creates a new webhook listener for async callback based responses to replicant transactions
func New(url, path string, router *httprouter.Router) (l *Listener) {
	l = &Listener{}
	l.url = url
	l.path = path
	l.router = router
	l.handles = make(map[string]handle)

	// The handler access the listener state of open dynamically allocated webhook endpoints
	router.Handle(http.MethodPost, path+"/:id", func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		l.mtx.Lock()
		defer l.mtx.Unlock()

		id := p.ByName("id")

		handle, ok := l.handles[id]
		if !ok {
			http.Error(w, "callback for id not found", http.StatusNotFound)
			return
		}

		buf, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "could not read request body", http.StatusBadRequest)
			return
		}

		// return response and cleanup resources
		handle.resp <- callback.Response{Data: buf}
		delete(l.handles, id)
		close(handle.done)
		close(handle.resp)

	})

	return l
}

// Listen creates a handle for webhook based callbacks
func (l *Listener) Listen(ctx context.Context, id string) (h *callback.Handle, err error) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	if _, ok := l.handles[id]; ok {
		return nil, errors.New("callback for id already exists")
	}

	whandle := handle{}
	whandle.resp = make(chan callback.Response, 1)
	whandle.done = make(chan struct{})
	l.handles[id] = whandle

	// callback address
	address := fmt.Sprintf("%s%s/%s", l.url, l.path, id)

	// Each registered webhook has a monitor goroutine for cancellation
	go func() {

		select {
		case <-ctx.Done():
			l.mtx.Lock()
			defer l.mtx.Unlock()
			if whcb, ok := l.handles[id]; ok {
				whcb.resp <- callback.Response{
					Error: fmt.Errorf("timeout waiting for webhook response on %s", address)}
				delete(l.handles, id)
				close(whandle.resp)
				close(whandle.done)
				return
			}

		// If response received
		case <-whandle.done:
			return
		}
	}()

	return &callback.Handle{Address: address, Response: whandle.resp}, nil
}

type handle struct {
	done chan struct{}
	resp chan callback.Response
}
