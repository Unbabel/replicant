package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/Unbabel/replicant/transaction"
	"github.com/Unbabel/replicant/transaction/callback"
)

// TODO: find a proper way of pipelining callbacks
type callbackProxy struct {
	uuid         string
	config       transaction.Config
	client       *http.Client
	serverURL    string
	advertiseURL string
}

func (c *callbackProxy) Listen(ctx context.Context) (h *callback.Handle, err error) {

	response := make(chan callback.Response, 1)

	h = &callback.Handle{}
	h.Response = response
	h.Address = c.advertiseURL + "/callback/" + c.uuid

	buf, err := json.Marshal(c.config)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost, c.serverURL+"/api/v1/callback/"+c.uuid, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}

	go func() {
		resp, err := c.client.Do(req)
		if err != nil {
			response <- callback.Response{Data: nil, Error: err}
			return
		}
		defer resp.Body.Close()

		buf, err := ioutil.ReadAll(resp.Body)
		response <- callback.Response{Data: buf, Error: err}
	}()

	return h, nil
}
