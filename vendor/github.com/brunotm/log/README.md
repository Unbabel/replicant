[![Build Status](https://travis-ci.org/brunotm/log.svg?branch=master)](https://travis-ci.org/brunotm/log)
[![Go Report Card](https://goreportcard.com/badge/github.com/brunotm/log)](https://goreportcard.com/report/github.com/brunotm/log)
[![GoDoc](https://godoc.org/github.com/brunotm/log?status.svg)](https://godoc.org/github.com/brunotm/log)


# log

A simple, leveled, fast, zero allocation, json structured logging package for Go.
Designed to make logging on the hot path dirt cheap, dependency free and my life easier.

It also supports nesting objects and arrays to create more complex log entries.

By default log entries with the same level and message within each second, will be sampled to cap CPU and I/O load under high logging activity.
This behaviour can be disabled by setting `Config.EnableSampling to false`

## Usage
```go
package main

import (
	"os"

	"github.com/brunotm/log"
)

func main() {
	config := log.DefaultConfig
	config.Level = log.DEBUG

	// New logger with added context
	l := log.New(os.Stderr, config).
		With(func(e log.Entry) {
			e.String("application", "myapp")
		})

	// Simple logging
	l.Info("info message").String("key", "value").Log()
	// {"level":"info","time":"2019-01-30T20:42:56.445Z","caller":"_local/main.go:21",
	// "application":"myapp","message":"info message","key":"value"}

	l.Warn("warn message").Bool("flag", false).Log()
	// {"level":"warn","time":"2019-01-30T20:42:56.446Z","caller":"_local/main.go:24",
	// "application":"myapp","message":"warn message","flag":false}

	l.Error("caught an error").String("error", "request error").Log()
	// {"level":"error","time":"2019-01-30T20:42:56.446Z","caller":"_local/main.go:27",
	// "application":"myapp","message":"caught an error","error":"request error"}

	// Create nested objects in log entry
	l.Debug("debug message").Object("request_data", func(o log.Object) {
		o.String("request_id", "4BA0D8B1-4ABA-4D70-A55C-3358667C058B").
			String("user_id", "3B1BA12B-68DF-4DB7-809B-1AC5D8AF663A").
			Float("value", 3.1415926535)
	}).Log()
	// {"level":"debug","time":"2019-01-30T22:44:45.193Z","caller":"_local/main.go:31",
	// "application":"myapp","message":"debug message","request_data":
	// {"request_id":"4BA0D8B1-4ABA-4D70-A55C-3358667C058B",
	// "user_id":"3B1BA12B-68DF-4DB7-809B-1AC5D8AF663A","value":3.1415926535}}

	// Create array objects in log entry
	l.Debug("debug message").Array("request_points", func(a log.Array) {
		a.AppendFloat(3.1415926535).
			AppendFloat(2.7182818284).
			AppendFloat(1.41421).
			AppendFloat(1.6180339887498948482)
	}).Log()
	// {"level":"debug","time":"2019-02-04T08:42:15.216Z","caller":"_local/main.go:44",
	// "application":"myapp","message":"debug message",
	// "request_points":[3.1415926535,2.7182818284,1.41421,1.618033988749895]}
}
```

## Performance on a 2,3 GHz Intel Core i5, 2017 13-inch Macbook Pro
Message: `{"level":"info","time":"2019-01-30T20:54:07.029Z","message":"informational message","string value":"text","int value":8,"float":722727272.0099,"int":8,"float value":722727272.0099}`

```
pkg: github.com/brunotm/log
BenchmarkLogNoSampling-4          2000000               860 ns/op               0 B/op          0 allocs/op
BenchmarkLogWithSampling-4       20000000               115 ns/op               0 B/op          0 allocs/op
BenchmarkLogNoLevel-4           500000000                3.68 ns/op             0 B/op          0 allocs/op
```

## Contact
Bruno Moura [brunotm@gmail.com](mailto:brunotm@gmail.com)

## License
log source code is available under the Apache Version 2.0 [License](/LICENSE)