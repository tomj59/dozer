// Command dozer is a TUI multi-shell: several shells in one terminal,
// arranged in rows and columns. See docs/SPEC.md.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/tomj59/dozer/internal/app"
	"github.com/tomj59/dozer/internal/emu"
	"github.com/tomj59/dozer/internal/pane"
)

var version = "0.0.0-m0"

func main() {
	var (
		emuName = flag.String("emu", "charm", fmt.Sprintf("terminal emulator back end %v (M0 spike)", emu.Names))
		run     = flag.String("p", "", "command to run in the pane (run: semantics, you stay at a prompt after)")
		exe     = flag.String("x", "", "command that IS the pane process (exec: semantics)")
		showVer = flag.Bool("version", false, "print version and exit")
	)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: dozer [flags]\n\nM0 spike: one full-screen pane. Prefix is Ctrl-a; Ctrl-a q quits.\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVer {
		fmt.Println("dozer", version)
		return
	}

	code, err := app.Run(app.Options{
		Emulator: *emuName,
		Spec:     pane.Spec{Run: *run, Exec: *exe},
		Version:  version,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "dozer:", err)
		os.Exit(1)
	}
	if code != 0 {
		fmt.Fprintf(os.Stderr, "dozer: pane exited with status %d\n", code)
	}
}
