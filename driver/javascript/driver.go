package javascript

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brunotm/log"
	"github.com/brunotm/replicant/transaction"
	"github.com/robertkrimen/otto"
)

// Driver for javascript language based transactions
type Driver struct {
	vm *otto.Otto
}

// New creates a new javascript transaction driver
func New() (d *Driver, err error) {
	d = &Driver{}
	d.vm = otto.New()

	// add logging to js vm
	d.vm.Set("replicant_log", func(call otto.FunctionCall) otto.Value {
		log.Info(call.Argument(0).String()).String("driver", "javascript").Log()
		return otto.Value{}
	})

	if _, err = d.vm.Run(coreJS); err != nil {
		return nil, fmt.Errorf("error initializing VM core objects: %s", err)
	}

	// add http request capabilities to js vm
	d.vm.Set("replicant_http_do", func(call otto.FunctionCall) otto.Value {
		jsonHRO := call.Argument(0).String()
		if jsonHRO == "undefined" {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("no http request was specified")})
			return r
		}
		hro := JSHttpRequest{}

		if err := json.Unmarshal([]byte(jsonHRO), &hro); err != nil {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("error deserializing request")})
			return r
		}

		formData := url.Values{}
		for k, v := range hro.FormData {
			formData.Set(k, v)
		}

		var body io.Reader
		if hro.Body != "" {
			body = strings.NewReader(hro.Body)
		} else if len(hro.FormData) > 0 {
			body = strings.NewReader(formData.Encode())
		}

		u, err := url.ParseRequestURI(hro.URL)
		if err != nil {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("failed to parse request URL")})
			return r
		}

		if len(hro.Params) > 0 {
			q, _ := url.ParseQuery(u.RawQuery)
			for k, v := range hro.Params {
				q.Add(k, v)
			}
			u.RawQuery = q.Encode()
		}

		req, err := http.NewRequest(hro.Method, u.String(), body)
		if err != nil {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("failed to create HTTP request: %s", err)})
			return r
		}

		for k, v := range hro.Header {
			req.Header.Set(k, v)
		}
		req.Close = true

		tr := &http.Transport{TLSClientConfig: &tls.Config{}}
		if hro.SSLSkipVerify {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}

		client := &http.Client{Transport: tr}
		resp, err := client.Do(req)
		if err != nil {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("request failed: %s", err)})
			return r
		}

		jsResp := JSHttpResponse{}
		jsResp.Status = resp.Status
		jsResp.StatusCode = resp.StatusCode
		jsResp.Protocol = resp.Request.Proto
		jsResp.Header = make(map[string]string)
		for k, v := range resp.Header {
			jsResp.Header[k] = v[0]
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			r, _ := d.toJSvalue(&JSHttpResponse{Error: fmt.Errorf("unable to read response body: %s", err)})
			return r
		}
		jsResp.Body = string(b)

		r, _ := d.toJSvalue(jsResp)
		return r
	})

	return d, nil
}

// Type returns this driver type
func (d *Driver) Type() (t string) {
	return "javascript"
}

// New creates a new javascript transaction
func (d *Driver) New(config transaction.Config) (tx transaction.Transaction, err error) {
	txn := &Transaction{}
	txn.config = config

	txn.timeout, err = time.ParseDuration(config.Timeout)
	if err != nil {
		return nil, err
	}

	txn.vm = d.vm.Copy()
	if _, err = txn.vm.Run(config.Script); err != nil {
		return nil, fmt.Errorf("error initializing transaction script: %s", err)
	}

	if config.CallBack != nil {
		if _, err = txn.vm.Run(config.CallBack.Script); err != nil {
			return nil, fmt.Errorf("error initializing callback handling script: %s", err)
		}
	}

	return txn, nil
}

func (d *Driver) toJSvalue(v interface{}) (o otto.Value, err error) {
	b, err := json.Marshal(v)
	if err != nil {
		return o, err
	}

	return d.vm.ToValue(string(b))
}

type JSHttpRequest struct {
	URL           string            `json:"url"`
	Method        string            `json:"method"`
	Body          string            `json:"body"`
	Header        map[string]string `json:"header"`
	Params        map[string]string `json:"params"`
	FormData      map[string]string `json:"form_data"`
	SSLSkipVerify bool              `json:"ssl_skip_verify"`
}
type JSHttpResponse struct {
	Status     string            `json:"status"`
	StatusCode int               `json:"status_code"`
	Protocol   string            `json:"protocol"`
	Body       string            `json:"body"`
	Header     map[string]string `json:"header"`
	Error      error             `json:"error"`
}

// MarshalJSON is custom marshaler for result
func (r *JSHttpResponse) MarshalJSON() (data []byte, err error) {
	var stringError string
	if r.Error != nil {
		stringError = r.Error.Error()
	}

	type alias JSHttpResponse
	return json.Marshal(&struct {
		*alias
		Error string `json:"error"`
	}{
		alias: (*alias)(r),
		Error: stringError,
	})
}

// UnmarshalJSON is custom unmarshaler for result
func (r *JSHttpResponse) UnmarshalJSON(data []byte) error {
	type alias JSHttpResponse
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

const coreJS = `
var replicant = {};
replicant.http = {};

replicant.Log = function (message) {
	replicant_log(message)
};

replicant.NewResponse = function () {
	return {
		data: "",
		message: "",
		failed: false,
		json: function () {
			return JSON.stringify(this);
		}
	};
};

replicant.http.NewRequest = function () {
	return {
		url: "",
		method: "",
		body: "",
		header: {},
		params: {},
		form_data: {},
		ssl_skip_verify: false,
		json: function () {
			return JSON.stringify(this)
		}
	};
};

replicant.http.Do = function (request) {
	resp = replicant_http_do(request.json())
	r = JSON.parse(resp);
	r.json = function () {
		return JSON.stringify(this);
	};
	return r;
};
`
