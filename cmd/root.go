// Copyright © 2016 Tom Maiaroto <tom@shift8creative.com>
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
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/apigateway"
	"github.com/aws/aws-sdk-go/service/lambda"
	// "github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
)

var cfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "aegis",
	Short: "Deploy RESTful serverless APIs ",
	Long:  `A tool to deploy a RESTful serverless API using AWS Lambda and API Gateway with a Lambda Proxy and an API Gateway ANY request.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// deploymentConfig holds the AWS Lambda configuration
type deploymentConfig struct {
	App struct {
		Name           string
		KeepBuildFiles bool
		BuildFileName  string
	}
	AWS struct {
		Region string
	}
	Lambda struct {
		Wrapper              string
		Runtime              string
		Handler              string
		FunctionName         string
		Alias                string
		Description          string
		MemorySize           int64
		Role                 string
		Timeout              int64
		SourceZip            string
		EnvironmentVariables map[string]*string
		KMSKeyArn            string
		VPC                  struct {
			SecurityGroups []string
			Subnets        []string
		}
	}
	API struct {
		Name             string
		Description      string
		Cache            bool
		CacheSize        string
		Stages           map[string]deploymentStage
		BinaryMediaTypes []*string
	}
}

// deploymentStage defines an API Gateway stage and holds configuration options for it
type deploymentStage struct {
	Name        string
	Description string
	Variables   map[string]*string
	Cache       bool
	CacheSize   string
}

// cfg holds the Aegis configuration for the Lambda function, API Gateway settings, etc.
var cfg deploymentConfig

// awsCfg holds the AWS configuration and credentials
var awsCfg aws.Config

// Execute adds all child commands to the root command sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "aegis", "config file (default is aegis.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	viper.SetConfigName("aegis") // name of config file (without extension)
	viper.AddConfigPath(".")
	// viper.AddConfigPath("$HOME")  // adding home directory as first search path
	viper.AutomaticEnv() // read in environment variables that match

	// Default config values
	viper.SetDefault("aws.region", "us-east-1")
	// Default Lambda config values
	viper.SetDefault("lambda.functionName", "aegis_example")
	// Valid runtimes:
	// nodejs
	// nodejs4.3
	// java8
	// python2.7
	viper.SetDefault("lambda.runtime", lambda.RuntimeNodejs43)
	viper.SetDefault("lambda.wrapper", "index_stdio.js") // TODO: allow multiple wrappers
	viper.SetDefault("lambda.handler", "index.handler")
	viper.SetDefault("lambda.alias", "current")
	// In megabytes
	viper.SetDefault("lambda.memorySize", int64(128))
	// In seconds
	viper.SetDefault("lambda.timeout", int64(3))
	// No suitable default for this
	// viper.SetDefault("lambda.role", "arn:aws:iam::account-id:role/lambda_basic_execution")
	// Set a default function name
	fName := "aegis_function"
	// Use the current directory name by default
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err == nil {
		// Prepend aegis_
		if dir != "/" && dir != "" {
			var buffer bytes.Buffer
			buffer.WriteString("aegis_")
			buffer.WriteString(dir)
			fName = buffer.String()
			buffer.Reset()
		}
	}
	viper.SetDefault("lambda.functionName", fName)
	// Default API Gateway config values
	viper.SetDefault("api.name", "Aegis API")
	viper.SetDefault("api.description", "")
	viper.SetDefault("api.cache", false)
	// For valid values, see: https://godoc.org/github.com/aws/aws-sdk-go/service/apigateway#pkg-constants
	viper.SetDefault("api.cacheSize", apigateway.CacheClusterSize05)

	// Default API stage (does not use caching, that comes with an additional cost)
	viper.SetDefault("api.stages", map[string]deploymentStage{
		"prod": deploymentStage{
			Name:        "prod",
			Description: "production stage",
			Cache:       false, // no cache by default
			// CacheSize:   apigateway.CacheClusterSize05, // never needed because above is false and empty value caught in deployment
		},
	})

	// By default do not keep the build files (clean up)
	viper.SetDefault("app.keepBuildFiles", false)
	// Just in case the temporary zip file that gets built creates a conflict, it can be adjusted. However, this is the default.
	viper.SetDefault("app.buildFileName", "aegis_function.zip")

	// If a config file is found, read it in.
	err = viper.ReadInConfig()
	if err == nil {
		// TODO verbose?
		// fmt.Println("Using config file:", viper.ConfigFileUsed())
	} else {
		// Not strictly required given defaults? This prevents other commands such as `init` from working.
		// fmt.Printf("%v %v\n", color.YellowString("Warning:"), "Could not find aegis config file.")
		// os.Exit(-1)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		fmt.Println("Could parse aegis config file.")
		os.Exit(-1)
	}

	// Initialize AWS config
	awsCfg = aws.Config{
		Region: aws.String(cfg.AWS.Region),
	}

}
