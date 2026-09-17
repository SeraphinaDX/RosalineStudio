// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"reflect"
	"testing"
)

func TestDataLinesTrimAndSkipBlankLines(t *testing.T) {
	want := []string{"Rose", "Lavender", "Midnight"}
	if got := dataLines(" Rose\r\n\r\nLavender\n Midnight "); !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestRadioDataSupportsFriendlyLabelsAndValues(t *testing.T) {
	got := parseRadioData([]string{"Automatic = auto", "Light", "Another automatic = auto"})
	want := []radioDataChoice{{Label: "Automatic", Value: "auto"}, {Label: "Light", Value: "Light"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %#v, got %#v", want, got)
	}
}

func TestTableDataUsesFirstLineAsHeadings(t *testing.T) {
	columns, rows := parseTableData([]string{"Name | Type | Status", "Rosaline | Library | Ready", "Studio | Application"})
	if !reflect.DeepEqual(columns, []string{"Name", "Type", "Status"}) {
		t.Fatalf("unexpected columns: %v", columns)
	}
	if !reflect.DeepEqual(rows, [][]string{{"Rosaline", "Library", "Ready"}, {"Studio", "Application"}}) {
		t.Fatalf("unexpected rows: %v", rows)
	}
}

func TestTreeDataCombinesSharedPaths(t *testing.T) {
	roots := parseTreeData([]string{"Project/Forms/MainForm", "Project/Forms/SettingsForm", "Project/Assets", "Dependencies"})
	if len(roots) != 2 || roots[0].Label != "Project" || len(roots[0].Children) != 2 {
		t.Fatalf("unexpected tree roots: %#v", roots)
	}
	forms := roots[0].Children[0]
	if forms.Value != "Project/Forms" || len(forms.Children) != 2 || forms.Children[1].Value != "Project/Forms/SettingsForm" {
		t.Fatalf("shared tree path was not combined: %#v", forms)
	}
}
