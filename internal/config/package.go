package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// A packaged workspace is a POSIX sh script with the YAML config embedded
// in it (docs/SPEC.md §4.12). It is the frozen, shareable form of a config:
// copy it anywhere and run it. It needs a dozer binary on PATH (or in
// $DOZER_BIN).
const (
	yamlOpen  = "cat <<'DOZER_YAML'"
	yamlClose = "DOZER_YAML"
)

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// PackageName derives the workspace name from a script path:
// "tools/my-tool.sh" → "my-tool".
func PackageName(path string) (string, error) {
	name := strings.TrimSuffix(filepath.Base(path), ".sh")
	if !nameRE.MatchString(name) {
		return "", fmt.Errorf("package name %q: use letters, digits, '.', '_' or '-'", name)
	}
	return name, nil
}

// Package renders a runnable script embedding the config. dir is the
// folder the script will live in: cwd paths inside it are written relative
// to the script, and paths under $HOME as ~/…, so the script works when
// copied to another folder or shared with another user.
func (c *Config) Package(name, dir, version, date string) ([]byte, error) {
	cp := *c
	cp.Name = name
	home, _ := os.UserHomeDir()
	absDir, _ := filepath.Abs(dir)
	body, err := cp.yaml("", func(p string) string { return portablePath(p, absDir, home) })
	if err != nil {
		return nil, err
	}
	if bytes.Contains(body, []byte("\n"+yamlClose+"\n")) || bytes.HasPrefix(body, []byte(yamlClose+"\n")) {
		return nil, fmt.Errorf("config contains a line %q, which would end the embedded YAML", yamlClose)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `#!/bin/sh
# %[1]s: a dozer workspace, packaged by dozer %[2]s on %[3]s.
#
#   ./%[1]s.sh                       launch it (extra dozer flags are passed on,
#                                    e.g. ./%[1]s.sh --prefix C-b)
#   ./%[1]s.sh --show-config         print the embedded config
#
# To change it: ./%[1]s.sh --show-config > %[1]s.yaml, edit that file, then
#   dozer -c %[1]s.yaml --package %[1]s.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer
# Relative cwd paths resolve against this script's folder.
set -e

yaml() {
%[4]s
%[5]s%[6]s
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "%[1]s: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
here=$(cd "$(dirname "$0")" && pwd)
exec "$dozer" --base "$here" -c /dev/fd/3 "$@" 3<<%[6]s
$(yaml)
%[6]s
`, name, version, date, yamlOpen, body, yamlClose)
	return []byte(b.String()), nil
}

// Unpack extracts the YAML embedded in a packaged script.
func Unpack(script []byte) ([]byte, bool) {
	if !bytes.HasPrefix(script, []byte("#!")) {
		return nil, false
	}
	start := bytes.Index(script, []byte("\n"+yamlOpen+"\n"))
	if start < 0 {
		return nil, false
	}
	rest := script[start+len(yamlOpen)+2:]
	end := bytes.Index(rest, []byte("\n"+yamlClose+"\n"))
	if end < 0 {
		return nil, false
	}
	return rest[:end+1], true
}

// portablePath rewrites a cwd for sharing: inside dir → relative to it,
// under home → ~/…, otherwise unchanged.
func portablePath(p, dir, home string) string {
	if p == "" || !filepath.IsAbs(p) {
		return p
	}
	if rel, err := filepath.Rel(dir, p); err == nil && !strings.HasPrefix(rel, "..") {
		if rel == "." {
			return "."
		}
		return "./" + rel
	}
	if home != "" {
		if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
			if rel == "." {
				return "~"
			}
			return "~/" + rel
		}
	}
	return p
}
