# ![Replicant](https://raw.githubusercontent.com/Unbabel/replicant/master/doc/logo.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/Unbabel/replicant?style=flat-square)](https://goreportcard.com/report/github.com/Unbabel/replicant)
[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/Unbabel/replicant)
[![Docker Cloud Automated build](https://img.shields.io/docker/cloud/automated/unbabel/replicant?style=flat-square)](https://hub.docker.com/r/unbabel/replicant)

Replicant is a synthetic testing service named after the bioengineered androids from Blade Runner. (all synthetics came from Blade Runner :)

It allows web application testing using chromedp, and api application testing using Go or Javascript. Provides a test manager, execution scheduler, api and facilities for emitting result data to external systems.

## Status

***Under heavy development and API changes are expected. Please file an issue if anything breaks.***

## Requirements

* Go 1.13
* External URL for API tests that require webhook based callbacks
* Chrome with remote debugging (CDP) either in headless mode or in foreground (useful for testing)

## Examples

## Running the replicant server locally with docker

```bash
docker stack deploy -c $PWD/docker-compose.yaml replicant
```

This will deploy the replicant server and 2 replicant executor nodes for web tests, persisting data under /data.

### Web application testing

Web application testing support is based on the FQL (Ferret Query Language), [documentation](https://github.com/MontFerret/ferret).

#### Test definition (can be also in json format)

```yaml
POST http://127.0.0.1:8080/api/v1/run
content-type: application/yaml

name: duckduckgo-web-search
driver: web
schedule: '@every 60s'
timeout: 50s
retry_count: 2
inputs:
  url: "https://duckduckgo.com"
  user_agent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/76.0.3809.87 Safari/537.36"
  timeout: 5000000
  text: "blade runner"
metadata:
  transaction: website-search
  application: duckduckgo
  environment: production
  component: website
script: |
  LET doc = DOCUMENT('{{ index . "url" }}', { driver: "cdp", userAgent: "{{ index . "user_agent" }}"})
  INPUT(doc, '#search_form_input_homepage', "{{ index . "text" }}")
  CLICK(doc, '#search_button_homepage')
  WAIT_NAVIGATION(doc)
  LET result = ELEMENT(doc, '#r1-0 > div > div.result__snippet.js-result-snippet').innerText
  RETURN {
    failed: result == "",
    message: "search result",
    data: result,
  }
```

#### Response

```json
{
  "data": [
    {
      "uuid": "01DSSR5GH2BPX4G5FFCEVPEBKK",
      "name": "duckduckgo-web-search",
      "driver": "web",
      "failed": true,
      "message": "",
      "data": "",
      "time": "2019-11-16T09:19:39.554976Z",
      "metadata": {
        "application": "duckduckgo",
        "component": "website",
        "environment": "production",
        "transaction": "website-search"
      },
      "retry_count": 0,
      "with_callback": false,
      "duration_seconds": 6.967938203,
      "error": "operation timed out: WAIT_NAVIGATION(doc) at 4:0"
    }
  ]
}
```

### API testing

##### Using the javascript driver
The following API is exposed by the javascript driver in order to perform HTTP calls and logging:
* `replicant.Log(string)` log messages from the javascript test on the replicant server log.


* `replicant.NewResult()` create a new response object to be returned as a result of the test, which should be modified accordingly to reflect the test result. The response must be returned as a serialized JSON object by calling its bounded method `Response.JSON`, E.g. `return response.JSON()`.

Result type attributes:
```js
{
		Data: "",
		Message: "",
		Failed: false,
}
```

* `replicant.http.NewRequest()` creates a new HTTP request object for performing HTTP calls.

HttpRequest attributes:
```js
{
		URL: "",
		Method: "",
		Body: "",
		Header: {},
		Params: {},
		FormData: {},
		SSLSkipVerify: false,
```

* `replicant.http.Do(HttpRequest) performs a HTTP request and returns its response.

HttpResponse attributes:
```js
{
	Status: ""
	StatusCode: 200
	Protocol: ""
	Body: ""
	Header: {}
	Error: ""
}
```

#### Test definition (can be also in JSON format)

```yaml
POST http://127.0.0.1:8080/api/v1/run
content-type: application/yaml

name: duckduckgo-api-search
driver: javascript
schedule: '@every 60s'
timeout: 60s
retry_count: 2
inputs:
  url: "https://api.duckduckgo.com"
  text: "blade runner"
metadata:
  transaction: api-search
  application: duckduckgo
  environment: production
  component: api
script: |
  function Run(ctx) {
    req = replicant.http.NewRequest()
    req.URL = "{{ index . "url" }}"
    req.Params.q = "{{ index . "text" }}"
    req.Params.format = "json"
    req.Params.no_redirect = "1"
    resp = replicant.http.Do(req)
    data = JSON.parse(resp.Body)
    rr = replicant.NewResponse()
    switch(data.RelatedTopics && data.RelatedTopics.length > 0) {
      case true:
        rr.Data = data.RelatedTopics[0].Text
        rr.Message = resp.Status
        rr.Failed = false
        break
      case false:
        rr.Data = JSON.stringify(data)
        rr.Message = resp.Status
        rr.Failed = true
        break
    }
    return rr.JSON()
  }
```


##### Using the Go driver
Standard Go code can be used to create tests using following rules:
* The package name must be `transaction`
* The test function must implement the following signature: `func Run(ctx context.Context) (message string, data string, err error)`.

***Keep in mind that unlike the javascript driver which doesn't expose any I/O or lower level functionality for accessing the underlying OS, the Go driver currently exposes all of the Go standard library. Only use this driver if you are absolutely sure of what you are doing. This is planned to change in the future.***

#### Test definition (can be also in JSON format)
```yaml
POST http://127.0.0.1:8080/api/v1/run
content-type: application/yaml

name: duckduckgo-api-search
driver: go
schedule: '@every 60s'
timeout: 60s
retry_count: 2
inputs:
  url: "https://api.duckduckgo.com/"
  text: "blade runner"
metadata:
  transaction: api-search
  application: duckduckgo
  environment: production
  component: api
script: |
  package transaction
  import "bytes"
  import "context"
  import "fmt"
  import "net/http"
  import "io/ioutil"
  import "net/http"
  import "regexp"
  func Run(ctx context.Context) (m string, d string, err error) {
    req, err := http.NewRequest(http.MethodGet, "{{ index . "url" }}", nil)
      if err != nil {
        return "request build failed", "", err
    }
    req.Header.Add("Accept-Charset","utf-8")
    q := req.URL.Query()
    q.Add("q", "{{ index . "text" }}")
    q.Add("format", "json")
    q.Add("pretty", "1")
    q.Add("no_redirect", "1")
    req.URL.RawQuery = q.Encode()
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
      return "failed to send request", "", err
    }
    buf, err := ioutil.ReadAll(resp.Body)
    if err != nil {
      return "failed to read response", "", err
    }
    rx, err := regexp.Compile(`"Text"\s*:\s*"(.*?)"`)
    if err != nil {
      return "failed to compile regexp", "", err
    }
    s := rx.FindSubmatch(buf)
    if len(s) < 2 {
      return "failed to find data", "", fmt.Errorf("no match")
    }
    return "search result", fmt.Sprintf("%s", s[1]), nil
  }
```

#### Response

```json
{
  "data": [
    {
      "uuid": "01DSSR7ST5Q1Y2Y7HDSQDNS7Y7",
      "name": "duckduckgo-api-search",
      "driver": "go",
      "failed": false,
      "message": "search result",
      "data": "Blade Runner A 1982 American neo-noir science fiction film directed by Ridley Scott, written by Hampton...",
      "time": "2019-11-16T09:20:54.597852Z",
      "metadata": {
        "application": "duckduckgo",
        "component": "api",
        "environment": "production",
        "transaction": "api-search"
      },
      "retry_count": 0,
      "with_callback": false,
      "duration_seconds": 0.486582328,
      "error": ""
    }
  ]
}
```

## API

| Method | Resource              | Action                                                  |
|--------|-----------------------|---------------------------------------------------------|
| POST   | /v1/transaction       | Add a managed transaction                               |
| GET    | /v1/transaction       | Get all managed transaction definitions                 |
| GET    | /v1/transaction/:name | Get a managed transaction definition by name            |
| DELETE | /v1/transaction/:name | Remove a managed transaction                            |
| POST   | /v1/run               | Run an ad-hoc transaction                               |
| POST   | /v1/run/:name         | Run a managed transaction by name                       |
| GET    | /v1/result            | Get all managed transaction last execution results      |
| GET    | /v1/result/:name      | Get the latest result for a managed transaction by name |
| GET    | /metrics              | Get metrics (prometheus emitter must be enabled)        |
| GET    | /debug/pprof          | Get available runtime profile data (debug enabled)      |
| GET    | /debug/pprof/:profile | Get profile data (for pprof, debug enabled)             |

## TODO

* Tests
* Developer and user documentation
* Add support for more conventional persistent stores
* Vault integration for secrets (inputs)
* Architecture and API documentation
* Javascript driver transaction support


## Acknowledgements

* [Yaegi is Another Elegant Go Interpreter](https://github.com/containous/yaegi)
* [Ferret Declarative web scraping](https://github.com/MontFerret/ferret)
* [otto is a JavaScript parser and interpreter written natively in Go](https://github.com/robertkrimen/otto)

## Contact

Bruno Moura [brunotm@gmail.com](mailto:brunotm@gmail.com)

## License

Replicant source code is available under the Apache Version 2.0 [License](https://github.com/Unbabel/replicant/blob/master/LICENSE)
