// Copyright 2019 AMIS Technologies

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

// 	http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metricsexporter

import (
	"context"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/getamis/sirius/log"
	"github.com/getamis/sirius/metrics"
)

const (
	blockNumberMetricName = "block_number"
	peerCountMetricName   = "peer_count"
	memPoolSizeMetricName = "raw_mempool_size"
)

type MetricsExporter struct {
	metricsRegistry *metrics.PrometheusRegistry
	gauges          map[string]metrics.Gauge
	client          *rpcclient.Client
	refreshInterval time.Duration
}

func New(metricsRegistry *metrics.PrometheusRegistry, client *rpcclient.Client, refreshInterval time.Duration) *MetricsExporter {
	return &MetricsExporter{
		metricsRegistry: metricsRegistry,
		gauges:          make(map[string]metrics.Gauge),
		client:          client,
		refreshInterval: refreshInterval,
	}
}

func (e *MetricsExporter) Run(ctx context.Context) {
	ticker := time.NewTicker(e.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Debug("Stop collect metrics", "err", ctx.Err())
			return
		case <-ticker.C:
			go func() {
				start := time.Now()
				e.getBlockNumber()
				e.getPeerInfo()
				e.getRawMempool()
				log.Trace("Scrap metrics", "elapsed", time.Since(start))
			}()
		}
	}
}

func (e *MetricsExporter) getBlockNumber() {
	gauge := e.getGauge(blockNumberMetricName)

	blockCount, err := e.client.GetBlockCount()
	if err != nil {
		log.Warn("Failed to get block number", "err", err)
	}
	gauge.Set(float64(blockCount))
}

func (e *MetricsExporter) getPeerInfo() {
	gauge := e.getGauge(peerCountMetricName)

	peerInfo, err := e.client.GetPeerInfo()
	if err != nil {
		log.Warn("Failed to get peer info", "err", err)
	}
	gauge.Set(float64(len(peerInfo)))
}

func (e *MetricsExporter) getRawMempool() {
	gauge := e.getGauge(memPoolSizeMetricName)

	hashes, err := e.client.GetRawMempool()
	if err != nil {
		log.Warn("Failed to get raw mempool", "err", err)
	}
	gauge.Set(float64(len(hashes)))
}

func (e *MetricsExporter) getGauge(name string) metrics.Gauge {
	var gauge metrics.Gauge
	gauge, ok := e.gauges[name]
	if !ok {
		gauge = e.metricsRegistry.NewGauge(name)
		e.gauges[name] = gauge
	}
	return gauge
}
