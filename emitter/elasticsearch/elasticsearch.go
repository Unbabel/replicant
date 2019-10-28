package elasticsearch

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
	"context"
	"crypto/tls"
	"math/rand"
	"net/http"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/olivere/elastic/v7"
	"github.com/brunotm/replicant/replicant/transaction"
)

type Config struct {
	Username          string
	Password          string
	Urls              []string
	Index             string
	MaxPendingBytes   int64
	MaxPendingResults int64
	MaxPendingTime    time.Duration
}

// Emitter for elasticsearch
type Emitter struct {
	client        *elastic.Client
	index         string
	entropy       *ulid.MonotonicEntropy
	bulkProcessor *elastic.BulkProcessor
}

// Close and flush pending data
func (e *Emitter) Close() {
	e.bulkProcessor.Flush()
	e.bulkProcessor.Close()
	e.client.Stop()
}

// Emit results
func (e *Emitter) Emit(result transaction.Result) {
	id, _ := ulid.New(ulid.Timestamp(result.Time), e.entropy)

	req := elastic.NewBulkIndexRequest().
		OpType("create").
		Index(e.index).
		Id(id.String()).
		Doc(result)

	e.bulkProcessor.Add(req)

}

// New creates a new transaction.Result emmiter
func New(config Config) (emitter *Emitter, err error) {
	emitter = &Emitter{}
	var opts []elastic.ClientOptionFunc

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	opts = append(opts, elastic.SetURL(config.Urls...), elastic.SetHttpClient(client))

	if config.Username != "" && config.Password != "" {
		opts = append(opts, elastic.SetBasicAuth(config.Username, config.Password))
	}

	emitter.client, err = elastic.NewSimpleClient(opts...)
	if err != nil {
		return nil, err
	}

	emitter.index = config.Index

	// emitter.bulkProcessor =
	bps := emitter.client.BulkProcessor()
	if config.MaxPendingBytes > 0 {
		bps = bps.BulkSize(int(config.MaxPendingBytes))
	}

	if config.MaxPendingResults > 0 {
		bps = bps.BulkActions(int(config.MaxPendingResults))
	}

	if config.MaxPendingTime > 0 {
		bps = bps.FlushInterval(config.MaxPendingTime)
	}

	emitter.bulkProcessor, err = bps.Do(context.Background())
	if err != nil {
		return nil, err
	}

	emitter.bulkProcessor.Start(context.Background())
	emitter.entropy = ulid.Monotonic(rand.New(rand.NewSource(time.Now().UnixNano())), 0)

	return emitter, nil
}
