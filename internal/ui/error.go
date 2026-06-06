package ui

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
)

func PrintAPIError(title string, err error) {
	if apiErr, ok := api.AsAPIError(err); ok {
		if title != "" {
			color.Red("%s", title)
		}

		if apiErr.Message != "" {
			fmt.Printf("  %s\n", apiErr.Message)
		}

		for _, detail := range apiErr.Details {
			fmt.Printf("  • %s\n", detail)
		}

		if len(apiErr.Hints) > 0 {
			fmt.Println()
			color.Yellow("What you can do:")
			for _, hint := range apiErr.Hints {
				fmt.Printf("  → %s\n", hint)
			}
		}

		return
	}

	if title != "" {
		color.Red("%s: %s", title, err)
		return
	}

	color.Red("%s", err)
}
