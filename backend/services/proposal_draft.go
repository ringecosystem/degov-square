package services

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
)

const (
	proposalDraftsFeature       = "proposal-drafts"
	proposalDraftPayloadVersion = 1
	maxProposalDraftPayloadSize = 1 << 20
	maxProposalDraftTitleRunes  = 200
	maxProposalDraftPageSize    = 100
	maxClientRequestIDRunes     = 100
)

type ProposalDraftError struct {
	Code            string
	CurrentRevision int
}

func (e *ProposalDraftError) Error() string {
	if e.CurrentRevision > 0 {
		return fmt.Sprintf("%s:current_revision=%d", e.Code, e.CurrentRevision)
	}
	return e.Code
}

type proposalDraftCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type ProposalDraftService struct {
	db  *gorm.DB
	now func() time.Time
}

func NewProposalDraftService() *ProposalDraftService {
	return newProposalDraftService(database.GetDB())
}

func newProposalDraftService(db *gorm.DB) *ProposalDraftService {
	return &ProposalDraftService{db: db, now: time.Now}
}

func (s *ProposalDraftService) List(user *types.UserSessInfo, input gqlmodels.ProposalDraftsInput) (*gqlmodels.ProposalDraftPage, error) {
	if err := validateProposalDraftUser(user); err != nil {
		return nil, err
	}
	if _, err := s.requireFeature(input.DaoCode); err != nil {
		return nil, err
	}
	first := 20
	if input.First != nil {
		first = int(*input.First)
	}
	if first < 1 || first > maxProposalDraftPageSize {
		return nil, draftError("invalid_page_size")
	}

	query := s.db.Where("user_id = ? AND dao_code = ?", user.Id, input.DaoCode)
	if input.After != nil && strings.TrimSpace(*input.After) != "" {
		cursor, err := decodeProposalDraftCursor(strings.TrimSpace(*input.After))
		if err != nil {
			return nil, draftError("invalid_cursor")
		}
		query = query.Where("utime < ? OR (utime = ? AND id < ?)", cursor.Time, cursor.Time, cursor.ID)
	}

	var drafts []dbmodels.ProposalDraft
	if err := query.Select("id", "dao_code", "chain_id", "title", "payload_version", "revision", "ctime", "utime").
		Order("utime DESC").Order("id DESC").Limit(first + 1).Find(&drafts).Error; err != nil {
		return nil, err
	}
	hasNextPage := len(drafts) > first
	if hasNextPage {
		drafts = drafts[:first]
	}
	items := make([]*gqlmodels.ProposalDraft, 0, len(drafts))
	for i := range drafts {
		items = append(items, proposalDraftToGraphQL(&drafts[i], false))
	}
	var endCursor *string
	if len(drafts) > 0 {
		encoded, err := encodeProposalDraftCursor(drafts[len(drafts)-1])
		if err != nil {
			return nil, err
		}
		endCursor = &encoded
	}
	return &gqlmodels.ProposalDraftPage{
		Items:    items,
		PageInfo: &gqlmodels.ProposalDraftPageInfo{EndCursor: endCursor, HasNextPage: hasNextPage},
	}, nil
}

func (s *ProposalDraftService) Get(user *types.UserSessInfo, input gqlmodels.ProposalDraftInput) (*gqlmodels.ProposalDraft, error) {
	if err := validateProposalDraftUser(user); err != nil {
		return nil, err
	}
	if _, err := s.requireFeature(input.DaoCode); err != nil {
		return nil, err
	}
	var draft dbmodels.ProposalDraft
	if err := s.db.Where("id = ? AND dao_code = ? AND user_id = ?", input.DraftID, input.DaoCode, user.Id).First(&draft).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, draftError("draft_not_found_or_forbidden")
		}
		return nil, err
	}
	return proposalDraftToGraphQL(&draft, true), nil
}

func (s *ProposalDraftService) Save(user *types.UserSessInfo, input gqlmodels.SaveProposalDraftInput) (*gqlmodels.ProposalDraft, error) {
	if err := validateProposalDraftUser(user); err != nil {
		return nil, err
	}
	dao, err := s.requireFeature(input.DaoCode)
	if err != nil {
		return nil, err
	}
	title, err := normalizeProposalDraftTitle(input.Title)
	if err != nil {
		return nil, err
	}
	payload, err := validateProposalDraftPayload(input.Payload, int(input.PayloadVersion))
	if err != nil {
		return nil, err
	}
	now := s.now().UTC()

	if input.DraftID == nil {
		clientRequestID := strings.TrimSpace(input.ClientRequestID)
		if clientRequestID == "" || utf8.RuneCountInString(clientRequestID) > maxClientRequestIDRunes {
			return nil, draftError("invalid_client_request_id")
		}
		draft := dbmodels.ProposalDraft{
			ID: utils.NextIDString(), ClientRequestID: clientRequestID, DaoCode: input.DaoCode,
			ChainID: dao.ChainID, UserID: user.Id, UserAddress: strings.ToLower(user.Address),
			Title: title, Payload: payload, PayloadVersion: int(input.PayloadVersion), Revision: 1,
			CTime: now, UTime: now,
		}
		result := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "dao_code"}, {Name: "client_request_id"}},
			DoNothing: true,
		}).Create(&draft)
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 0 {
			var existing dbmodels.ProposalDraft
			if err := s.db.Where("user_id = ? AND dao_code = ? AND client_request_id = ?", user.Id, input.DaoCode, clientRequestID).First(&existing).Error; err != nil {
				return nil, err
			}
			draft = existing
		}
		return proposalDraftToGraphQL(&draft, true), nil
	}

	if input.Revision == nil || *input.Revision < 1 {
		return nil, draftError("draft_revision_required")
	}
	var draft dbmodels.ProposalDraft
	result := s.db.Model(&draft).Clauses(clause.Returning{}).
		Where("id = ? AND dao_code = ? AND user_id = ? AND revision = ?", *input.DraftID, input.DaoCode, user.Id, *input.Revision).
		Updates(map[string]any{
			"title": title, "payload": payload, "payload_version": int(input.PayloadVersion),
			"revision": gorm.Expr("revision + 1"), "utime": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		var current dbmodels.ProposalDraft
		if err := s.db.Select("revision").Where("id = ? AND dao_code = ? AND user_id = ?", *input.DraftID, input.DaoCode, user.Id).First(&current).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, draftError("draft_not_found_or_forbidden")
			}
			return nil, err
		}
		return nil, &ProposalDraftError{Code: "draft_revision_conflict", CurrentRevision: current.Revision}
	}
	return proposalDraftToGraphQL(&draft, true), nil
}

func (s *ProposalDraftService) Delete(user *types.UserSessInfo, input gqlmodels.DeleteProposalDraftInput) (bool, error) {
	if err := validateProposalDraftUser(user); err != nil {
		return false, err
	}
	if _, err := s.requireFeature(input.DaoCode); err != nil {
		return false, err
	}
	result := s.db.Where("id = ? AND dao_code = ? AND user_id = ?", input.DraftID, input.DaoCode, user.Id).Delete(&dbmodels.ProposalDraft{})
	if result.Error != nil {
		return false, result.Error
	}
	// Deletion is idempotent so confirmed proposal cleanup can be safely retried.
	return true, nil
}

func (s *ProposalDraftService) requireFeature(daoCode string) (*dbmodels.Dao, error) {
	var dao dbmodels.Dao
	if err := s.db.Select("id", "code", "chain_id", "state", "features").Where("code = ?", daoCode).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, draftError("dao_not_found")
		}
		return nil, err
	}
	if dao.State != dbmodels.DaoStateActive {
		return nil, draftError("dao_inactive")
	}
	if strings.TrimSpace(dao.Features) == "" {
		return nil, draftError("dao_feature_disabled")
	}
	var features []string
	if err := json.Unmarshal([]byte(dao.Features), &features); err != nil {
		return nil, fmt.Errorf("decode DAO features: %w", err)
	}
	for _, feature := range features {
		if feature == proposalDraftsFeature {
			return &dao, nil
		}
	}
	return nil, draftError("dao_feature_disabled")
}

func validateProposalDraftUser(user *types.UserSessInfo) error {
	if user == nil || strings.TrimSpace(user.Id) == "" || strings.TrimSpace(user.Address) == "" {
		return draftError("authentication_required")
	}
	return nil
}

func validateProposalDraftPayload(payload string, version int) (string, error) {
	if version != proposalDraftPayloadVersion {
		return "", draftError("unsupported_payload_version")
	}
	if len(payload) == 0 || len(payload) > maxProposalDraftPayloadSize || !json.Valid([]byte(payload)) {
		return "", draftError("invalid_draft_payload")
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(payload)); err != nil {
		return "", draftError("invalid_draft_payload")
	}
	if compact.Len() > maxProposalDraftPayloadSize {
		return "", draftError("invalid_draft_payload")
	}
	return compact.String(), nil
}

func normalizeProposalDraftTitle(value string) (string, error) {
	title := strings.TrimSpace(value)
	if title == "" {
		return "Untitled draft", nil
	}
	if utf8.RuneCountInString(title) > maxProposalDraftTitleRunes {
		return "", draftError("draft_title_too_long")
	}
	return title, nil
}

func proposalDraftToGraphQL(draft *dbmodels.ProposalDraft, includePayload bool) *gqlmodels.ProposalDraft {
	var payload *string
	if includePayload {
		payload = &draft.Payload
	}
	return &gqlmodels.ProposalDraft{
		ID: draft.ID, DaoCode: draft.DaoCode, ChainID: int32(draft.ChainID), Title: draft.Title,
		Payload: payload, PayloadVersion: int32(draft.PayloadVersion), Revision: int32(draft.Revision),
		Ctime: draft.CTime, Utime: draft.UTime,
	}
}

func encodeProposalDraftCursor(draft dbmodels.ProposalDraft) (string, error) {
	payload, err := json.Marshal(proposalDraftCursor{Time: draft.UTime, ID: draft.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeProposalDraftCursor(value string) (*proposalDraftCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	var cursor proposalDraftCursor
	if err := json.Unmarshal(payload, &cursor); err != nil || cursor.Time.IsZero() || strings.TrimSpace(cursor.ID) == "" {
		return nil, errors.New("invalid cursor")
	}
	return &cursor, nil
}

func draftError(code string) error {
	return &ProposalDraftError{Code: code}
}
