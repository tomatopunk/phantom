package repl

import (
	"flag"
	"strings"
)

// Flags holds parsed CLI flags for the REPL.
type Flags struct {
	AgentAddr  string
	Token      string
	ScriptPath string
}

// ParseFlags parses args into Flags. Returns remaining args and any parse error.
func ParseFlags(args []string) (Flags, []string, error) {
	fs := flag.NewFlagSet("phantom", flag.ContinueOnError)
	var f Flags
	fs.StringVar(&f.AgentAddr, "agent", "", "agent address (host:port)")
	fs.StringVar(&f.Token, "token", "", "optional bearer token")
	fs.StringVar(&f.ScriptPath, "x", "", "run script file and exit")
	if err := fs.Parse(args); err != nil {
		return Flags{}, nil, err
	}
	f.AgentAddr = strings.TrimSpace(f.AgentAddr)
	f.Token = strings.TrimSpace(f.Token)
	return f, fs.Args(), nil
}
