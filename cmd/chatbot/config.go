package main

import (
	"os"

	"github.com/spf13/pflag"
)

type config struct {
	development        bool
	chatIDs            []string
	postgresConnString string
	openAIAPIKey       string
}

func newConfig(args []string) (config, error) {
	cfg := config{
		openAIAPIKey: os.Getenv("OPENAI_API_KEY"),
	}

	flagSet := pflag.NewFlagSet(programName, pflag.ContinueOnError)

	flagSet.BoolVarP(
		&cfg.development,
		"development",
		"d",
		false,
		"Enable development mode",
	)
	flagSet.StringSliceVarP(
		&cfg.chatIDs,
		"chats",
		"c",
		nil,
		"Chat IDs to add to the store",
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
