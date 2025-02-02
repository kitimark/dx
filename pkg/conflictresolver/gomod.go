package conflictresolver

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/kitimark/dx/pkg/exec"
	"github.com/kitimark/dx/pkg/utils"
	"golang.org/x/mod/modfile"
)

type GoModResolver struct{}

var goModConflictedFileLists = []string{"go.mod", "go.sum"}

func (r *GoModResolver) Name() string {
	return "go mod"
}

func (r *GoModResolver) Detect(fileNames []string) bool {
	for _, f := range fileNames {
		if slices.Contains(goModConflictedFileLists, f) {
			return true
		}
	}
	return false
}

func (r *GoModResolver) Resolve(fileNames []string) error {
	err := checkGoFilesConflictedIsResolved(fileNames)
	if err != nil {
		return err
	}
	err = resolveGoModConflicted(fileNames)
	if err != nil {
		return err
	}
	out, err := exec.OutputErr("go", "mod", "tidy")
	if err != nil {
		slog.Error(out)
		return err
	}
	return nil
}

var goFileRegex = regexp.MustCompile("(.*)\\.go")

func checkGoFilesConflictedIsResolved(conflictedFiles []string) error {
	var stillConflictedFiles []string
	for _, f := range conflictedFiles {
		if goFileRegex.MatchString(f) {
			isStillConflicted, err := isContentStillConflict(f)
			if err != nil {
				return err
			}
			if isStillConflicted {
				stillConflictedFiles = append(stillConflictedFiles, f)
			}
		}
	}

	if len(stillConflictedFiles) != 0 {
		return fmt.Errorf(`go files still conflicted, resolve them first
%s`, strings.Join(stillConflictedFiles, "\n"))
	}
	return nil
}

var conflictContentPattern = regexp.MustCompile("^(<<<<<<<|=======|>>>>>>>)(.*)")

func isContentStillConflict(file string) (bool, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(content), "\n") {
		if conflictContentPattern.MatchString(line) {
			return true, nil
		}
	}
	return false, nil
}

func resolveGoModConflicted(files []string) error {
	for _, filename := range files {
		if !slices.Contains(goModConflictedFileLists, filename) {
			continue
		}
		b, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		b = removeConflictAnnotation(b)
		if filename == "go.mod" {
			b, err = formatGoMod(filename, b)
			if err != nil {
				return err
			}
		}
		err = os.WriteFile(filename, b, 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeConflictAnnotation(b []byte) []byte {
	var result []string
	for _, line := range strings.Split(string(b), "\n") {
		if !conflictContentPattern.MatchString(line) {
			result = append(result, line)
		}
	}
	return []byte(strings.Join(result, "\n"))
}

var dontFixRetract modfile.VersionFixer = func(_, vers string) (string, error) {
	return vers, nil
}

func formatGoMod(filename string, b []byte) ([]byte, error) {
	gomod, err := modfile.Parse(filename, b, dontFixRetract)
	if err != nil {
		return nil, err
	}
	syntax := gomod.Syntax
	requireMods, requireIndirectMods := extractAllRequireMods(gomod)
	err = cleanupAllRequireMods(gomod)
	if err != nil {
		return nil, err
	}
	assignRequireMods(syntax, requireMods, false)
	assignRequireMods(syntax, requireIndirectMods, true)

	syntax.Cleanup()
	b = modfile.Format(syntax)
	return b, nil
}

func cleanupAllRequireMods(gomod *modfile.File) error {
	for _, req := range gomod.Require {
		// when go.mod has conflict. it might have a multiple module
		// that's same Path but difference Version
		// in the drop require if we drop the same path before
		// it will be empty path in the same module. we have to ignore this drop
		if req.Mod.Path == "" {
			continue
		}
		err := gomod.DropRequire(req.Mod.Path)
		if err != nil {
			return err
		}
	}
	return nil
}

func extractAllRequireMods(gomod *modfile.File) (requireMods []*modfile.Require, requireIndirectMods []*modfile.Require) {
	for _, mod := range gomod.Require {
		c := utils.ShallowPtrCopy(*mod)
		if mod.Indirect {
			requireIndirectMods = append(requireIndirectMods, c)
		} else {
			requireMods = append(requireMods, c)
		}
	}
	return
}

func assignRequireMods(modSyntax *modfile.FileSyntax, mods []*modfile.Require, indirect bool) {
	requireLineBlock := &modfile.LineBlock{
		Token: []string{"require"},
	}
	for _, m := range mods {
		l := &modfile.Line{
			Token:   []string{m.Mod.Path, m.Mod.Version},
			InBlock: true,
		}
		if indirect {
			l.Comments = modfile.Comments{
				Suffix: []modfile.Comment{{
					Token:  "// indirect",
					Suffix: true,
				}},
			}
		}
		requireLineBlock.Line = append(requireLineBlock.Line, l)
	}
	modSyntax.Stmt = append(modSyntax.Stmt, requireLineBlock)
}
