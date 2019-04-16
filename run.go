package gocredits

import (
	"bytes"
	"os/exec"
	"strings"

	"golang.org/x/xerrors"
)

func run(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	bufErr := &bytes.Buffer{}
	cmd.Stderr = bufErr
	out, err := cmd.Output()
	if err != nil {
		return "", xerrors.Errorf("command %q failed with following output: %s: %w",
			strings.Join(append([]string{command}, args...), " "),
			bufErr.String(),
			err,
		)
	}
	return strings.TrimSpace(string(out)), nil
}
