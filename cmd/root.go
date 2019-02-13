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
	"fmt"
	"github.com/gofunct/cloud/inject"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
)

var (
	cfgFile string
	port    string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "cloudctl",
	Long: `Cloudctl is a fast and flexible cloud development utility for multiple platforms

Author: Coleman Word
Download: gp get github.com/gofunct/cloud/...

 oooooooo8 o888                              oooo             o8   o888 
o888     88  888   ooooooo  oooo  oooo   ooooo888   ooooooo  o888oo  888 
888          888 888     888 888   888 888    888 888     888 888    888 
888o     oo  888 888     888 888   888 888    888 888         888    888 
 888oooo88  o888o  88ooo88    888o88 8o  88ooo888o  88ooo888   888o o888o
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	{
		rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "goexec.yaml", "config file (default is $PWD/cloudctl.yaml)")
		rootCmd.PersistentFlags().StringVar(&port, "port", ":8080", "port to run app on")
		rootCmd.PersistentFlags().StringVar(&inject.Configuration.ClientSecret, "clientsecret", "", "Oauth client secret")
		rootCmd.PersistentFlags().StringVar(&inject.Configuration.ClientId, "dbname", "clientid", "Oauth client id")
		rootCmd.PersistentFlags().StringVar(&inject.Configuration.Redirect, "redirect", "", "Oauth redirect url")
		rootCmd.PersistentFlags().StringVar(&inject.Configuration.Project, "project", "", "gcloud project id")
		rootCmd.PersistentFlags().StringVar(&inject.Configuration.Bucket, "bucket", "", "gcloud bucket to use for storage")
	}
	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		log.Println(err.Error())
	}
	if err := viper.BindPFlags(rootCmd.PersistentFlags()); err != nil {
		log.Println(err.Error())
	}

	for _, cmd := range rootCmd.Commands() {
		if err := viper.BindPFlags(cmd.Flags()); err != nil {
			log.Println(err.Error())
		}
		if err := viper.BindPFlags(cmd.PersistentFlags()); err != nil {
			log.Println(err.Error())
		}
	}
	if err := viper.Unmarshal(inject.Configuration); err != nil {
		log.Println("Failed to unmarshal config")
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.

		// Search config in home directory with name ".temp" (without extension).
		viper.AddConfigPath(os.Getenv("PWD"))
		viper.SetConfigName("cloudctl")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
