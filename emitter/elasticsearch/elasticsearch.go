// Package elasticsearch implements a result emitter for elasticsearch.
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
	"fmt"
	"net/http"
	"time"

	"github.com/Unbabel/replicant/transaction"
	"github.com/olivere/elastic/v7"
)

// Config for elasticsearch emitter
type Config struct {
	Username          string        `json:"username" yaml:"username"`
	Password          string        `json:"password" yaml:"password"`
	Urls              []string      `json:"urls" yaml:"urls"`
	Index             string        `json:"index" yaml:"index"`
	MaxPendingBytes   int64         `json:"max_pending_bytes" yaml:"max_pending_bytes"`
	MaxPendingResults int64         `json:"max_pending_results" yaml:"max_pending_results"`
	MaxPendingTime    time.Duration `json:"max_pending_time" yaml:"max_pending_time"`
}

// Emitter for elasticsearch
type Emitter struct {
	client        *elastic.Client
	index         string
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

	req := elastic.NewBulkIndexRequest().
		OpType("create").
		Index(e.index).
		Id(result.UUID).
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
		return nil, fmt.Errorf("emitter/elasticsearch: error creating client: %w", err)
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
		return nil, fmt.Errorf("emitter/elasticsearch: error creating bulk processor: %w", err)
	}

	emitter.bulkProcessor.Start(context.Background())
	return emitter, nil
}
