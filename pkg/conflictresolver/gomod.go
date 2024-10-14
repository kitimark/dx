package conflictresolver

import (
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/kitimark/dx/pkg/exec"
	"golang.org/x/mod/modfile"
)

type GoModResolver struct{}

var goModConflictedFileLists = []string{"go.mod", "go.sum"}

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
		content := removeConflictAnnotation(string(b))
		if filename == "go.mod" {
			content, err = formatGoMod(filename, []byte(content))
			if err != nil {
				return err
			}
		}
		err = os.WriteFile(filename, []byte(content), 0)
		if err != nil {
			return err
		}
	}
	return nil
}

func removeConflictAnnotation(content string) string {
	var result []string
	for _, line := range strings.Split(content, "\n") {
		if !conflictContentPattern.MatchString(line) {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n")
}

var dontFixRetract modfile.VersionFixer = func(_, vers string) (string, error) {
	return vers, nil
}

func formatGoMod(filename string, b []byte) (string, error) {
	gomod, err := modfile.Parse(filename, b, dontFixRetract)
	if err != nil {
		return "", err
	}
	var requireMods []*modfile.Require
	var requireIndirectMods []*modfile.Require
	for _, mod := range gomod.Require {
		if mod.Indirect {
			requireIndirectMods = append(requireIndirectMods, mod)
		} else {
			requireMods = append(requireMods, mod)
		}
	}
	modSyntax := gomod.Syntax
	// clean up all require mods
	for _, stmt := range modSyntax.Stmt {
		switch expr := stmt.(type) {
		case *modfile.Line:
			if len(expr.Token) == 0 {
				continue
			}
			if expr.Token[0] == "require" {
				expr.Token = nil
				expr.Comments.Suffix = nil
			}
		case *modfile.LineBlock:
			if len(expr.Token) == 0 {
				continue
			}
			if expr.Token[0] == "require" {
				for _, l := range expr.Line {
					l.Token = nil
					l.Comments.Suffix = nil
				}
			}
		}
	}
	// assign require mods
	requireLineBlock := &modfile.LineBlock{
		Token: []string{"require"},
	}
	for _, m := range requireMods {
		l := &modfile.Line{
			Token:   []string{m.Mod.Path, m.Mod.Version},
			InBlock: true,
		}
		requireLineBlock.Line = append(requireLineBlock.Line, l)
	}
	modSyntax.Stmt = append(modSyntax.Stmt, requireLineBlock)

	requireIndirectLineBlock := &modfile.LineBlock{
		Token: []string{"require"},
	}
	for _, m := range requireIndirectMods {
		l := &modfile.Line{
			Comments: modfile.Comments{
				Suffix: []modfile.Comment{{
					Token:  "// indirect",
					Suffix: true,
				}},
			},
			Token:   []string{m.Mod.Path, m.Mod.Version},
			InBlock: true,
		}
		requireIndirectLineBlock.Line = append(requireIndirectLineBlock.Line, l)
	}
	modSyntax.Stmt = append(modSyntax.Stmt, requireIndirectLineBlock)

	modSyntax.Cleanup()
	b = modfile.Format(modSyntax)
	return string(b), nil
}
