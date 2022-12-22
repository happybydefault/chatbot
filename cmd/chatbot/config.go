package main

import (
	"github.com/spf13/pflag"
)

type config struct {
	development        bool
	postgresConnString string
}

func newConfig(args []string) (config, error) {
	var cfg config

	flagSet := pflag.NewFlagSet(programName, pflag.ContinueOnError)

	flagSet.BoolVarP(
		&cfg.development,
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
