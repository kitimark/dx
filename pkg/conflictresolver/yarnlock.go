package conflictresolver

import (
	"fmt"
	"github.com/kitimark/dx/pkg/exec"
	"log/slog"
	"slices"
)

type YarnLockResolver struct{}

var (
	yarnLockFileName    = "yarn.lock"
	packageJsonFileName = "package.json"
)

func (r *YarnLockResolver) Name() string {
	return "yarn lock"
}

func (r *YarnLockResolver) Detect(fileNames []string) bool {
	return slices.Contains(fileNames, yarnLockFileName)
}

func (r *YarnLockResolver) Resolve(fileNames []string) error {
	isStillConflict, err := isContentStillConflict(packageJsonFileName)
	if err != nil {
		return err
	}
	if isStillConflict {
		return fmt.Errorf("package.json file is still conflicted, resolve them first")
	}

	slog.Info("try to run yarn again")
	out, err := exec.OutputErr("yarn")
	if err != nil {
		slog.Error(out)
		return err
	}

	return nil
}
