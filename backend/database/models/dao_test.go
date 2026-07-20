package dbmodels

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestLastTrackedProposalIDUsesTextColumn(t *testing.T) {
	field, ok := reflect.TypeOf(Dao{}).FieldByName("LastTrackedProposalID")
	if !ok {
		t.Fatal("Dao.LastTrackedProposalID field not found")
	}
	if got := field.Tag.Get("gorm"); !strings.Contains(got, "type:text") {
		t.Fatalf("LastTrackedProposalID gorm tag = %q, want type:text", got)
	}

	migration, err := os.ReadFile("../../migrations/000007_proposal_cursor_text.up.sql")
	if err != nil {
		t.Fatalf("read cursor migration: %v", err)
	}
	if got := string(migration); !strings.Contains(got, "ALTER COLUMN last_tracked_proposal_id TYPE text") {
		t.Fatalf("migration = %q, want last_tracked_proposal_id text conversion", got)
	}
}
