package log

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
	"runtime"
	"strconv"
	"strings"
	"time"
)

// Entry is a structured log entry. A entry is not safe for concurrent use.
// A entry must be logged by calling Log(), and cannot be reused after.
type Entry struct {
	o     Object
	l     *Logger
	level Level
}

func (e Entry) reset() {
	e.l = nil
	e.o.enc.reset()
}

// Log logs the current entry. An entry must not be used after calling Log.
func (e Entry) Log() {
	if e.o.enc != nil {
		e.o.enc.closeObject()
		e.l.write(e)
	}
}

// Discard the current entry without logging it.
// func (e Entry) discard() {
// 	if e.o.enc != nil {
// 		e.l.discard(e)
// 	}
// }

// Level returns the log level of current entry.
func (e Entry) Level() (level Level) {
	return e.level
}

// Bytes return the current entry bytes. This is intended to be used in hooks
// That will be applied after calling Log().
// The returned []byte is not a copy and must not be modified directly.
func (e Entry) Bytes() (data []byte) {
	return e.o.enc.data
}

// Bool adds the given bool key/value
func (e Entry) Bool(key string, value bool) (Entry Entry) {
	if e.o.enc != nil {
		e.o.Bool(key, value)
	}
	return e
}

// Float adds the given float key/value
func (e Entry) Float(key string, value float64) (entry Entry) {
	if e.o.enc != nil {
		e.o.Float(key, value)
	}
	return e
}

// Int adds the given int key/value
func (e Entry) Int(key string, value int64) (entry Entry) {
	if e.o.enc != nil {
		e.o.Int(key, value)
	}
	return e
}

// Uint adds the given uint key/value
func (e Entry) Uint(key string, value uint64) (entry Entry) {
	if e.o.enc != nil {
		e.o.Uint(key, value)
	}
	return e
}

// String adds the given string key/value
func (e Entry) String(key string, value string) (entry Entry) {
	if e.o.enc != nil {
		e.o.String(key, value)
	}
	return e
}

// Null adds a null value for the given key
func (e Entry) Null(key string) (entry Entry) {
	if e.o.enc != nil {
		e.o.Null(key)
	}
	return e
}

// Error adds the given error key/value
func (e Entry) Error(key string, value error) (entry Entry) {
	if e.o.enc != nil {
		e.o.Error(key, value)
	}
	return e
}

// Object creates a json object
func (e Entry) Object(key string, fn func(Object)) (entry Entry) {
	if e.o.enc != nil {
		e.o.Object(key, fn)
	}
	return e
}

// Array creates a json array
func (e Entry) Array(key string, fn func(Array)) (entry Entry) {
	if e.o.enc != nil {
		e.o.Array(key, fn)
	}
	return e
}

func (e Entry) init(level Level) {

	t := time.Now()
	e.level = level

	e.o.enc.openObject()
	e.o.enc.addKey("level")
	e.o.enc.AppendString(level.String())

	if e.l.config.EnableTime {
		e.o.enc.addKey(e.l.config.TimeField)

		switch e.l.config.TimeFormat {
		case Unix:
			e.o.enc.AppendInt(t.Unix())
		case UnixMilli:
			e.o.enc.AppendInt(t.UnixNano() / 1000000)
		case UnixNano:
			e.o.enc.AppendInt(t.UnixNano())
		default:
			e.o.enc.data = append(e.o.enc.data, '"')
			e.o.enc.data = t.AppendFormat(e.o.enc.data, e.l.config.TimeFormat)
			e.o.enc.data = append(e.o.enc.data, '"')
		}

	}

	if e.l.config.EnableCaller {
		_, f, l, ok := runtime.Caller(3 + e.l.config.CallerSkip)
		e.o.enc.addKey("caller")
		if ok {
			idx := strings.LastIndexByte(f, '/')
			idx = strings.LastIndexByte(f[:idx], '/')
			e.o.enc.AppendString(f[idx+1:] + ":" + strconv.Itoa(l))
		} else if !ok {
			e.o.enc.AppendString("???")
		}

	}
}
