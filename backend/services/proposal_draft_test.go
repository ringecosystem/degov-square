package services

import (
	"errors"
	"strings"
	"testing"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newProposalDraftTestService(t *testing.T, features string) *ProposalDraftService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE dgv_dao (id TEXT PRIMARY KEY, code TEXT UNIQUE NOT NULL, chain_id INTEGER NOT NULL, state TEXT NOT NULL, features TEXT)`,
		`CREATE TABLE dgv_proposal_draft (
			id TEXT PRIMARY KEY, client_request_id TEXT NOT NULL, dao_code TEXT NOT NULL,
			chain_id INTEGER NOT NULL, user_id TEXT NOT NULL, user_address TEXT NOT NULL,
			title TEXT NOT NULL, payload TEXT NOT NULL, payload_version INTEGER NOT NULL,
			revision INTEGER NOT NULL, ctime DATETIME NOT NULL, utime DATETIME NOT NULL,
			UNIQUE (user_id, dao_code, client_request_id)
		)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create test table: %v", err)
		}
	}
	if err := db.Exec(`INSERT INTO dgv_dao (id, code, chain_id, state, features) VALUES (?, ?, ?, ?, ?)`, "dao-id", "demo", 46, dbmodels.DaoStateActive, features).Error; err != nil {
		t.Fatalf("seed DAO: %v", err)
	}
	return newProposalDraftService(db)
}

func saveNewProposalDraft(t *testing.T, service *ProposalDraftService, user *types.UserSessInfo, requestID, title string) *gqlmodels.ProposalDraft {
	t.Helper()
	draft, err := service.Save(user, gqlmodels.SaveProposalDraftInput{
		DaoCode: "demo", ClientRequestID: requestID, Title: title,
		Payload: `{"actions":[]}`, PayloadVersion: proposalDraftPayloadVersion,
	})
	if err != nil {
		t.Fatalf("Save(create): %v", err)
	}
	return draft
}

func TestProposalDraftMultipleDraftsAndIdempotentCreate(t *testing.T) {
	service := newProposalDraftTestService(t, `["proposal-drafts"]`)
	user := &types.UserSessInfo{Id: "owner", Address: "0xAa00000000000000000000000000000000000001"}

	first := saveNewProposalDraft(t, service, user, "request-1", " First ")
	retry := saveNewProposalDraft(t, service, user, "request-1", "ignored retry")
	second := saveNewProposalDraft(t, service, user, "request-2", "")
	if first.ID != retry.ID || retry.Title != "First" {
		t.Fatalf("idempotent retry = %#v, first = %#v", retry, first)
	}
	if second.ID == first.ID || second.Title != "Untitled draft" {
		t.Fatalf("second = %#v", second)
	}
	var count int64
	if err := service.db.Model(&dbmodels.ProposalDraft{}).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("draft count = %d, %v", count, err)
	}
	var stored dbmodels.ProposalDraft
	if err := service.db.Where("id = ?", first.ID).First(&stored).Error; err != nil {
		t.Fatalf("load stored draft: %v", err)
	}
	if stored.ChainID != 46 || stored.UserAddress != strings.ToLower(user.Address) {
		t.Fatalf("stored scope = %#v", stored)
	}
}

func TestProposalDraftOwnerScopeRevisionAndDelete(t *testing.T) {
	service := newProposalDraftTestService(t, `["proposal-drafts"]`)
	owner := &types.UserSessInfo{Id: "owner", Address: "0x0000000000000000000000000000000000000001"}
	other := &types.UserSessInfo{Id: "other", Address: "0x0000000000000000000000000000000000000002"}
	draft := saveNewProposalDraft(t, service, owner, "request-1", "Initial")

	_, err := service.Get(other, gqlmodels.ProposalDraftInput{DaoCode: "demo", DraftID: draft.ID})
	assertProposalDraftError(t, err, "draft_not_found_or_forbidden")
	one := int32(1)
	updated, err := service.Save(owner, gqlmodels.SaveProposalDraftInput{
		DaoCode: "demo", DraftID: &draft.ID, ClientRequestID: "request-1", Title: "Updated",
		Payload: `{"actions":[{"id":"1"}]}`, PayloadVersion: proposalDraftPayloadVersion, Revision: &one,
	})
	if err != nil || updated.Revision != 2 || updated.Title != "Updated" {
		t.Fatalf("Save(update) = %#v, %v", updated, err)
	}
	_, err = service.Save(owner, gqlmodels.SaveProposalDraftInput{
		DaoCode: "demo", DraftID: &draft.ID, ClientRequestID: "request-1", Title: "Stale",
		Payload: `{"actions":[]}`, PayloadVersion: proposalDraftPayloadVersion, Revision: &one,
	})
	var conflict *ProposalDraftError
	if !errors.As(err, &conflict) || conflict.Code != "draft_revision_conflict" || conflict.CurrentRevision != 2 {
		t.Fatalf("conflict = %#v, %v", conflict, err)
	}

	deleted, err := service.Delete(owner, gqlmodels.DeleteProposalDraftInput{DaoCode: "demo", DraftID: draft.ID})
	if err != nil || !deleted {
		t.Fatalf("Delete() = %v, %v", deleted, err)
	}
	deleted, err = service.Delete(owner, gqlmodels.DeleteProposalDraftInput{DaoCode: "demo", DraftID: draft.ID})
	if err != nil || !deleted {
		t.Fatalf("Delete(retry) = %v, %v", deleted, err)
	}
	_, err = service.Save(owner, gqlmodels.SaveProposalDraftInput{
		DaoCode: "demo", DraftID: &draft.ID, ClientRequestID: "request-1", Title: "Late",
		Payload: `{"actions":[]}`, PayloadVersion: proposalDraftPayloadVersion, Revision: &one,
	})
	assertProposalDraftError(t, err, "draft_not_found_or_forbidden")
}

func TestProposalDraftListPaginationOmitsPayload(t *testing.T) {
	service := newProposalDraftTestService(t, `["proposal-drafts"]`)
	user := &types.UserSessInfo{Id: "owner", Address: "0x0000000000000000000000000000000000000001"}
	base := time.Date(2026, 8, 18, 0, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return base }
	firstDraft := saveNewProposalDraft(t, service, user, "request-1", "First")
	service.now = func() time.Time { return base.Add(time.Minute) }
	secondDraft := saveNewProposalDraft(t, service, user, "request-2", "Second")

	one := int32(1)
	page1, err := service.List(user, gqlmodels.ProposalDraftsInput{DaoCode: "demo", First: &one})
	if err != nil || len(page1.Items) != 1 || page1.Items[0].ID != secondDraft.ID || page1.Items[0].Payload != nil || !page1.PageInfo.HasNextPage || page1.PageInfo.EndCursor == nil {
		t.Fatalf("List(page 1) = %#v, %v", page1, err)
	}
	paddedCursor := " " + *page1.PageInfo.EndCursor + "\n"
	page2, err := service.List(user, gqlmodels.ProposalDraftsInput{DaoCode: "demo", First: &one, After: &paddedCursor})
	if err != nil || len(page2.Items) != 1 || page2.Items[0].ID != firstDraft.ID || page2.PageInfo.HasNextPage {
		t.Fatalf("List(page 2) = %#v, %v", page2, err)
	}
	full, err := service.Get(user, gqlmodels.ProposalDraftInput{DaoCode: "demo", DraftID: firstDraft.ID})
	if err != nil || full.Payload == nil || *full.Payload != `{"actions":[]}` {
		t.Fatalf("Get() = %#v, %v", full, err)
	}
}

func TestProposalDraftFeatureAndPayloadValidation(t *testing.T) {
	service := newProposalDraftTestService(t, `[]`)
	user := &types.UserSessInfo{Id: "owner", Address: "0x0000000000000000000000000000000000000001"}
	_, err := service.List(user, gqlmodels.ProposalDraftsInput{DaoCode: "demo"})
	assertProposalDraftError(t, err, "dao_feature_disabled")
	if err := service.db.Model(&dbmodels.Dao{}).Where("code = ?", "demo").Update("features", `["proposal-drafts"]`).Error; err != nil {
		t.Fatalf("enable feature: %v", err)
	}

	for name, input := range map[string]gqlmodels.SaveProposalDraftInput{
		"invalid JSON":    {DaoCode: "demo", ClientRequestID: "one", Title: "x", Payload: `{`, PayloadVersion: 1},
		"unknown version": {DaoCode: "demo", ClientRequestID: "two", Title: "x", Payload: `{}`, PayloadVersion: 2},
		"oversized":       {DaoCode: "demo", ClientRequestID: "three", Title: "x", Payload: `"` + strings.Repeat("x", maxProposalDraftPayloadSize) + `"`, PayloadVersion: 1},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := service.Save(user, input)
			if err == nil {
				t.Fatal("Save() error = nil")
			}
		})
	}
	if err := service.db.Model(&dbmodels.Dao{}).Where("code = ?", "demo").Update("state", dbmodels.DaoStateInactive).Error; err != nil {
		t.Fatalf("deactivate DAO: %v", err)
	}
	_, err = service.List(user, gqlmodels.ProposalDraftsInput{DaoCode: "demo"})
	assertProposalDraftError(t, err, "dao_inactive")
}

func assertProposalDraftError(t *testing.T, err error, code string) {
	t.Helper()
	var draftErr *ProposalDraftError
	if !errors.As(err, &draftErr) || draftErr.Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}
