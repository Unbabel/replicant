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

// Array value
type Array struct {
	enc *encoder
}

// AppendBool value to array
func (a Array) AppendBool(value bool) (array Array) {
	a.enc.AppendBool(value)
	return a
}

// AppendFloat value to array
func (a Array) AppendFloat(value float64) (array Array) {
	a.enc.AppendFloat(value)
	return a
}

// AppendInt value to array
func (a Array) AppendInt(value int64) (array Array) {
	a.enc.AppendInt(value)
	return a
}

// AppendUint value to array
func (a Array) AppendUint(value uint64) (array Array) {
	a.enc.AppendUint(value)
	return a
}

// AppendString value to array
func (a Array) AppendString(value string) (array Array) {
	a.enc.AppendString(value)
	return a
}

// AppendNull value to array
func (a Array) AppendNull() (array Array) {
	a.enc.AppendBytes(nullBytes)
	return a
}

// Object creates a json object
func (a Array) Object(fn func(Object)) (array Array) {
	var o Object
	a.enc.openObject()
	o.enc = a.enc
	fn(o)
	a.enc.closeObject()
	return a
}

// Array creates a json array
func (a Array) Array(key string, fn func(Array)) (array Array) {
	a.enc.openArray()
	fn(a)
	a.enc.closeArray()
	return a
}
