package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/pachimsh/cli/internal/api"
	"github.com/pachimsh/cli/internal/config"
	"github.com/pachimsh/cli/internal/git"
)

type deployPlanDecision int

const (
	planDeploy deployPlanDecision = iota
	planWatch
	planCancelled
)

type deployPlan struct {
	Target           *pushTarget
	SiteInfo         *api.SiteInfo
	CWD              string
	Branch           string
	CommitShort      string
	DirtyCount       int
	BranchMismatch   bool
	HasActiveDeploy  bool
	GitMergeDisabled bool
	FirstDeploy      bool
	ProductionRisk   bool
	AutoResolved     bool
}

func buildDeployPlan(target *pushTarget, siteInfo *api.SiteInfo, cwd string) *deployPlan {
	plan := &deployPlan{
		Target:          target,
		SiteInfo:        siteInfo,
		CWD:             cwd,
		Branch:          resolveDeployBranch(target, siteInfo, cwd),
		AutoResolved:    target.AutoResolved,
		FirstDeploy:     siteInfo.SetupType == "",
		HasActiveDeploy: siteInfo.ActiveDeployment != nil,
		GitMergeDisabled: !siteInfo.GitMerge && siteInfo.SetupType != "",
	}

	if target.Head != nil && target.Head.Commit != "" {
		short := target.Head.Commit
		if len(short) > 7 {
			short = short[:7]
		}
		plan.CommitShort = short
	}

	if wt, ok := git.WorkingTreeStatusIn(cwd); ok {
		plan.DirtyCount = wt.ChangedFiles
	}

	mapped := strings.TrimSpace(target.Site.DeployBranch)
	if target.Head != nil && mapped != "" && target.Head.Branch != "" {
		plan.BranchMismatch = !strings.EqualFold(mapped, target.Head.Branch)
	}

	checkBranch := plan.Branch
	if checkBranch == "" && target.Head != nil {
		checkBranch = target.Head.Branch
	}
	plan.ProductionRisk = config.IsProductionBranch(checkBranch) && plan.DirtyCount > 0

	return plan
}

func (p *deployPlan) canSilentDeploy(yes, force bool) bool {
	if yes {
		return !p.ProductionRisk || force
	}

	if p.ProductionRisk && !force {
		return false
	}
	if p.DirtyCount > 0 {
		return false
	}
	if p.BranchMismatch {
		return false
	}
	if p.HasActiveDeploy {
		return false
	}
	if p.GitMergeDisabled {
		return false
	}

	return p.AutoResolved || pushSiteFlag != ""
}

func printDeployPlan(p *deployPlan) {
	name := p.Target.Site.DisplayName(p.Target.Alias)
	fmt.Println()
	fmt.Println("┌─ Deploy plan " + strings.Repeat("─", 46))
	fmt.Printf("│ Site:     %s\n", p.Target.Site.Domain)
	if name != p.Target.Alias {
		fmt.Printf("│ Alias:    %s (%s)\n", name, p.Target.Alias)
	} else {
		fmt.Printf("│ Alias:    %s\n", p.Target.Alias)
	}

	if p.AutoResolved {
		fmt.Println("│ Match:    git branch mapping")
	}

	if p.Branch != "" {
		line := fmt.Sprintf("│ Branch:   %s", p.Branch)
		if p.CommitShort != "" {
			line += fmt.Sprintf(" @ %s", p.CommitShort)
		}
		fmt.Println(line)
	}

	if p.GitMergeDisabled {
		fmt.Println("│ Mode:     full replace (git merge off)")
	} else if p.FirstDeploy {
		fmt.Println("│ Mode:     first deploy")
	} else {
		fmt.Println("│ Mode:     git merge")
	}

	if p.DirtyCount > 0 {
		fmt.Printf("│ Git:      %d uncommitted change(s) ⚠\n", p.DirtyCount)
	} else {
		fmt.Println("│ Git:      clean")
	}

	if p.BranchMismatch {
		fmt.Println("│ Warning:  checked-out branch differs from mapped branch ⚠")
	}

	if p.ProductionRisk {
		fmt.Println("│ Warning:  deploying production branch with local changes ⚠")
	}

	if p.HasActiveDeploy {
		active := p.SiteInfo.ActiveDeployment
		fmt.Printf("│ Server:   deployment in progress (%s) ⚠\n", active.Status)
	} else {
		fmt.Println("│ Server:   idle")
	}

	fmt.Println("└" + strings.Repeat("─", 59))
}

func resolveDeployPlan(client *api.Client, p *deployPlan, yes, force, dryRun bool) (deployPlanDecision, bool, error) {
	if dryRun {
		printDeployPlan(p)
		fmt.Println()
		color.Cyan("  Dry run — no deployment started.")
		return planCancelled, false, nil
	}

	if p.canSilentDeploy(yes, force) {
		if !quietFlag {
			branchPart := p.Branch
			if branchPart == "" {
				branchPart = "default"
			}
			commitPart := ""
			if p.CommitShort != "" {
				commitPart = fmt.Sprintf(" @ %s", p.CommitShort)
			}
			logNotice("→ Deploying %s (%s%s)...", p.Target.Site.Domain, branchPart, commitPart)
		}
		return planDeploy, false, nil
	}

	printDeployPlan(p)
	fmt.Println()

	enableGitMerge := false

	if p.ProductionRisk && !force {
		fmt.Println("  You are deploying a production branch with uncommitted changes.")
		if !readYesNo("  Continue anyway? [y/N]: ", false) {
			return planCancelled, false, nil
		}
	}

	if p.HasActiveDeploy {
		fmt.Println("  [D] Deploy now (queue new deployment)")
		fmt.Println("  [W] Watch current deployment")
		fmt.Println("  [C] Cancel")
		fmt.Println()
		fmt.Print("  Choose an option [D/w/c]: ")

		choice := readPlanChoice("")
		switch choice {
		case "w", "watch":
			return planWatch, false, nil
		case "c", "cancel":
			return planCancelled, false, nil
		default:
			// continue to git merge / final confirm
		}
	}

	if p.GitMergeDisabled {
		fmt.Println("  Git merge is disabled — upload replaces all code on the server.")
		fmt.Println("  [E] Enable git merge and deploy")
		fmt.Println("  [D] Deploy with full replace")
		fmt.Println("  [C] Cancel")
		fmt.Println()
		fmt.Print("  Choose an option [E/d/c]: ")

		choice := readPlanChoice("e")
		switch choice {
		case "e", "enable":
			enableGitMerge = true
		case "d", "deploy":
			// full replace
		default:
			return planCancelled, false, nil
		}
	} else if !yes {
		if !readYesNo("  Deploy now? [Y/n]: ", true) {
			return planCancelled, false, nil
		}
	}

	if enableGitMerge {
		if !enableGitMergeOnServer(client, p.Target.Site.ID) {
			return planCancelled, false, nil
		}
	}

	return planDeploy, enableGitMerge, nil
}

func readPlanChoice(defaultChoice string) string {
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "" {
		return defaultChoice
	}
	return answer
}

func enableGitMergeOnServer(client *api.Client, siteID string) bool {
	enabled, err := client.ToggleGitMerge(siteID)
	if err != nil {
		color.Red("Failed to enable git merge: %s", err)
		return false
	}
	if enabled && !quietFlag {
		logSuccess("✓ Git merge enabled.")
	}
	return true
}
