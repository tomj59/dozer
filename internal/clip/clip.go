// Package clip puts copied text on the system clipboard.
//
// Two routes, used together:
//   - OSC 52, an escape sequence the host terminal turns into a clipboard
//     write. It works over ssh, but not every terminal supports it: macOS
//     Terminal.app ignores it.
//   - The local clipboard command (pbcopy, wl-copy, xclip, xsel), when dozer
//     runs on the machine whose clipboard you use, i.e. not over ssh.
package clip

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Command returns the local clipboard command for this environment, or nil
// if there's none (for example over ssh, where it would fill the remote
// machine's clipboard instead of yours). getenv is os.Getenv.
func Command(goos string, getenv func(string) string, look func(string) (string, error)) []string {
	if getenv("SSH_CONNECTION") != "" || getenv("SSH_TTY") != "" {
		return nil
	}
	has := func(name string) bool { _, err := look(name); return err == nil }
	switch {
	case goos == "darwin" && has("pbcopy"):
		return []string{"pbcopy"}
	case getenv("WAYLAND_DISPLAY") != "" && has("wl-copy"):
		return []string{"wl-copy"}
	case getenv("DISPLAY") != "" && has("xclip"):
		return []string{"xclip", "-selection", "clipboard"}
	case getenv("DISPLAY") != "" && has("xsel"):
		return []string{"xsel", "--clipboard", "--input"}
	}
	return nil
}

// Local runs the local clipboard command with text on stdin.
func Local(argv []string, text string) error {
	if len(argv) == 0 {
		return errors.New("no clipboard command")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return errors.New(argv[0] + ": " + s)
		}
		return errors.New(argv[0] + ": " + err.Error())
	}
	return nil
}

// GOOS is runtime.GOOS, for callers that want the default.
const GOOS = runtime.GOOS

// NoOSC52 reports whether the host terminal is known to ignore OSC 52.
func NoOSC52(getenv func(string) string) bool {
	return getenv("TERM_PROGRAM") == "Apple_Terminal"
}
