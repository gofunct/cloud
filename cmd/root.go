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
	"github.com/fatih/color"
	"github.com/gofunct/cloud/inject"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"log"
	"os"
	"time"
)

var (
	cfgFile string
	config  = &inject.Config{}
	home = os.Getenv("HOME")
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use: "cloudctl",
	Long: color.BlueString(`Cloudctl is a fast and flexible cloud development utility for multiple platforms

Author: Coleman Word
Download: gp get github.com/gofunct/cloud/...

 oooooooo8 o888                              oooo             o8   o888 
o888     88  888   ooooooo  oooo  oooo   ooooo888   ooooooo  o888oo  888 
888          888 888     888 888   888 888    888 888     888 888    888 
888o     oo  888 888     888 888   888 888    888 888         888    888 
 888oooo88  o888o  88ooo88    888o88 8o  88ooo888o  88ooo888   888o o888o
`),
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
		rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $PWD/cloudctl.yaml)")
		rootCmd.PersistentFlags().StringVar(&config.SQLRegion, "sqlregion", "", "cloud sql region")
		rootCmd.PersistentFlags().StringVar(&config.DbName, "dbname", "defdb", "database name")
		rootCmd.PersistentFlags().StringVar(&config.DbHost, "dbhost", "", "database host")
		rootCmd.PersistentFlags().StringVar(&config.DbUser, "dbuser", "defuser", "database user name")
		rootCmd.PersistentFlags().StringVar(&config.DbPass, "dbpass", "defpass", "database password")
		rootCmd.PersistentFlags().StringVar(&config.Bucket, "bucket", "defbucket", "blob storage bucket")
		rootCmd.PersistentFlags().StringVar(&config.Env, "env", "gcp", "target deployment environment-> |local|gcp|aws|")
		rootCmd.PersistentFlags().DurationVar(&config.RunVarWait, "runvarwait", 30*time.Second, "timeout for runtime config watcher")
		rootCmd.PersistentFlags().StringVar(&config.RunVar, "runvar", "hello world", "runtime variable value")
		rootCmd.PersistentFlags().StringVar(&config.RunVarName, "runvarname", "motd", "runtime variable name")
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
	if err := viper.Unmarshal(config); err != nil {
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
