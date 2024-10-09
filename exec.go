package dx

import (
	"log/slog"
	"os/exec"
)

func execOutputErr(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	slog.Debug("exec command", "cmd", cmd.String())
	b, err := cmd.CombinedOutput()
	slog.Debug("exec result", "result", string(b))
	return string(b), err
}
