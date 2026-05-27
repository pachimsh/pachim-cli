package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pachim/cli/internal/config"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:   "use [alias]",
	Short: "Set the default site for deployment",
	Long: `Change which site 'pachim push' deploys to by default.
If no alias is provided, an interactive selection is shown.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runUse,
}

func init() {
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	projCfg, err := config.LoadProjectConfig()
	if err != nil {
		color.Red("Project not initialized. Run: pachim init")
		return nil
	}

	if len(projCfg.Sites) == 0 {
		color.Red("No sites configured. Run: pachim init")
		return nil
	}

	var targetAlias string

	if len(args) == 1 {
		targetAlias = args[0]
	} else {
		fmt.Println()
		fmt.Println("Configured sites:")
		fmt.Println(strings.Repeat("-", 50))

		aliases := make([]string, 0, len(projCfg.Sites))
		i := 0
		for alias, site := range projCfg.Sites {
			i++
			marker := "  "
			if alias == projCfg.Default {
				marker = "▸ "
			}
			fmt.Printf("%s%d) %s (%s)\n", marker, i, alias, site.Domain)
			aliases = append(aliases, alias)
		}
		fmt.Println(strings.Repeat("-", 50))
		fmt.Println()

		reader := bufio.NewReader(os.Stdin)
		fmt.Print("Select site number: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		idx, err := strconv.Atoi(input)
		if err != nil || idx < 1 || idx > len(aliases) {
			color.Red("Invalid selection.")
			return nil
		}
		targetAlias = aliases[idx-1]
	}

	if _, ok := projCfg.Sites[targetAlias]; !ok {
		color.Red("Site '%s' not found in .pachim.json", targetAlias)
		fmt.Println("Available sites:")
		for alias := range projCfg.Sites {
			fmt.Printf("  • %s\n", alias)
		}
		return nil
	}

	projCfg.Default = targetAlias
	if err := config.SaveProjectConfig(projCfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	color.Green("✓ Default site set to: %s", targetAlias)
	return nil
}
