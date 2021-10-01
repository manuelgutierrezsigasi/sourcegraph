package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"

	"github.com/sourcegraph/sourcegraph/dev/sg/internal/run"
	"github.com/sourcegraph/sourcegraph/lib/output"
)

var (
	runFlagSet = flag.NewFlagSet("sg run", flag.ExitOnError)
	runCommand = &ffcli.Command{
		Name:       "run",
		ShortUsage: "sg run <command>...",
		ShortHelp:  "Run the given commands.",
		LongHelp:   constructRunCmdLongHelp(),
		FlagSet:    runFlagSet,
		Exec:       runExec,
	}
)

func runExec(ctx context.Context, args []string) error {
	ok, errLine := parseConf(*configFlag, *overwriteConfigFlag)
	if !ok {
		out.WriteLine(errLine)
		os.Exit(1)
	}

	if len(args) == 0 {
		out.WriteLine(output.Linef("", output.StyleWarning, "No command specified"))
		return flag.ErrHelp
	}

	var cmds []run.Command
	for _, arg := range args {
		cmd, ok := globalConf.Commands[arg]
		if !ok {
			out.WriteLine(output.Linef("", output.StyleWarning, "ERROR: command %q not found :(", arg))
			return flag.ErrHelp
		}
		cmds = append(cmds, cmd)
	}

	return run.Commands(ctx, globalConf.Env, cmds...)
}
func constructRunCmdLongHelp() string {
	var out strings.Builder

	fmt.Fprintf(&out, "  Runs the given command. If given a whitespace-separated list of commands it runs the set of commands.\n")

	// Attempt to parse config to list available commands, but don't fail on
	// error, because we should never error when the user wants --help output.
	_, _ = parseConf(*configFlag, *overwriteConfigFlag)

	if globalConf != nil {
		fmt.Fprintf(&out, "\n")
		fmt.Fprintf(&out, "AVAILABLE COMMANDS IN %s%s%s\n", output.StyleBold, *configFlag, output.StyleReset)

		for name := range globalConf.Commands {
			fmt.Fprintf(&out, "  %s\n", name)
		}
	}

	return out.String()
}