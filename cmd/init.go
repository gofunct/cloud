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
		switch config.Env {
		case "gcp":
			if config.DbPass == "" {
				config.DbPass = "gcpadmin"
			}
			app, cleanup, err = inject.SetupGCP(ctx, config)
		case "aws":
			if config.DbPass == "" {
				config.DbPass = "awsadmin"
			}
			app, cleanup, err = inject.SetupAWS(ctx, config)
		case "local":
			if config.DbHost == "" {
				config.DbHost = "localhost"
			}
			if config.DbPass == "" {
				config.DbPass = "localadmin"
			}
			app, cleanup, err = inject.SetupLocal(ctx, config)
		default:
			log.Fatalf("unknown env=%s\n valid: |local|gcp|aws|", config.Env)
		}
		if err != nil {
			log.Fatal(err)
		}
		defer cleanup()

		// Set up URL routes.
		r := mux.NewRouter()
		r.HandleFunc("/", inject.Index(app, config))
		r.HandleFunc("/sign", inject.Sign(app, config))
		r.HandleFunc("/blob/{key:.+}", inject.ServeBlob(app, config))

		// Listen and serve HTTP.
		log.Printf("Running, connected to %q cloud", config.Env)
		log.Fatal(app.Server.ListenAndServe(":8080", r))
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
