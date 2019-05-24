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

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/getamis/sirius/log"
	"github.com/getamis/sirius/metrics"
	"github.com/spf13/cobra"

	"github.com/getamis/hyperbtc/metricsexporter"
)

const (
	metricsPath = "/metrics"
)

var (
	host            string
	port            int
	rpcUser         string
	rpcPassword     string
	rpcHost         string
	refreshInterval time.Duration
	namespace       string
	labels          map[string]string
)

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

// RootCmd represents the Prometheus metrics exporter
var RootCmd = &cobra.Command{
	Use:   "metrics-exporter",
	Short: "The Bitcoin metrics exporter for Prometheus",
	Long:  `The Bitcoin metrics exporter for Prometheus.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config := &rpcclient.ConnConfig{
			Host:         rpcHost,
			User:         rpcUser,
			Pass:         rpcPassword,
			DisableTLS:   true,
			HTTPPostMode: true,
		}
		client, err := rpcclient.New(config, nil)
		if err != nil {
			panic(err)
		}
		defer client.Shutdown()

		metricsRegistry := metrics.NewPrometheusRegistry()
		metricsRegistry.SetNamespace(namespace)
		metricsRegistry.AppendLabels(labels)

		exporter := metricsexporter.New(metricsRegistry, client, refreshInterval)

		// Start to collect metrics
		runCtx, runCancel := context.WithCancel(context.Background())
		defer runCancel()
		go exporter.Run(runCtx)

		mux := http.NewServeMux()
		mux.Handle(metricsPath, metricsRegistry)
		httpSrv := &http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: mux,
		}

		// Start http server
		go func() {
			log.Info("Starting metrics exporter", "endpoint", fmt.Sprintf("http://%s:%d%s", host, port, metricsPath))
			err := httpSrv.ListenAndServe()
			if err != nil {
				log.Debug("Http server is stopping with error", "err", err)
			}
		}()

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
		defer signal.Stop(sigs)
		log.Debug("Shutting down", "signal", <-sigs)

		// Shutdown http server and the graceful period is 30 seconds
		downCtx, downCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer downCancel()
		httpSrv.Shutdown(downCtx)

		return nil
	},
}

func init() {
	RootCmd.Flags().StringVar(&host, "host", "0.0.0.0", "The HTTP server listening address")
	RootCmd.Flags().IntVar(&port, "port", 9092, "The HTTP server listening port")
	RootCmd.Flags().StringVar(&rpcUser, "rpcuser", "", "Username for JSON-RPC connections")
	RootCmd.Flags().StringVar(&rpcPassword, "rpcpassword", "", "Password for JSON-RPC connections")
	RootCmd.Flags().StringVar(&rpcHost, "rpchost", "127.0.0.1:8332", "Host for JSON-RPC connections")
	RootCmd.Flags().DurationVar(&refreshInterval, "period", 15*time.Second, "The metrics refresh interval")
	RootCmd.Flags().StringVar(&namespace, "namespace", "btc", "The namespace of metrics")
	RootCmd.Flags().StringToStringVar(&labels, "labels", map[string]string{}, "The labels of metrics. For example: k1=v1,k2=v2")
}
