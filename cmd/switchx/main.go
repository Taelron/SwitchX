// Command switchx is the terminal time tracker for consulting teams.
//
// This is the M1 placeholder; subsequent issues replace it with the full
// bootstrap flow and TUI. See the M1 milestone in Linear:
// https://linear.app/taelron/project/switchx-0b0069bd1c04
//
// TAE-6 wires the config loader so the binary can verify config-file
// behavior end-to-end. TAE-13 replaces this minimal wiring with the
// full startup decision tree (config -> secret -> pool -> migrations
// -> home).
package main

import (
	"fmt"
	"os"

	"github.com/Taelron/SwitchX/internal/config"
)

func main() {
	if _, err := config.Load(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("switchx (placeholder) — config OK")
}
