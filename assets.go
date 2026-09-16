// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

func designAssetDirectory(designPath string) string {
	if strings.TrimSpace(designPath) == "" {
		return ""
	}
	return strings.TrimSuffix(designPath, filepath.Ext(designPath)) + ".assets"
}

func importImageAsset(sourcePath, designPath string) (string, *rosaline.Picture, error) {
	picture, err := rosaline.LoadImage(sourcePath)
	if err != nil {
		return "", nil, err
	}
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", nil, fmt.Errorf("read image asset: %w", err)
	}
	directory := designAssetDirectory(designPath)
	if directory == "" {
		return "", nil, fmt.Errorf("save the design before importing an image")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", nil, fmt.Errorf("create asset folder: %w", err)
	}
	name := safeAssetName(filepath.Base(sourcePath), data)
	if err := writeFileAtomically(filepath.Join(directory, name), data, 0o644); err != nil {
		return "", nil, fmt.Errorf("copy image asset: %w", err)
	}
	return name, picture, nil
}

func safeAssetName(filename string, data []byte) string {
	extension := strings.ToLower(filepath.Ext(filename))
	stem := strings.TrimSuffix(filename, filepath.Ext(filename))
	var cleaned strings.Builder
	for _, character := range stem {
		switch {
		case unicode.IsLetter(character), unicode.IsDigit(character):
			cleaned.WriteRune(unicode.ToLower(character))
		case cleaned.Len() > 0 && !strings.HasSuffix(cleaned.String(), "-"):
			cleaned.WriteByte('-')
		}
	}
	base := strings.Trim(cleaned.String(), "-")
	if base == "" {
		base = "image"
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%s-%x%s", base, hash[:4], extension)
}

func projectAssets(project *designProject) []string {
	seen := make(map[string]bool)
	var visit func(*designNode)
	visit = func(node *designNode) {
		if node == nil {
			return
		}
		if node.Kind == kindImage && node.Asset != "" {
			seen[node.Asset] = true
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	if project != nil {
		project.visitRoots(visit)
	}
	result := make([]string, 0, len(seen))
	for name := range seen {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func copyProjectAssets(project *designProject, generatedDirectory string) (created, updated []string, err error) {
	assets := projectAssets(project)
	if len(assets) == 0 {
		return nil, nil, nil
	}
	sourceDirectory := generatedDirectory + ".assets"
	targetDirectory := filepath.Join(generatedDirectory, "assets")
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create generated asset folder: %w", err)
	}
	for _, name := range assets {
		data, readErr := os.ReadFile(filepath.Join(sourceDirectory, name))
		if readErr != nil {
			return nil, nil, fmt.Errorf("read image asset %q: %w", name, readErr)
		}
		target := filepath.Join(targetDirectory, name)
		_, statErr := os.Stat(target)
		existed := statErr == nil
		if statErr != nil && !os.IsNotExist(statErr) {
			return nil, nil, fmt.Errorf("inspect generated image asset %q: %w", name, statErr)
		}
		if writeErr := writeFileAtomically(target, data, 0o644); writeErr != nil {
			return nil, nil, fmt.Errorf("write generated image asset %q: %w", name, writeErr)
		}
		path := filepath.ToSlash(filepath.Join("assets", name))
		if existed {
			updated = append(updated, path)
		} else {
			created = append(created, path)
		}
	}
	return created, updated, nil
}

func copyDesignAssets(project *designProject, oldDesignPath, newDesignPath string) error {
	if oldDesignPath == "" || oldDesignPath == newDesignPath {
		return nil
	}
	assets := projectAssets(project)
	if len(assets) == 0 {
		return nil
	}
	sourceDirectory := designAssetDirectory(oldDesignPath)
	targetDirectory := designAssetDirectory(newDesignPath)
	if err := os.MkdirAll(targetDirectory, 0o755); err != nil {
		return fmt.Errorf("create new design asset folder: %w", err)
	}
	for _, name := range assets {
		data, err := os.ReadFile(filepath.Join(sourceDirectory, name))
		if err != nil {
			return fmt.Errorf("read design asset %q: %w", name, err)
		}
		if err := writeFileAtomically(filepath.Join(targetDirectory, name), data, 0o644); err != nil {
			return fmt.Errorf("copy design asset %q: %w", name, err)
		}
	}
	return nil
}
