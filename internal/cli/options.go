package cli

import (
	"flag"
	"fmt"
	"io"
)

type Options struct {
	Run        bool
	OpenReport bool
}

func ParseArgs(args []string) (Options, error) {
	normalized := normalizeArgs(args)
	flags := flag.NewFlagSet("splitter", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var opts Options
	flags.BoolVar(&opts.Run, "run", false, "Run in headless mode")
	flags.BoolVar(&opts.OpenReport, "open-report", false, "Open report after run")

	if err := flags.Parse(normalized); err != nil {
		return Options{}, err
	}
	if opts.OpenReport && !opts.Run {
		return Options{}, fmt.Errorf("--open-report can only be used with --run")
	}
	return opts, nil
}

func normalizeArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		switch arg {
		case "/run":
			out = append(out, "--run")
		case "/open-report":
			out = append(out, "--open-report")
		default:
			out = append(out, arg)
		}
	}
	return out
}
