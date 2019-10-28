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

// Adapted from https://github.com/uber-go/zap/blob/master/zapcore/sampler.go

// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

import (
	"sync/atomic"
	"time"
)

const (
	countersPerLevel = 4096
)

type counter struct {
	resetAt int64
	counter uint64
}

func (c *counter) incCheckReset(t int64, tick time.Duration) uint64 {

	resetAfter := atomic.LoadInt64(&c.resetAt)
	if resetAfter > t {
		return atomic.AddUint64(&c.counter, 1)
	}

	atomic.StoreUint64(&c.counter, 1)

	newResetAfter := t + tick.Nanoseconds()
	if !atomic.CompareAndSwapInt64(&c.resetAt, resetAfter, newResetAfter) {
		// We raced with another goroutine trying to reset, and it also reset
		// the counter to 1, so we need to reincrement the counter.
		return atomic.AddUint64(&c.counter, 1)
	}

	return 1
}

type counters [maxLevel][countersPerLevel]counter

func (cs *counters) get(lvl Level, message string) *counter {
	i := lvl - 1
	j := fnv64a(message)%countersPerLevel - 1
	return &cs[i][j]
}

// Sample incoming to cap the CPU and I/O load of logging while attempting to
// preserve a representative subset of the logging activity for each level and message.
//
// Sample by logging the first N entries with a given level and message
// each tick. If more Entries with the same level and message are seen during
// the same interval, every Mth message is logged and the rest are dropped.
//
// Keep in mind that this sampling implementation is optimized for speed over
// absolute precision; under load, each tick may be slightly over- or
// under-sampled.
type sampler struct {
	counters counters
	tick     time.Duration
	start    uint64
	factor   uint64
}

func newSampler(tick time.Duration, start, factor int) (s *sampler) {
	return &sampler{
		tick:     tick,
		counters: counters{},
		start:    uint64(start),
		factor:   uint64(factor),
	}
}

func (s *sampler) check(lvl Level, msg string) (ok bool) {
	counter := s.counters.get(lvl, msg)
	n := counter.incCheckReset(time.Now().UnixNano(), s.tick)
	if n > s.start && (n-s.start)%s.factor != 0 {
		return false
	}
	return true
}

// fnv64a, adapted from "hash/fnv"
func fnv64a(s string) uint64 {
	const (
		offset64 uint64 = 14695981039346656037
		prime64  uint64 = 1099511628211
	)

	hash := offset64
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= prime64
	}
	return hash
}
