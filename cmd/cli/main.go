// Command cli is the Phantom client (REPL + gRPC).
package main

import (
	"log"
	"os"

	"github.com/tomatopunk/phantom/pkg/cli/repl"
)

func main() {
	if err := repl.Run(os.Args[1:]); err != nil {
		log.Fatalf("cli: %v", err)
	}
}
