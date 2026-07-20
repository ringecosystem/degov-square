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

	upMigration, err := os.ReadFile("../../migrations/000007_proposal_cursor_text.up.sql")
	if err != nil {
		t.Fatalf("read cursor up migration: %v", err)
	}
	upSQL := string(upMigration)
	if !strings.Contains(upSQL, "ALTER COLUMN last_tracked_proposal_id TYPE text") {
		t.Fatalf("up migration = %q, want last_tracked_proposal_id text conversion", upSQL)
	}
	for _, destructive := range []string{"left(", "substring(", "substr(", "USING"} {
		if strings.Contains(strings.ToLower(upSQL), strings.ToLower(destructive)) {
			t.Fatalf("up migration = %q, must widen without data rewrite %q", upSQL, destructive)
		}
	}

	downMigration, err := os.ReadFile("../../migrations/000007_proposal_cursor_text.down.sql")
	if err != nil {
		t.Fatalf("read cursor down migration: %v", err)
	}
	downSQL := string(downMigration)
	for _, required := range []string{
		"IF EXISTS",
		"length(last_tracked_proposal_id) > 255",
		"RAISE EXCEPTION",
		"END;\n$$;",
		"ALTER COLUMN last_tracked_proposal_id TYPE varchar(255)",
	} {
		if !strings.Contains(downSQL, required) {
			t.Fatalf("down migration = %q, want explicit non-lossy rollback contract %q", downSQL, required)
		}
	}
	for _, destructive := range []string{"left(", "substring(", "substr("} {
		if strings.Contains(strings.ToLower(downSQL), destructive) {
			t.Fatalf("down migration = %q, must refuse rather than truncate with %q", downSQL, destructive)
		}
	}
}
