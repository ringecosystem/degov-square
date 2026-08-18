package services

import (
	"errors"
	"fmt"
	"testing"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newProposalCommentTestService(t *testing.T, features string) *ProposalCommentService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	for _, statement := range []string{
		`CREATE TABLE dgv_dao (id TEXT PRIMARY KEY, code TEXT UNIQUE NOT NULL, chain_id INTEGER NOT NULL, state TEXT NOT NULL, features TEXT)`,
		`CREATE TABLE dgv_proposal_tracking (id TEXT PRIMARY KEY, dao_code TEXT NOT NULL, proposal_id TEXT NOT NULL)`,
		`CREATE TABLE dgv_proposal_comment (id TEXT PRIMARY KEY, dao_code TEXT NOT NULL, chain_id INTEGER NOT NULL, proposal_id TEXT NOT NULL, user_id TEXT NOT NULL, user_address TEXT NOT NULL, reply_to_id TEXT, body TEXT NOT NULL, state TEXT NOT NULL, ctime DATETIME NOT NULL, utime DATETIME)`,
	} {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("create test table: %v", err)
		}
	}
	if err := db.Exec(`INSERT INTO dgv_dao (id, code, chain_id, state, features) VALUES (?, ?, ?, ?, ?)`, "dao-id", "demo", 46, dbmodels.DaoStateActive, features).Error; err != nil {
		t.Fatalf("seed DAO: %v", err)
	}
	if err := db.Exec(`INSERT INTO dgv_proposal_tracking (id, dao_code, proposal_id) VALUES (?, ?, ?)`, "proposal-row", "demo", "42").Error; err != nil {
		t.Fatalf("seed proposal: %v", err)
	}
	return newProposalCommentService(db)
}

func TestProposalCommentsFeatureScopeAndThreadLifecycle(t *testing.T) {
	service := newProposalCommentTestService(t, `[]`)
	user := &types.UserSessInfo{Id: "user-1", Address: "0xAa00000000000000000000000000000000000001"}

	_, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: "42"})
	assertProposalCommentError(t, err, "dao_feature_disabled")
	if err := service.db.Model(&dbmodels.Dao{}).Where("code = ?", "demo").Update("features", `["proposal-comments"]`).Error; err != nil {
		t.Fatalf("enable feature: %v", err)
	}

	root, err := service.Create(user, gqlmodels.CreateProposalCommentInput{
		DaoCode: "demo", ProposalID: "0x2a", Body: "  root comment  ",
	})
	if err != nil {
		t.Fatalf("Create(root): %v", err)
	}
	if root.ProposalID != "42" || root.Body == nil || *root.Body != "root comment" || root.AuthorAddress != "0xaa00000000000000000000000000000000000001" {
		t.Fatalf("root = %#v", root)
	}

	reply, err := service.Create(user, gqlmodels.CreateProposalCommentInput{
		DaoCode: "demo", ProposalID: "42", Body: "reply", ReplyToID: &root.ID,
	})
	if err != nil {
		t.Fatalf("Create(reply): %v", err)
	}
	_, err = service.Create(user, gqlmodels.CreateProposalCommentInput{
		DaoCode: "demo", ProposalID: "42", Body: "nested", ReplyToID: &reply.ID,
	})
	assertProposalCommentError(t, err, "reply_depth_exceeded")

	deleted, err := service.Delete(user, gqlmodels.DeleteProposalCommentInput{DaoCode: "demo", CommentID: root.ID})
	if err != nil {
		t.Fatalf("Delete(root): %v", err)
	}
	if deleted.State != gqlmodels.ProposalCommentStateDeleted || deleted.Body != nil {
		t.Fatalf("deleted = %#v", deleted)
	}

	page, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: "0x2A"})
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].Body != nil || page.Items[1].ReplyToID == nil {
		t.Fatalf("page = %#v", page)
	}
}

func TestProposalCommentsResolveHexTrackingID(t *testing.T) {
	service := newProposalCommentTestService(t, `["proposal-comments"]`)
	hexProposalID := "0x26e6249ecca1c50024b5baeca6084b6c6eeae30e2df78fa072f611cf09cd377e"
	if err := service.db.Exec(
		`INSERT INTO dgv_proposal_tracking (id, dao_code, proposal_id) VALUES (?, ?, ?)`,
		"hex-proposal-row", "demo", hexProposalID,
	).Error; err != nil {
		t.Fatalf("seed hex proposal: %v", err)
	}

	page, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: hexProposalID})
	if err != nil {
		t.Fatalf("List(hex proposal): %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("items = %d, want 0", len(page.Items))
	}
}

func TestProposalCommentsOwnershipPaginationAndValidation(t *testing.T) {
	service := newProposalCommentTestService(t, `["proposal-comments"]`)
	owner := &types.UserSessInfo{Id: "owner", Address: "0x0000000000000000000000000000000000000001"}
	other := &types.UserSessInfo{Id: "other", Address: "0x0000000000000000000000000000000000000002"}

	created := make([]*gqlmodels.ProposalComment, 0, 3)
	for i := range 3 {
		comment, err := service.Create(owner, gqlmodels.CreateProposalCommentInput{
			DaoCode: "demo", ProposalID: "42", Body: fmt.Sprintf("comment %d", i),
		})
		if err != nil {
			t.Fatalf("Create(%d): %v", i, err)
		}
		created = append(created, comment)
	}

	first := int32(2)
	page1, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: "42", First: &first})
	if err != nil {
		t.Fatalf("List(page 1): %v", err)
	}
	if len(page1.Items) != 2 || !page1.PageInfo.HasNextPage || page1.PageInfo.EndCursor == nil {
		t.Fatalf("page 1 = %#v", page1)
	}
	page2, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: "42", First: &first, After: page1.PageInfo.EndCursor})
	if err != nil {
		t.Fatalf("List(page 2): %v", err)
	}
	if len(page2.Items) != 1 || page2.PageInfo.HasNextPage {
		t.Fatalf("page 2 = %#v", page2)
	}
	paddedCursor := " \t" + *page1.PageInfo.EndCursor + "\n"
	paddedPage, err := service.List(gqlmodels.ProposalCommentsInput{DaoCode: "demo", ProposalID: "42", First: &first, After: &paddedCursor})
	if err != nil || len(paddedPage.Items) != 1 {
		t.Fatalf("List(padded cursor) = %#v, %v", paddedPage, err)
	}

	_, err = service.Update(other, gqlmodels.UpdateProposalCommentInput{DaoCode: "demo", CommentID: created[0].ID, Body: "stolen"})
	assertProposalCommentError(t, err, "comment_not_found_or_forbidden")
	updated, err := service.Update(owner, gqlmodels.UpdateProposalCommentInput{DaoCode: "demo", CommentID: created[0].ID, Body: "updated"})
	if err != nil || updated.Body == nil || *updated.Body != "updated" || updated.Utime == nil {
		t.Fatalf("Update(owner) = %#v, %v", updated, err)
	}

	_, err = service.Create(owner, gqlmodels.CreateProposalCommentInput{DaoCode: "demo", ProposalID: "missing", Body: "body"})
	assertProposalCommentError(t, err, "invalid_proposal_id")
	_, err = service.Create(owner, gqlmodels.CreateProposalCommentInput{DaoCode: "demo", ProposalID: "43", Body: "body"})
	assertProposalCommentError(t, err, "proposal_not_found")
	_, err = service.Create(owner, gqlmodels.CreateProposalCommentInput{DaoCode: "demo", ProposalID: "42", Body: "  "})
	assertProposalCommentError(t, err, "comment_body_required")
}

func TestProposalCommentWriteRateLimit(t *testing.T) {
	service := newProposalCommentTestService(t, `["proposal-comments"]`)
	user := &types.UserSessInfo{Id: "owner", Address: "0x0000000000000000000000000000000000000001"}
	for i := 0; i < commentWritesPerMinute; i++ {
		if err := service.allowWrite(user); err != nil {
			t.Fatalf("allowWrite(%d): %v", i, err)
		}
	}
	assertProposalCommentError(t, service.allowWrite(user), "comment_rate_limited")
}

func TestProposalCommentWriteRateLimitBoundsTrackedWallets(t *testing.T) {
	service := newProposalCommentTestService(t, `["proposal-comments"]`)
	minute := service.now().UTC().Truncate(time.Minute)
	for i := 0; i < maxCommentLimiterEntries; i++ {
		service.limiter.windows[fmt.Sprintf("wallet-%d", i)] = proposalCommentWriteWindow{Minute: minute, Count: 1}
	}
	user := &types.UserSessInfo{Id: "new-owner", Address: "0x0000000000000000000000000000000000000001"}
	assertProposalCommentError(t, service.allowWrite(user), "comment_rate_limited")
	if got := len(service.limiter.windows); got != maxCommentLimiterEntries {
		t.Fatalf("tracked wallets = %d, want %d", got, maxCommentLimiterEntries)
	}
}

func assertProposalCommentError(t *testing.T, err error, code string) {
	t.Helper()
	var commentErr *ProposalCommentError
	if !errors.As(err, &commentErr) || commentErr.Code != code {
		t.Fatalf("error = %v, want %s", err, code)
	}
}
