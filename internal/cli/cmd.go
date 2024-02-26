// Package cli contains the agent command line interface.
package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/telekom-mms/fw-id-agent/internal/agent"
	"github.com/telekom-mms/fw-id-agent/pkg/client"
	"github.com/telekom-mms/fw-id-agent/pkg/status"
)

var (
	// command is the command specified on the command line.
	command = ""

	// verbose specifies verbose output.
	verbose = false

	// json specifies whether output should be formatted as json.
	json = false
)

// parseCommandLine parses the command line arguments.
func parseCommandLine(args []string) error {
	// status subcommand
	statusCmd := flag.NewFlagSet("status", flag.ContinueOnError)
	statusCmd.BoolVar(&verbose, "verbose", verbose, "set verbose output")
	statusCmd.BoolVar(&json, "json", json, "set json output")

	// command line arguments
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	ver := flags.Bool("version", false, "print version")

	// usage output
	flags.Usage = func() {
		cmd := flags.Name()
		w := flags.Output()
		usage := func(f string, args ...any) {
			_, _ = fmt.Fprintf(w, f, args...)
		}
		usage("Usage:\n")
		usage("  %s [options] [command]\n", cmd)
		usage("\nOptions:\n")
		flags.PrintDefaults()
		usage("\nCommands:\n")
		usage("  status\n")
		usage("        show agent status\n")
		usage("  monitor\n")
		usage("        monitor agent status updates\n")
		usage("  relogin\n")
		usage("        relogin agent\n")
	}

	// parse command line arguments
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}

	// print version?
	if *ver {
		fmt.Println(agent.Version)
		// treat -version like -help
		return flag.ErrHelp
	}

	// parse subcommands
	command = flags.Arg(0)
	switch command {
	case "status":
		if err := statusCmd.Parse(args[2:]); err != nil {
			return err
		}
	case "monitor":
	case "relogin":
	default:
		flags.Usage()
		return fmt.Errorf("unknown command")
	}

	return nil
}

// printStatus prints status.
func printStatus(out io.Writer, s *status.Status, verbose bool) error {
	printf := func(format string, a ...any) {
		_, _ = fmt.Fprintf(out, format, a...)
	}
	printf("Trusted Network:    %s\n", s.TrustedNetwork)
	printf("Login State:        %s\n", s.LoginState)
	if verbose {
		// last keep-alive info
		if s.LastKeepAlive <= 0 {
			printf("Last Keep-Alive:\n")
		} else {
			lastKeepAlive := time.Unix(s.LastKeepAlive, 0)
			printf("Last Keep-Alive:    %s\n", lastKeepAlive)
		}

		// kerberos info
		printf("Kerberos TGT:\n")

		// kerberos tgt start time
		if s.KerberosTGT.StartTime <= 0 {
			printf("- Start Time:\n")
		} else {
			tgtStartTime := time.Unix(s.KerberosTGT.StartTime, 0)
			printf("- Start Time:       %s\n", tgtStartTime)
		}

		// kerberos tgt end time
		if s.KerberosTGT.EndTime <= 0 {
			printf("- End Time:\n")
		} else {
			tgtEndTime := time.Unix(s.KerberosTGT.EndTime, 0)
			printf("- End Time:         %s\n", tgtEndTime)
		}

		// agent config
		config, err := s.Config.JSON()
		if err != nil {
			return err
		}
		printf("Config:             %s\n", config)
	}

	return nil
}

// getStatus retrieves the agent status and prints it.
func getStatus(c client.Client) error {
	// query status
	s, err := c.Query()
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}

	if json {
		// print status as json
		j, err := s.JSONIndent()
		if err != nil {
			return fmt.Errorf("error converting status to json: %w", err)
		}
		fmt.Println(string(j))
		return nil
	}

	// print status
	return printStatus(os.Stdout, s, verbose)
}

// relogin sends a relogin request to the agent.
func relogin(c client.Client) error {
	// send request to agent
	if err := c.ReLogin(); err != nil {
		return fmt.Errorf("re-login request failed: %w", err)
	}
	return nil
}

// monitor subscribes to status updates from the agent and displays them.
func monitor(c client.Client) error {
	// get status updates
	updates, err := c.Subscribe()
	if err != nil {
		return fmt.Errorf("error subscribing to status updates: %w", err)
	}
	for u := range updates {
		log.Println("Got status update:")
		if err := printStatus(os.Stdout, u, true); err != nil {
			return err
		}
	}
	return nil
}

func runCommand(c client.Client, command string) error {
	switch command {
	case "status":
		return getStatus(c)
	case "monitor":
		return monitor(c)
	case "relogin":
		return relogin(c)
	}
	return nil
}

func run(args []string) error {
	// parse command line
	if err := parseCommandLine(args); err != nil {
		return err
	}

	// create client
	c, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("could not create client: %w", err)
	}
	defer func() { _ = c.Close() }()

	// run commands
	return runCommand(c, command)
}

// Run is the main entry point.
func Run() {
	if err := run(os.Args); err != nil {
		if err != flag.ErrHelp {
			log.Fatal(err)
		}
		return
	}
}
