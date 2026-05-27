package main

import (
	"github.com/pachimsh/cli/cmd"
	"github.com/pachimsh/cli/internal/api"
)

var apiBaseURL string

func main() {
	if apiBaseURL != "" {
		api.DefaultBaseURL = apiBaseURL
	}

	cmd.Execute()
}
