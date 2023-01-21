package main

import (
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type config struct {
	isDevelopment      bool
	postgresConnString string
	openAIAPIKey       string
}

func newConfig(args []string) (config, error) {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	if openAIAPIKey == "" {
		return config{}, fmt.Errorf("environment variable OPENAI_API_KEY must be set")
	}

	cfg := config{
		openAIAPIKey: openAIAPIKey,
	}

	flagSet := pflag.NewFlagSet(programName, pflag.ContinueOnError)

	flagSet.BoolVarP(
		&cfg.isDevelopment,
		"development",
		"d",
		false,
		"Enable development mode",
	)
	flagSet.StringVarP(
		&cfg.postgresConnString,
		"postgres",
		"p",
		"postgres://user:password@localhost:5432/database",
		"Postgres connection string",
	)

	err := flagSet.Parse(args)
	if err != nil {
		return config{}, err
	}

	return cfg, err
}
