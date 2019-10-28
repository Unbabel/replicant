package log

import "strconv"

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

const (
	hex = "0123456789abcdef"
)

type encoder struct {
	data  []byte
	index int64
}

func (e *encoder) checkComma() {
	if len(e.data) > 0 {

		switch e.data[len(e.data)-1] {
		case '{', '[', ':':
			return
		default:
			e.data = append(e.data, ',')
		}

	}
}

func (e *encoder) openObject() {
	e.checkComma()
	e.data = append(e.data, '{')
}

func (e *encoder) closeObject() {
	e.data = append(e.data, '}')
}

func (e *encoder) openArray() {
	e.checkComma()
	e.data = append(e.data, '[')
}

func (e *encoder) closeArray() {
	e.data = append(e.data, ']')
}

func (e *encoder) reset() {
	e.data = e.data[:0]
	e.index = -1
}

func (e *encoder) AppendBool(value bool) {
	e.checkComma()
	e.data = strconv.AppendBool(e.data, value)
}

func (e *encoder) AppendFloat(value float64) {
	e.checkComma()
	e.data = strconv.AppendFloat(e.data, value, 'f', -1, 64)
}

func (e *encoder) AppendInt(value int64) {
	e.checkComma()
	e.data = strconv.AppendInt(e.data, value, 10)
}

func (e *encoder) AppendUint(value uint64) {
	e.checkComma()
	e.data = strconv.AppendUint(e.data, value, 10)
}

func (e *encoder) AppendString(value string) {
	e.checkComma()
	e.writeString(value)
}

func (e *encoder) AppendBytes(value []byte) {
	e.checkComma()
	e.data = append(e.data, value...)
}

// based on https://golang.org/src/encoding/json/encode.go:884
func (e *encoder) writeString(s string) {
	e.data = append(e.data, '"')

	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 0x20 && c != '\\' && c != '"' {
			e.data = append(e.data, c)
			continue
		}
		switch c {
		case '"', '\\':
			e.data = append(e.data, '\\', '"')
		case '\n':
			e.data = append(e.data, '\\', '\n')
		case '\f':
			e.data = append(e.data, '\\', '\f')
		case '\b':
			e.data = append(e.data, '\\', '\b')
		case '\r':
			e.data = append(e.data, '\\', '\r')
		case '\t':
			e.data = append(e.data, '\\', '\t')
		default:
			e.data = append(e.data, `\u00`...)
			e.data = append(e.data, hex[c>>4], hex[c&0xF])
		}
		continue
	}

	e.data = append(e.data, '"')
}

func (e *encoder) addKey(key string) {
	e.checkComma()
	e.data = append(e.data, '"')
	e.data = append(e.data, key...)
	e.data = append(e.data, '"', ':')
}
