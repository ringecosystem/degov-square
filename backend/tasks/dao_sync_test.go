package tasks

import (
	"reflect"
	"testing"
)

func TestParseFeatureList(t *testing.T) {
	got := parseFeatureList(" proposal-comments, proposal-drafts ,, ")
	want := []string{"proposal-comments", "proposal-drafts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseFeatureList() = %v, want %v", got, want)
	}
}

func TestMergeFeatures(t *testing.T) {
	got := mergeFeatures(
		[]string{"proposal-simulation", "proposal-comments"},
		[]string{"proposal-comments", "proposal-drafts"},
	)
	want := []string{"proposal-simulation", "proposal-comments", "proposal-drafts"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeFeatures() = %v, want %v", got, want)
	}
}
