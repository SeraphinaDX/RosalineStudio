// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type projectFileKind uint8

const (
	projectFileSource projectFileKind = iota
	projectFileGenerated
	projectFileProject
	projectFileAsset
)

type projectFile struct {
	Path string
	Kind projectFileKind
}

func (file projectFile) editable() bool {
	return file.Kind == projectFileSource
}

func (file projectFile) textFile() bool {
	switch strings.ToLower(filepath.Ext(file.Path)) {
	case ".go", ".mod", ".sum", ".md", ".txt", ".json", ".toml", ".yaml", ".yml":
		return true
	default:
		return filepath.Base(file.Path) == "go.mod" || filepath.Base(file.Path) == "go.sum"
	}
}

func scanProjectFiles(root string) ([]projectFile, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("the generated project folder is not ready")
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("inspect generated project: %w", err)
	}
	if !info.IsDir() {
		return nil, errors.New("the generated project path is not a folder")
	}
	result := make([]projectFile, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".rosaline-studio-") {
			return nil
		}
		relative = filepath.ToSlash(relative)
		kind := projectFileProject
		if relative == "assets" || strings.HasPrefix(relative, "assets/") {
			kind = projectFileAsset
		} else if strings.EqualFold(filepath.Ext(relative), ".go") {
			kind = projectFileSource
			if strings.HasSuffix(strings.ToLower(relative), "_generated.go") {
				kind = projectFileGenerated
			}
		}
		result = append(result, projectFile{Path: relative, Kind: kind})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan generated project: %w", err)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Kind != result[j].Kind {
			return result[i].Kind < result[j].Kind
		}
		return result[i].Path < result[j].Path
	})
	return result, nil
}

var goFilenamePattern = regexp.MustCompile(`^[A-Za-z0-9_]+\.go$`)

func validateGoFilename(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("enter a Go filename such as document.go")
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) {
		return errors.New("enter a filename without a folder")
	}
	if !goFilenamePattern.MatchString(name) || strings.HasPrefix(name, ".") {
		return errors.New("Go filenames may contain letters, numbers, and underscores and must end in .go")
	}
	if strings.HasSuffix(strings.ToLower(name), "_generated.go") {
		return errors.New("names ending in _generated.go are reserved for Studio")
	}
	return nil
}

func protectedDeveloperFile(path string) bool {
	switch filepath.ToSlash(path) {
	case "main.go", "handlers.go":
		return true
	default:
		return false
	}
}

func safeProjectPath(root, relative string) (string, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(relative) == "" {
		return "", errors.New("project path is empty")
	}
	relative = filepath.FromSlash(relative)
	if filepath.IsAbs(relative) {
		return "", errors.New("project paths must be relative")
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("project path leaves the generated project")
	}
	full := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("project path leaves the generated project")
	}
	return full, nil
}

type buildDiagnostic struct {
	File    string
	Line    int
	Column  int
	Message string
}

func (diagnostic buildDiagnostic) label() string {
	location := diagnostic.File + ":" + strconv.Itoa(diagnostic.Line)
	if diagnostic.Column > 0 {
		location += ":" + strconv.Itoa(diagnostic.Column)
	}
	return location + "  " + diagnostic.Message
}

var buildDiagnosticPattern = regexp.MustCompile(`^(.+?\.go):(\d+)(?::(\d+))?:\s*(.+)$`)

func parseBuildDiagnostics(output, root string) []buildDiagnostic {
	result := make([]buildDiagnostic, 0)
	seen := make(map[string]bool)
	for _, line := range strings.Split(strings.ReplaceAll(output, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		matches := buildDiagnosticPattern.FindStringSubmatch(line)
		if len(matches) == 0 {
			continue
		}
		path := strings.TrimPrefix(matches[1], "."+string(filepath.Separator))
		if filepath.IsAbs(path) && root != "" {
			if relative, err := filepath.Rel(root, path); err == nil {
				path = relative
			}
		}
		path = filepath.ToSlash(filepath.Clean(path))
		if path == "." || strings.HasPrefix(path, "../") {
			continue
		}
		lineNumber, _ := strconv.Atoi(matches[2])
		column, _ := strconv.Atoi(matches[3])
		diagnostic := buildDiagnostic{File: path, Line: lineNumber, Column: column, Message: strings.TrimSpace(matches[4])}
		key := diagnostic.label()
		if !seen[key] {
			seen[key] = true
			result = append(result, diagnostic)
		}
	}
	return result
}
