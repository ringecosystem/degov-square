package services

import (
	"testing"
	"time"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/types"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTestProposalService(t *testing.T) *ProposalService {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}

	if err := db.Exec(`
		CREATE TABLE dgv_proposal_tracking (
			id TEXT PRIMARY KEY,
			dao_code TEXT NOT NULL,
			chain_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			proposal_link TEXT NOT NULL,
			proposal_id TEXT NOT NULL,
			state TEXT NOT NULL,
			proposal_created_at DATETIME,
			proposal_at_block INTEGER NOT NULL,
			times_track INTEGER NOT NULL DEFAULT 0,
			time_next_track DATETIME,
			message TEXT,
			offset_tracking_vote INTEGER DEFAULT 0,
			fulfilled INTEGER DEFAULT 0,
			fulfilled_explain TEXT,
			fulfilled_at DATETIME,
			times_fulfill INTEGER DEFAULT 0,
			fulfill_errored INTEGER DEFAULT 0,
			ctime DATETIME NOT NULL,
			utime DATETIME
		)
	`).Error; err != nil {
		t.Fatalf("create proposal tracking table: %v", err)
	}

	return &ProposalService{db: db}
}

func seedProposalTracking(t *testing.T, service *ProposalService, proposal dbmodels.ProposalTracking) {
	t.Helper()

	if err := service.db.Create(&proposal).Error; err != nil {
		t.Fatalf("seed proposal tracking: %v", err)
	}
}

func TestTrackingStateProposalsWithoutTimesTrackLimitReturnsStalledRows(t *testing.T) {
	service := newTestProposalService(t)
	readyAt := time.Now().Add(-time.Hour)

	seedProposalTracking(t, service, dbmodels.ProposalTracking{
		ID:            "proposal-1",
		DaoCode:       "ring-dao",
		ChainId:       46,
		Title:         "Stalled active proposal",
		ProposalLink:  "https://gov.ringdao.com/proposal/1",
		ProposalID:    "0x1",
		State:         dbmodels.ProposalStateActive,
		TimesTrack:    10,
		TimeNextTrack: &readyAt,
		CTime:         time.Now(),
	})

	proposals, err := service.TrackingStateProposals(types.TrackingStateProposalsInput{
		DaoCode: "ring-dao",
		States:  []dbmodels.ProposalState{dbmodels.ProposalStateActive},
	})
	if err != nil {
		t.Fatalf("TrackingStateProposals returned error: %v", err)
	}

	if len(proposals) != 1 {
		t.Fatalf("expected stalled active proposal to remain trackable, got %d rows", len(proposals))
	}
}

func TestResetProposalTrackingStatusClearsRetryMetadata(t *testing.T) {
	service := newTestProposalService(t)
	nextTrackAt := time.Now().Add(2 * time.Hour)

	seedProposalTracking(t, service, dbmodels.ProposalTracking{
		ID:            "proposal-2",
		DaoCode:       "ring-dao",
		ChainId:       46,
		Title:         "Retry budget proposal",
		ProposalLink:  "https://gov.ringdao.com/proposal/2",
		ProposalID:    "0x2",
		State:         dbmodels.ProposalStateActive,
		TimesTrack:    7,
		TimeNextTrack: &nextTrackAt,
		Message:       "temporary rpc timeout",
		CTime:         time.Now(),
	})

	if err := service.ResetProposalTrackingStatus("0x2", "ring-dao"); err != nil {
		t.Fatalf("ResetProposalTrackingStatus returned error: %v", err)
	}

	var proposal dbmodels.ProposalTracking
	if err := service.db.Where("proposal_id = ? AND dao_code = ?", "0x2", "ring-dao").First(&proposal).Error; err != nil {
		t.Fatalf("reload proposal tracking: %v", err)
	}

	if proposal.TimesTrack != 0 {
		t.Fatalf("expected times_track reset to 0, got %d", proposal.TimesTrack)
	}
	if proposal.TimeNextTrack != nil {
		t.Fatalf("expected time_next_track cleared, got %v", proposal.TimeNextTrack)
	}
	if proposal.Message != "" {
		t.Fatalf("expected message cleared, got %q", proposal.Message)
	}
}

func TestUpdateProposalStateClearsRetryMetadata(t *testing.T) {
	service := newTestProposalService(t)
	nextTrackAt := time.Now().Add(2 * time.Hour)

	seedProposalTracking(t, service, dbmodels.ProposalTracking{
		ID:            "proposal-3",
		DaoCode:       "ring-dao",
		ChainId:       46,
		Title:         "State transition proposal",
		ProposalLink:  "https://gov.ringdao.com/proposal/3",
		ProposalID:    "0x3",
		State:         dbmodels.ProposalStateActive,
		TimesTrack:    5,
		TimeNextTrack: &nextTrackAt,
		Message:       "temporary rpc timeout",
		CTime:         time.Now(),
	})

	if err := service.UpdateProposalState("0x3", "ring-dao", dbmodels.ProposalStateExecuted); err != nil {
		t.Fatalf("UpdateProposalState returned error: %v", err)
	}

	var proposal dbmodels.ProposalTracking
	if err := service.db.Where("proposal_id = ? AND dao_code = ?", "0x3", "ring-dao").First(&proposal).Error; err != nil {
		t.Fatalf("reload proposal tracking: %v", err)
	}

	if proposal.State != dbmodels.ProposalStateExecuted {
		t.Fatalf("expected state EXECUTED, got %s", proposal.State)
	}
	if proposal.TimesTrack != 0 {
		t.Fatalf("expected times_track reset to 0, got %d", proposal.TimesTrack)
	}
	if proposal.TimeNextTrack != nil {
		t.Fatalf("expected time_next_track cleared, got %v", proposal.TimeNextTrack)
	}
	if proposal.Message != "" {
		t.Fatalf("expected message cleared, got %q", proposal.Message)
	}
}
