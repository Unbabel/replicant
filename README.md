# ![Replicant](https://raw.githubusercontent.com/brunotm/replicant/master/doc/logo.png)

[![Go Report Card](https://goreportcard.com/badge/github.com/brunotm/replicant?style=flat-square)](https://goreportcard.com/report/github.com/brunotm/replicant)
[![GoDoc](https://img.shields.io/badge/api-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/brunotm/replicant)
[![Docker Cloud Automated build](https://img.shields.io/docker/cloud/automated/brunotm/replicant?style=flat-square)](https://hub.docker.com/r/brunotm/replicant)

Replicant is a synthetic transaction execution framework named after the bioengineered androids from Blade Runner. (all synthetics came from Blade Runner :)

It defines a common interface for transactions and results, provides a transaction manager, execution scheduler, api and facilities for emitting result data to external systems.

## Status

***Under heavy development and API changes are expected. Please file an issue if anything breaks.***

## Requirements

* Go 1.13
* External URL for API tests that require webhook based callbacks
* Chrome with remote debugging (CDP) either in headless mode or in foreground (useful for testing)

## Examples

## Running the replicant server locally with docker

Using [example config](https://github.com/brunotm/replicant/blob/master/example-config.yaml) from the project root dir.

```bash
docker stack deploy -c $PWD/docker-compose.yaml replicant
```

This will deploy the replicant server and 2 chrome-headless nodes for web tests, persisting data under /data.

### Web application testing (local development)

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

### API testing (local development)

API testing support is based on interpreted go code, [documentation](https://github.com/containous/yaegi).

#### Test definition (can be also in json format)

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

## Related Projects

* [Yaegi is Another Elegant Go Interpreter](https://github.com/containous/yaegi)
* [Ferret Declarative web scraping](https://github.com/MontFerret/ferret)

## Contact

Bruno Moura [brunotm@gmail.com](mailto:brunotm@gmail.com)

## License

Replicant source code is available under the Apache Version 2.0 [License](https://github.com/brunotm/replicant/blob/master/LICENSE)