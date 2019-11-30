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

	"github.com/brunotm/replicant/log"
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
	err = d.vm.Set("replicant_log", func(call otto.FunctionCall) otto.Value {
		log.Info(call.Argument(0).String()).String("driver", "javascript").Log()
		return otto.Value{}
	})
	if err != nil {
		return nil, fmt.Errorf("driver/javascript: error setting replicant_log: %w", err)
	}

	// add sleep to js vm
	err = d.vm.Set("replicant_sleep", func(call otto.FunctionCall) otto.Value {
		ms, _ := call.Argument(0).ToInteger()
		log.Debug("sleeping").String("driver", "javascript").Int("milliseconds", ms).Log()
		time.Sleep(time.Duration(ms) * time.Millisecond)
		return otto.Value{}
	})
	if err != nil {
		return nil, fmt.Errorf("driver/javascript: error setting replicant_sleep: %w", err)
	}

	// add http request capabilities to js vm
	err = d.vm.Set("replicant_http_do", func(call otto.FunctionCall) otto.Value {
		jsonHRO := call.Argument(0).String()
		if jsonHRO == "undefined" {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("no http request was specified")})
			return r
		}
		hro := httpRequest{}

		if err := json.Unmarshal([]byte(jsonHRO), &hro); err != nil {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("error deserializing request: %w", err)})
			return r
		}

		// handle form data if specified
		formData := url.Values{}
		for k, v := range hro.FormData {
			formData.Set(k, v)
		}

		// handle request body if specified
		// if both are specified request body have precedence
		var body io.Reader
		if hro.Body != "" {
			body = strings.NewReader(hro.Body)
		} else if len(hro.FormData) > 0 {
			body = strings.NewReader(formData.Encode())
		}

		u, err := url.ParseRequestURI(hro.URL)
		if err != nil {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("error parsing request url: %w", err)})
			return r
		}

		// handle url query parameters
		if len(hro.Params) > 0 {
			q, _ := url.ParseQuery(u.RawQuery)
			for k, v := range hro.Params {
				q.Add(k, v)
			}
			u.RawQuery = q.Encode()
		}

		req, err := http.NewRequest(hro.Method, u.String(), body)
		if err != nil {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("error creating http request: %w", err)})
			return r
		}
		req.Close = true

		for k, v := range hro.Header {
			req.Header.Set(k, v)
		}

		tr := &http.Transport{TLSClientConfig: &tls.Config{}}
		if hro.SSLSkipVerify {
			tr.TLSClientConfig.InsecureSkipVerify = true
		}

		client := &http.Client{Transport: tr}
		defer client.CloseIdleConnections()

		resp, err := client.Do(req)
		log.Debug("http request").String("url", hro.URL).
			String("method", hro.Method).Bool("skip_ssl_verify", hro.SSLSkipVerify).
			String("status", resp.Status).Error("error", err).Log()

		if err != nil {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("error performing request: %w", err)})
			return r
		}
		defer resp.Body.Close()

		jsResp := httpResponse{}
		jsResp.Status = resp.Status
		jsResp.StatusCode = resp.StatusCode
		jsResp.Protocol = resp.Request.Proto
		jsResp.Header = make(map[string]string)
		for k, v := range resp.Header {
			jsResp.Header[k] = v[0]
		}

		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			r, _ := d.toJSvalue(&httpResponse{Error: fmt.Errorf("error reading response body: %w", err)})
			return r
		}
		jsResp.Body = string(b)

		r, _ := d.toJSvalue(jsResp)
		return r
	})
	if err != nil {
		return nil, fmt.Errorf("driver/javascript: error setting replicant_http_do: %w", err)
	}

	// load replicant javascript utils
	if _, err = d.vm.Run(replicantJS); err != nil {
		return nil, fmt.Errorf("driver/javascript: error initializing replicant core objects: %w", err)
	}

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
		return nil, fmt.Errorf("driver/javascript: error initializing transaction script: %w", err)
	}

	if config.CallBack != nil {
		if _, err = txn.vm.Run(config.CallBack.Script); err != nil {
			return nil, fmt.Errorf("driver/javascript: error initializing callback handling script: %w", err)
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

type httpRequest struct {
	URL           string            `json:"URL"`
	Method        string            `json:"Method"`
	Body          string            `json:"Body"`
	Header        map[string]string `json:"Header"`
	Params        map[string]string `json:"Params"`
	FormData      map[string]string `json:"FormData"`
	SSLSkipVerify bool              `json:"SSLSkipVerify"`
}
type httpResponse struct {
	Status     string            `json:"Status"`
	StatusCode int               `json:"StatusCode"`
	Protocol   string            `json:"Protocol"`
	Body       string            `json:"Body"`
	Header     map[string]string `json:"Header"`
	Error      error             `json:"Error"`
}

type jsResult struct {
	Data    string
	Failed  bool
	Message string
}

// MarshalJSON is custom marshaler for result
func (r *httpResponse) MarshalJSON() (data []byte, err error) {
	var stringError string
	if r.Error != nil {
		stringError = r.Error.Error()
	}

	type alias httpResponse
	return json.Marshal(&struct {
		*alias
		Error string `json:"error"`
	}{
		alias: (*alias)(r),
		Error: stringError,
	})
}

// UnmarshalJSON is custom unmarshaler for result
func (r *httpResponse) UnmarshalJSON(data []byte) error {
	type alias httpResponse
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

const replicantJS = `
var replicant = {};
replicant.http = {};

replicant.Log = function (message) {
	replicant_log(message)
};

replicant.NewResponse = function () {
	return {
		Data: "",
		Message: "",
		Failed: false,
		JSON: function () {
			return JSON.stringify(this);
		}
	};
};

replicant.http.NewRequest = function () {
	return {
		URL: "",
		Method: "",
		Body: "",
		Header: {},
		Params: {},
		FormData: {},
		SSLSkipVerify: false,
		JSON: function () {
			return JSON.stringify(this)
		}
	};
};

replicant.http.Do = function (request) {
	resp = replicant_http_do(request.JSON())
	r = JSON.parse(resp);
	r.JSON = function () {
		return JSON.stringify(this);
	};
	return r;
};

replicant.Sleep = function(milliseconds) {
  replicant_sleep(milliseconds)
};
`
