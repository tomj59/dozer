// Command dozer is a TUI multi-shell: several shells in one terminal,
// arranged in rows and columns. See docs/QUICKSTART.md and docs/SPEC.md.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/term"

	"github.com/tomj59/dozer/internal/app"
	"github.com/tomj59/dozer/internal/config"
	"github.com/tomj59/dozer/internal/emu"
	"github.com/tomj59/dozer/internal/layout"
	"github.com/tomj59/dozer/internal/pane"
)

var version = "dev"

// cmdList collects -p/-x flags in the order given.
type cmdList struct {
	specs *[]pane.Spec
	exec  bool
}

func (c cmdList) String() string { return "" }
func (c cmdList) Set(v string) error {
	s := pane.Spec{Run: v}
	if c.exec {
		s = pane.Spec{Exec: v}
	}
	*c.specs = append(*c.specs, s)
	return nil
}

type strList []string

func (s *strList) String() string     { return strings.Join(*s, " ") }
func (s *strList) Set(v string) error { *s = append(*s, v); return nil }

const usage = `usage: dozer [flags] [profile | config.yaml]

Several shells in one terminal. Prefix key Ctrl-a, then:
  ←↑↓→ / hjkl / 1-9 / o   focus a pane       z   zoom the focused pane
  H J K L   move the focused pane's border (repeatable)   =  reset sizes
  r   restart focused pane   R   restart every dead pane   x  kill focused pane
  t   rename pane   s   toggle status bar   C-l redraw   q quit   Esc cancel

Examples:
  dozer                          default 2,1 layout, your shell in every pane
  dozer -l 2,2,1                 five panes
  dozer -l 2,1 -p htop -p 'git log'   commands fill panes in reading order
  dozer -c examples/dev.yaml     load a config (see examples/)
  dozer dev                      load ~/.config/dozer/dev.yaml
  printf 'top\nping 1.1.1.1\n' | dozer   piped: one pane command per line
  dozer -l 2,1 -p htop --save my.yaml   turn a command line into a config file
  dozer -c my.yaml --package my-tool.sh  a runnable, shareable script with the config inside

Flags:
`

func main() {
	var (
		specs     []pane.Spec
		cfgFile   = flag.String("c", "", "config `file` (YAML); - reads it from stdin")
		layoutArg = flag.String("l", "", "layout shorthand, e.g. 2,1 or 2,2,1")
		heights   = flag.String("heights", "", "row heights, e.g. 60,40 (60 = 60%, also 12c and 2fr)")
		widths    strList
		prefix    = flag.String("prefix", "", "prefix key, e.g. C-b (default C-a)")
		quitAll   = flag.Bool("quit-when-all-exited", false, "quit once every pane has exited (default: stay, showing dead panes)")
		noMouse   = flag.Bool("no-mouse", false, "leave the mouse to the terminal (no click focus, wheel scrollback or pane-aware selection)")
		check     = flag.Bool("check", false, "validate and print the resolved configuration, then exit")
		save      = flag.String("save", "", "write the resolved configuration as YAML to `file` (- = stdout), then exit")
		pkg       = flag.String("package", "", "write a self-contained, runnable `script` (NAME.sh) with the configuration inside, then exit")
		cfgText   = flag.String("config-text", "", "the configuration itself, as YAML or JSON `text` (instead of a file)")
		emuName   = flag.String("emu", "charm", fmt.Sprintf("terminal emulator back end %v", emu.Names))
		showVer   = flag.Bool("version", false, "print version and exit")
	)
	flag.Var(cmdList{&specs, false}, "p", "run `command` in the next pane, then leave a prompt (repeatable)")
	flag.Var(cmdList{&specs, true}, "x", "exec `command` as the next pane's process (repeatable)")
	flag.Var(&widths, "widths", "column widths for one row, ROW:SIZES, e.g. 1:30,70 (repeatable)")
	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		flag.PrintDefaults()
	}
	flag.Parse()
	pane.Version = version
	if *showVer {
		fmt.Println("dozer", version)
		return
	}

	cfg, err := build(*cfgFile, *cfgText, flag.Args(), *layoutArg, *heights, widths, *prefix, *quitAll, specs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "dozer:", err)
		os.Exit(2)
	}
	if *noMouse {
		cfg.Mouse = false
	}
	if *check {
		fmt.Print(cfg.Describe())
		if *save == "" {
			return
		}
	}
	if *pkg != "" {
		if err := writePackage(cfg, *pkg); err != nil {
			fmt.Fprintln(os.Stderr, "dozer: --package:", err)
			os.Exit(1)
		}
		return
	}
	if *save != "" {
		hdr := fmt.Sprintf("Saved by dozer %s --save on %s.\nConfiguration only: layout, sizes, display settings and each pane's launch command.", version, time.Now().Format("2006-01-02 15:04"))
		out, err := cfg.YAML(hdr)
		if err == nil {
			if *save == "-" {
				_, err = os.Stdout.Write(out)
			} else {
				err = os.WriteFile(*save, out, 0o644)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "dozer: --save:", err)
			os.Exit(1)
		}
		if *save != "-" {
			fmt.Fprintf(os.Stderr, "dozer: saved %s (run it with: dozer -c %s)\n", *save, *save)
		}
		return
	}

	in := os.Stdin
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Stdin was piped (commands already read); the keyboard is the tty.
		tty, err := os.Open("/dev/tty")
		if err != nil {
			fmt.Fprintln(os.Stderr, "dozer: no terminal for keyboard input:", err)
			os.Exit(1)
		}
		defer tty.Close()
		in = tty
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintln(os.Stderr, "dozer: stdout is not a terminal")
		os.Exit(1)
	}
	if _, err := emu.New(*emuName, 1, 1); err != nil {
		fmt.Fprintln(os.Stderr, "dozer:", err)
		os.Exit(2)
	}
	if err := app.Run(cfg, app.Options{Emulator: *emuName, Version: version, Input: in}); err != nil {
		fmt.Fprintln(os.Stderr, "dozer:", err)
		os.Exit(1)
	}
}

// build resolves the launch config: file/profile, then flag overrides.
func build(cfgFile, cfgText string, args []string, layoutArg, heights string, widths []string,
	prefix string, quitAll bool, specs []pane.Spec) (*config.Config, error) {
	if (cfgFile != "" || cfgText != "") && len(args) > 0 {
		return nil, fmt.Errorf("give either -c FILE, --config-text or a profile name, not several")
	}
	if cfgFile != "" && cfgText != "" {
		return nil, fmt.Errorf("give either -c FILE or --config-text, not both")
	}
	if len(args) > 1 {
		return nil, fmt.Errorf("unexpected arguments %v (quote commands: -p 'git log')", args[1:])
	}
	path := cfgFile
	if path == "" && cfgText == "" {
		arg := ""
		if len(args) == 1 {
			arg = args[0]
		}
		// A bare `dozer -l …` or `dozer -p …` ignores auto-found defaults.
		if arg != "" || (layoutArg == "" && len(specs) == 0) {
			p, err := config.Find(arg)
			if err != nil {
				return nil, err
			}
			path = p
		}
	}
	cfg := config.Defaults()
	switch {
	case cfgText != "" || path == "-":
		// Config with no file behind it: --config-text, or -c - (stdin,
		// which is how packaged scripts run). Relative cwds resolve against
		// the directory dozer runs in.
		data, src := []byte(cfgText), "(--config-text)"
		if path == "-" {
			var err error
			if data, err = io.ReadAll(os.Stdin); err != nil {
				return nil, err
			}
			src = "(stdin)"
		}
		if y, ok := config.Unpack(data); ok { // a package piped in: dozer -c - < tool.sh
			data = y
		}
		c, err := config.Parse(data, "")
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src, err)
		}
		c.Source = src
		cfg = c
	case path != "":
		c, err := config.Load(path)
		if err != nil {
			return nil, err
		}
		cfg = c
	}

	// Piped stdin: one pane command per line (unless it carried the config).
	if path != "-" && !term.IsTerminal(int(os.Stdin.Fd())) {
		piped, err := readCommands(os.Stdin)
		if err != nil {
			return nil, err
		}
		specs = append(specs, piped...)
	}

	if layoutArg != "" {
		root, err := layout.Parse(layoutArg)
		if err != nil {
			return nil, err
		}
		cfg.Layout = root
		old := cfg.Panes
		cfg.Panes = make([]pane.Spec, root.Panes())
		copy(cfg.Panes, old) // keep list-form panes from the config, by position
	} else if cfg.Source == "" && len(specs) > len(cfg.Panes) && len(specs) <= 9 {
		// No layout given: fit the commands. 1-3 → one row, else two rows.
		n := len(specs)
		spec := fmt.Sprint(n)
		if n > 3 {
			spec = fmt.Sprintf("%d,%d", (n+1)/2, n/2)
		}
		root, _ := layout.Parse(spec)
		cfg.Layout = root
		cfg.Panes = make([]pane.Spec, n)
	}
	if heights != "" {
		sz, err := layout.ParseSizes(heights)
		if err != nil {
			return nil, err
		}
		if err := layout.SetHeights(cfg.Layout, sz); err != nil {
			return nil, err
		}
	}
	for _, w := range widths {
		row, sz, err := layout.ParseWidths(w)
		if err != nil {
			return nil, err
		}
		if err := layout.SetWidths(cfg.Layout, row, sz); err != nil {
			return nil, err
		}
	}
	if len(specs) > len(cfg.Panes) {
		return nil, fmt.Errorf("%d commands but the layout has %d panes", len(specs), len(cfg.Panes))
	}
	for i, s := range specs {
		// Keep the pane's config (cwd, env, shell); replace what it runs.
		p := cfg.Panes[i]
		p.Run, p.Exec, p.Restart, p.Title = s.Run, s.Exec, "", ""
		cfg.Panes[i] = p
	}
	if prefix != "" {
		b, err := config.ParsePrefix(prefix)
		if err != nil {
			return nil, err
		}
		cfg.Prefix = b
	}
	if quitAll {
		cfg.QuitWhenAllExited = true
	}
	return cfg, nil
}

func readCommands(r io.Reader) ([]pane.Spec, error) {
	var out []pane.Spec
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, pane.Spec{Run: line})
	}
	return out, sc.Err()
}

// writePackage writes cfg as a packaged script (docs/SPEC.md §4.12).
func writePackage(cfg *config.Config, path string) error {
	if !strings.HasSuffix(path, ".sh") {
		path += ".sh"
	}
	name, err := config.PackageName(path)
	if err != nil {
		return err
	}
	// Only overwrite our own packages, never an unrelated script.
	if old, err := os.ReadFile(path); err == nil {
		if _, ok := config.Unpack(old); !ok {
			return fmt.Errorf("%s exists and is not a dozer package; not overwriting", path)
		}
	}
	out, err := cfg.Package(name, version, time.Now().Format("2006-01-02"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, out, 0o755); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "dozer: packaged %s (run it with: %s)\n", path, runHint(path))
	return nil
}

func runHint(path string) string {
	if filepath.IsAbs(path) || strings.Contains(path, "/") {
		return path
	}
	return "./" + path
}
