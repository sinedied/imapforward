package main

import (
	"reflect"
	"testing"
)

func TestNormalizeTargetLabels(t *testing.T) {
	got := normalizeTargetLabels("INBOX", []string{"Import/Work", "Import/Work", "INBOX", "", " Import/Personal "})
	want := []string{"Import/Work", "Import/Personal"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestLabelParents(t *testing.T) {
	got := labelParents("Import/Work/Clients")
	want := []string{"Import", "Import/Work"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestOrderedLabelsForCreation(t *testing.T) {
	got := orderedLabelsForCreation([]string{"Import/Work/Clients", "Import/Team", "Import/Work"})
	want := []string{"Import", "Import/Work", "Import/Work/Clients", "Import/Team"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
