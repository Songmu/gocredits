package gocredits

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

func run(command string, args ...string) (string, error) {
	return runInDir("", command, args...)
}

func runInDir(dir, command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	bufErr := &bytes.Buffer{}
	cmd.Stderr = bufErr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("command %q failed with following output: %s: %w",
			strings.Join(append([]string{command}, args...), " "),
			bufErr.String(),
			err,
		)
	}
	return strings.TrimSpace(string(out)), nil
}
