package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
)

type pushTarget struct {
	Alias        string
	Site         config.SiteConfig
	Head         *git.Head
	AutoResolved bool
}

func resolvePushTarget(projCfg *config.ProjectConfig, creds *config.Credentials, cwd string) (*pushTarget, error) {
	if pushSiteFlag != "" {
		site, ok := projCfg.Sites[pushSiteFlag]
		if !ok {
			color.Red("Site '%s' not found in .pachim.json", pushSiteFlag)
			fmt.Println("Available sites:")
			for alias := range projCfg.Sites {
				fmt.Printf("  • %s\n", alias)
			}
			return nil, errPushAborted
		}

		return &pushTarget{
			Alias: pushSiteFlag,
			Site:  site,
			Head:  currentHeadOrNil(cwd),
		}, nil
	}

	head := currentHeadOrNil(cwd)
	if head != nil && head.Branch != "" {
		matches := config.SitesForBranch(projCfg, head.Branch)
		switch len(matches) {
		case 1:
			alias := matches[0]
			logVerbose("Matched git branch '%s' to site '%s'", head.Branch, alias)
			if verboseFlag {
				color.Cyan("Matched git branch '%s' to site '%s'", head.Branch, alias)
			}
			return &pushTarget{
				Alias:        alias,
				Site:         projCfg.Sites[alias],
				Head:         head,
				AutoResolved: true,
			}, nil
		case 0:
			if len(projCfg.Sites) > 1 && !quietFlag {
				color.Yellow("No site is mapped to git branch '%s'.", head.Branch)
			}
		default:
			if pushYesFlag {
				return nil, fmt.Errorf("branch '%s' maps to multiple sites (%s); use --site", head.Branch, strings.Join(matches, ", "))
			}
			alias, ok := promptSelectSite(projCfg, matches, fmt.Sprintf("Branch '%s' matches multiple sites", head.Branch))
			if !ok {
				return nil, errPushAborted
			}
			return &pushTarget{
				Alias: alias,
				Site:  projCfg.Sites[alias],
				Head:  head,
			}, nil
		}
	}

	if alias, ok := config.OnlySiteAlias(projCfg); ok {
		return &pushTarget{
			Alias: alias,
			Site:  projCfg.Sites[alias],
			Head:  head,
		}, nil
	}

	if pushYesFlag {
		return nil, fmt.Errorf("multiple sites configured; use --site or map deploy_branch in .pachim.json")
	}

	alias, ok := promptSelectSite(projCfg, nil, "Select site to deploy")
	if !ok {
		return nil, errPushAborted
	}

	target := &pushTarget{
		Alias: alias,
		Site:  projCfg.Sites[alias],
		Head:  head,
	}

	if head != nil && head.Branch != "" {
		offerSaveDeployBranchMapping(projCfg, creds, alias, head.Branch)
		target.Site = projCfg.Sites[alias]
	}

	return target, nil
}

func currentHeadOrNil(cwd string) *git.Head {
	head, ok := git.CurrentHead(cwd)
	if !ok {
		return nil
	}
	return head
}

func promptSelectSite(projCfg *config.ProjectConfig, limitTo []string, title string) (string, bool) {
	fmt.Println()
	fmt.Println(title + ":")
	fmt.Println(strings.Repeat("-", 60))

	allowed := make(map[string]struct{}, len(limitTo))
	for _, alias := range limitTo {
		allowed[alias] = struct{}{}
	}

	var choices []string
	for _, alias := range config.SortedSiteAliases(projCfg) {
		if len(limitTo) > 0 {
			if _, ok := allowed[alias]; !ok {
				continue
			}
		}
		choices = append(choices, alias)
	}

	if len(choices) == 0 {
		color.Red("No sites available to select.")
		return "", false
	}

	for i, alias := range choices {
		site := projCfg.Sites[alias]
		name := site.DisplayName(alias)
		marker := "  "
		if alias == projCfg.Default {
			marker = "* "
		}

		branchLabel := site.DeployBranch
		if branchLabel == "" {
			branchLabel = "(any branch)"
		}

		fmt.Printf("%s%d) %s — %s [branch: %s]\n", marker, i+1, name, site.Domain, branchLabel)
	}

	fmt.Println(strings.Repeat("-", 60))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Select site number: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(choices) {
		color.Red("Invalid selection.")
		return "", false
	}

	return choices[idx-1], true
}

func resolveDeployBranch(target *pushTarget, siteInfo *api.SiteInfo, cwd string) string {
	if pushBranchFlag != "" {
		return strings.TrimSpace(pushBranchFlag)
	}

	if branch := strings.TrimSpace(target.Site.DeployBranch); branch != "" {
		return branch
	}

	if target.Head != nil && target.Head.Branch != "" {
		return target.Head.Branch
	}

	if siteInfo != nil {
		return strings.TrimSpace(siteInfo.DeployBranch)
	}

	return ""
}

func readYesNo(prompt string, defaultYes bool) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer == "" {
		return defaultYes
	}

	return answer == "y" || answer == "yes"
}

var errPushAborted = fmt.Errorf("push cancelled")
