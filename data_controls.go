// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"
)

type radioDataChoice struct {
	Label string
	Value string
}

type treeDataItem struct {
	Label    string
	Value    string
	Children []*treeDataItem
}

func dataKind(kind widgetKind) bool {
	switch kind {
	case kindList, kindTable, kindTree, kindRadioGroup:
		return true
	default:
		return false
	}
}

func dataLines(value string) []string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	result := make([]string, 0)
	for _, line := range strings.Split(value, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func parseRadioData(lines []string) []radioDataChoice {
	result := make([]radioDataChoice, 0, len(lines))
	seen := make(map[string]bool)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		label, value := line, line
		if before, after, found := strings.Cut(line, "="); found {
			label, value = strings.TrimSpace(before), strings.TrimSpace(after)
			if label == "" {
				label = value
			}
			if value == "" {
				value = label
			}
		}
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, radioDataChoice{Label: label, Value: value})
	}
	return result
}

func splitTableCells(line string) []string {
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func parseTableData(lines []string) (columns []string, rows [][]string) {
	clean := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			clean = append(clean, line)
		}
	}
	if len(clean) == 0 {
		return []string{"Value"}, nil
	}
	columns = splitTableCells(clean[0])
	for index, column := range columns {
		if column == "" {
			columns[index] = fmt.Sprintf("Column %d", index+1)
		}
	}
	for _, line := range clean[1:] {
		rows = append(rows, splitTableCells(line))
	}
	return columns, rows
}

func parseTreeData(lines []string) []*treeDataItem {
	roots := make([]*treeDataItem, 0)
	for _, line := range lines {
		parts := strings.Split(strings.TrimSpace(line), "/")
		clean := make([]string, 0, len(parts))
		for _, part := range parts {
			if part = strings.TrimSpace(part); part != "" {
				clean = append(clean, part)
			}
		}
		if len(clean) == 0 {
			continue
		}
		children := &roots
		path := make([]string, 0, len(clean))
		for _, part := range clean {
			path = append(path, part)
			var item *treeDataItem
			for _, candidate := range *children {
				if candidate.Label == part {
					item = candidate
					break
				}
			}
			if item == nil {
				item = &treeDataItem{Label: part, Value: strings.Join(path, "/")}
				*children = append(*children, item)
			}
			children = &item.Children
		}
	}
	return roots
}

func dataHelp(kind widgetKind) string {
	switch kind {
	case kindList:
		return "Enter one list item per line."
	case kindTable:
		return "Use | between cells. The first line contains column headings; later lines are preview rows."
	case kindTree:
		return "Enter one path per line and use / for nesting, such as Project/Forms/MainForm."
	case kindRadioGroup:
		return "Enter one choice per line. Use Label = value when the stored value should differ from its label."
	default:
		return "The selected control does not use designer data."
	}
}
