package main

import (
	"github.com/pachim/cli/cmd"
	"github.com/pachim/cli/internal/api"
)

var apiBaseURL string

func main() {
	if apiBaseURL != "" {
		api.DefaultBaseURL = apiBaseURL
	}

	cmd.Execute()
}
