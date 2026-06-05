package cmd

import (
	"fmt"

	"github.com/fatih/color"
)

var verboseFlag bool
var quietFlag bool

func initOutputFlags() {
	rootCmd.PersistentFlags().BoolVar(&verboseFlag, "verbose", false, "Show detailed sync and resolution output")
	rootCmd.PersistentFlags().BoolVar(&quietFlag, "quiet", false, "Minimal output (errors and results only)")
}

func logVerbose(format string, args ...interface{}) {
	if verboseFlag {
		fmt.Printf(format+"\n", args...)
	}
}

func logInfo(format string, args ...interface{}) {
	if quietFlag {
		return
	}
	fmt.Printf(format+"\n", args...)
}

func logNotice(format string, args ...interface{}) {
	if quietFlag {
		return
	}
	color.Cyan(format, args...)
}

func logWarn(format string, args ...interface{}) {
	if quietFlag {
		return
	}
	color.Yellow(format, args...)
}

func logSuccess(format string, args ...interface{}) {
	color.Green(format, args...)
}
