package repl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tomatopunk/phantom/pkg/api/proto"
	"github.com/tomatopunk/phantom/pkg/cli/client"
)

const prompt = "phantom> "

// Run starts the REPL with the given args (e.g. --agent, --token). Blocks until exit.
func Run(args []string) error {
	flags, _, err := ParseFlags(args)
	if err != nil {
		return err
	}
	if flags.AgentAddr == "" {
		return fmt.Errorf("missing -agent; usage: phantom -agent <host:port> [-token <token>]")
	}

	ctx := context.Background()
	cli, err := client.New(ctx, flags.AgentAddr, flags.Token)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer cli.Close()

	if _, err := cli.Connect(ctx, ""); err != nil {
		return fmt.Errorf("session: %w", err)
	}

	if flags.ScriptPath != "" {
		f, err := os.Open(flags.ScriptPath)
		if err != nil {
			return fmt.Errorf("script %s: %w", flags.ScriptPath, err)
		}
		defer f.Close()
		return runInteractive(ctx, cli, f, os.Stdout, true)
	}
	return runInteractive(ctx, cli, os.Stdin, os.Stdout, false)
}

// runInteractive reads lines from in, sends to agent, writes responses to out.
// When scriptMode is true, first command failure returns an error (for non-zero exit).
func runInteractive(ctx context.Context, c *client.Client, in io.Reader, out io.Writer, scriptMode bool) error {
	if !scriptMode {
		go streamEventsToWriter(ctx, c, out)
	}
	scanner := bufio.NewScanner(in)
	for {
		if !scriptMode {
			fmt.Fprint(out, prompt)
		}
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if isExitCommand(line) {
			break
		}
		resp, err := c.Execute(ctx, line)
		if err != nil {
			fmt.Fprintf(out, "error: %v\n", err)
			if scriptMode {
				return err
			}
			continue
		}
		writeResponse(out, resp)
		if scriptMode && !resp.GetOk() {
			return fmt.Errorf("script failed: %s", resp.GetErrorMessage())
		}
	}
	return scanner.Err()
}

// streamEventsToWriter runs StreamEvents in the background and prints each event to out.
func streamEventsToWriter(ctx context.Context, c *client.Client, out io.Writer) {
	stream, err := c.StreamEvents(ctx)
	if err != nil {
		return
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			return
		}
		if ev == nil {
			continue
		}
		fmt.Fprintf(out, "[event] type=%s pid=%d tgid=%d cpu=%d\n",
			ev.GetEventType(), ev.GetPid(), ev.GetTgid(), ev.GetCpu())
	}
}

func isExitCommand(line string) bool {
	parts := strings.Fields(line)
	if len(parts) == 0 {
		return false
	}
	lower := strings.ToLower(parts[0])
	return lower == "quit" || lower == "exit" || lower == "q"
}

func writeResponse(out io.Writer, resp *proto.ExecuteResponse) {
	if resp == nil {
		return
	}
	if !resp.GetOk() {
		fmt.Fprintf(out, "%s\n", resp.GetErrorMessage())
		return
	}
	if resp.GetOutput() != "" {
		fmt.Fprintf(out, "%s\n", resp.GetOutput())
	}
}
