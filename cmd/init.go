// Copyright © 2019 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"context"
	"github.com/gofunct/cloud/inject"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"net/http/pprof"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "A brief description of your command",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		var app *inject.Application
		var cleanup func()
		var err error
		app, cleanup, err = inject.SetupGCP(ctx, config)
		if err != nil {
			log.Fatal(err)
		}
		defer cleanup()

		// Set up URL routes.
		r := mux.NewRouter()
		r.HandleFunc("/", inject.Index(app, config))
		r.HandleFunc("/sign", inject.Sign(app, config))
		r.HandleFunc("/blob/{key:.+}", inject.ServeBlob(app, config))
		r.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
		r.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
		r.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
		r.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
		r.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
		r.Handle("/metrics", promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))
		// Listen and serve HTTP.
		log.Println("Running, connected to google cloud platform")
		log.Fatal(app.Server.ListenAndServe(port, r))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
