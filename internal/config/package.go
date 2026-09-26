package config

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// A packaged workspace is a POSIX sh script with the YAML config embedded
// in it (docs/SPEC.md §4.12). Nothing is read from or written to disk when
// it runs: the script pipes its YAML to `dozer -c -`, the same as
//
//	dozer -c - < config.yaml
//
// so the script is the whole package. It needs only a dozer binary (on
// PATH, or $DOZER_BIN).
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

// Package renders a runnable script embedding the config. cwd and env are
// written as the config's author wrote them, so relative cwds resolve
// against the folder the script is run from and ~/$HOME resolve for
// whoever runs it.
func (c *Config) Package(name, version, date string) ([]byte, error) {
	cp := *c
	cp.Name = name
	body, err := cp.YAML("")
	if err != nil {
		return nil, err
	}
	if bytes.Contains(body, []byte("\n"+yamlClose+"\n")) || bytes.HasPrefix(body, []byte(yamlClose+"\n")) {
		return nil, fmt.Errorf("config contains a line %q, which would end the embedded YAML", yamlClose)
	}
	var b strings.Builder
	fmt.Fprintf(&b, `#!/bin/sh
# %[1]s: a dozer workspace, packaged by dozer %[2]s on %[3]s.
# Self-contained: the configuration is inside this script and is piped
# straight to dozer (dozer -c -). Nothing else is read or written.
#
#   ./%[1]s.sh                  launch it; extra dozer flags pass through,
#                               e.g. ./%[1]s.sh --prefix C-b
#   ./%[1]s.sh --show-config    print the embedded config
#
# To change it: ./%[1]s.sh --show-config > %[1]s.yaml, edit, then
#   dozer -c %[1]s.yaml --package %[1]s.sh
#
# Needs dozer on your PATH (or DOZER_BIN=/path/to/dozer): https://github.com/tomj59/dozer

yaml() {
%[4]s
%[5]s%[6]s
}

case "${1:-}" in
--show-config) yaml; exit 0 ;;
esac

# Check the environment.
dozer=${DOZER_BIN:-dozer}
if ! command -v "$dozer" >/dev/null 2>&1; then
	echo "%[1]s: needs dozer on your PATH (https://github.com/tomj59/dozer), or set DOZER_BIN" >&2
	exit 127
fi
if [ ! -t 1 ]; then
	echo "%[1]s: needs to run in a terminal" >&2
	exit 1
fi

# Run: config on stdin; dozer reads the keyboard from the terminal.
yaml | "$dozer" -c - "$@"
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
