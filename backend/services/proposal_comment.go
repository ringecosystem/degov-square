package services

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
)

const (
	proposalCommentsFeature  = "proposal-comments"
	maxCommentBodyRunes      = 10_000
	maxCommentsPageSize      = 100
	commentWritesPerMinute   = 20
	maxCommentLimiterEntries = 10_000
)

type ProposalCommentError struct {
	Code string
}

func (e *ProposalCommentError) Error() string { return e.Code }

type proposalCommentCursor struct {
	Time time.Time `json:"time"`
	ID   string    `json:"id"`
}

type proposalCommentWriteWindow struct {
	Minute time.Time
	Count  int
}

type proposalCommentLimiter struct {
	mu      sync.Mutex
	windows map[string]proposalCommentWriteWindow
}

type ProposalCommentService struct {
	db      *gorm.DB
	now     func() time.Time
	limiter *proposalCommentLimiter
}

func NewProposalCommentService() *ProposalCommentService {
	return newProposalCommentService(database.GetDB())
}

func newProposalCommentService(db *gorm.DB) *ProposalCommentService {
	return &ProposalCommentService{
		db:  db,
		now: time.Now,
		limiter: &proposalCommentLimiter{
			windows: make(map[string]proposalCommentWriteWindow),
		},
	}
}

func (s *ProposalCommentService) List(input gqlmodels.ProposalCommentsInput) (*gqlmodels.ProposalCommentPage, error) {
	proposalID, _, err := s.resolveScope(input.DaoCode, input.ProposalID)
	if err != nil {
		return nil, err
	}

	first := 20
	if input.First != nil {
		first = int(*input.First)
	}
	if first < 1 || first > maxCommentsPageSize {
		return nil, commentError("invalid_page_size")
	}

	query := s.db.Where("dao_code = ? AND proposal_id = ?", input.DaoCode, proposalID)
	if input.After != nil && strings.TrimSpace(*input.After) != "" {
		cursor, err := decodeProposalCommentCursor(strings.TrimSpace(*input.After))
		if err != nil {
			return nil, commentError("invalid_cursor")
		}
		query = query.Where("ctime > ? OR (ctime = ? AND id > ?)", cursor.Time, cursor.Time, cursor.ID)
	}

	var comments []dbmodels.ProposalComment
	if err := query.Order("ctime ASC").Order("id ASC").Limit(first + 1).Find(&comments).Error; err != nil {
		return nil, err
	}

	hasNextPage := len(comments) > first
	if hasNextPage {
		comments = comments[:first]
	}
	items := make([]*gqlmodels.ProposalComment, 0, len(comments))
	for i := range comments {
		items = append(items, proposalCommentToGraphQL(&comments[i]))
	}

	var endCursor *string
	if len(comments) > 0 {
		encoded, err := encodeProposalCommentCursor(comments[len(comments)-1])
		if err != nil {
			return nil, err
		}
		endCursor = &encoded
	}

	return &gqlmodels.ProposalCommentPage{
		Items: items,
		PageInfo: &gqlmodels.ProposalCommentPageInfo{
			EndCursor:   endCursor,
			HasNextPage: hasNextPage,
		},
	}, nil
}

func (s *ProposalCommentService) Create(user *types.UserSessInfo, input gqlmodels.CreateProposalCommentInput) (*gqlmodels.ProposalComment, error) {
	if err := s.allowWrite(user); err != nil {
		return nil, err
	}
	body, err := validateCommentBody(input.Body)
	if err != nil {
		return nil, err
	}
	proposalID, dao, err := s.resolveScope(input.DaoCode, input.ProposalID)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	comment := dbmodels.ProposalComment{
		ID:          utils.NextIDString(),
		DaoCode:     input.DaoCode,
		ChainID:     dao.ChainID,
		ProposalID:  proposalID,
		UserID:      user.Id,
		UserAddress: strings.ToLower(user.Address),
		ReplyToID:   input.ReplyToID,
		Body:        body,
		State:       dbmodels.ProposalCommentStateActive,
		CTime:       now,
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if input.ReplyToID != nil {
			var parent dbmodels.ProposalComment
			if err := tx.Where("id = ? AND dao_code = ? AND proposal_id = ?", *input.ReplyToID, input.DaoCode, proposalID).First(&parent).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return commentError("reply_parent_not_found")
				}
				return err
			}
			if parent.State != dbmodels.ProposalCommentStateActive {
				return commentError("reply_parent_deleted")
			}
		}
		return tx.Create(&comment).Error
	})
	if err != nil {
		return nil, err
	}
	return proposalCommentToGraphQL(&comment), nil
}

func (s *ProposalCommentService) Update(user *types.UserSessInfo, input gqlmodels.UpdateProposalCommentInput) (*gqlmodels.ProposalComment, error) {
	if err := s.allowWrite(user); err != nil {
		return nil, err
	}
	if _, err := s.requireFeature(input.DaoCode); err != nil {
		return nil, err
	}
	body, err := validateCommentBody(input.Body)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	result := s.db.Model(&dbmodels.ProposalComment{}).
		Where("id = ? AND dao_code = ? AND user_id = ? AND state = ?", input.CommentID, input.DaoCode, user.Id, dbmodels.ProposalCommentStateActive).
		Updates(map[string]any{"body": body, "utime": now})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, commentError("comment_not_found_or_forbidden")
	}

	var comment dbmodels.ProposalComment
	if err := s.db.Where("id = ?", input.CommentID).First(&comment).Error; err != nil {
		return nil, err
	}
	return proposalCommentToGraphQL(&comment), nil
}

func (s *ProposalCommentService) Delete(user *types.UserSessInfo, input gqlmodels.DeleteProposalCommentInput) (*gqlmodels.ProposalComment, error) {
	if err := s.allowWrite(user); err != nil {
		return nil, err
	}
	if _, err := s.requireFeature(input.DaoCode); err != nil {
		return nil, err
	}

	now := s.now().UTC()
	result := s.db.Model(&dbmodels.ProposalComment{}).
		Where("id = ? AND dao_code = ? AND user_id = ? AND state = ?", input.CommentID, input.DaoCode, user.Id, dbmodels.ProposalCommentStateActive).
		Updates(map[string]any{
			"body":  "",
			"state": dbmodels.ProposalCommentStateDeleted,
			"utime": now,
		})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, commentError("comment_not_found_or_forbidden")
	}

	var comment dbmodels.ProposalComment
	if err := s.db.Where("id = ?", input.CommentID).First(&comment).Error; err != nil {
		return nil, err
	}
	return proposalCommentToGraphQL(&comment), nil
}

func (s *ProposalCommentService) resolveScope(daoCode, rawProposalID string) (string, *dbmodels.Dao, error) {
	dao, err := s.requireFeature(daoCode)
	if err != nil {
		return "", nil, err
	}
	proposalIDs, err := proposalCommentIDCandidates(rawProposalID)
	if err != nil {
		return "", nil, commentError("invalid_proposal_id")
	}

	var proposal dbmodels.ProposalTracking
	if err := s.db.Where("dao_code = ? AND proposal_id IN ?", daoCode, proposalIDs).
		First(&proposal).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil, commentError("proposal_not_found")
		}
		return "", nil, err
	}
	return proposal.ProposalID, dao, nil
}

func (s *ProposalCommentService) requireFeature(daoCode string) (*dbmodels.Dao, error) {
	var dao dbmodels.Dao
	if err := s.db.Select("id", "code", "chain_id", "state", "features").Where("code = ?", daoCode).First(&dao).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, commentError("dao_not_found")
		}
		return nil, err
	}
	if dao.State != dbmodels.DaoStateActive {
		return nil, commentError("dao_inactive")
	}

	if strings.TrimSpace(dao.Features) == "" {
		return nil, commentError("dao_feature_disabled")
	}
	var features []string
	if err := json.Unmarshal([]byte(dao.Features), &features); err != nil {
		return nil, fmt.Errorf("decode DAO features: %w", err)
	}
	for _, feature := range features {
		if feature == proposalCommentsFeature {
			return &dao, nil
		}
	}
	return nil, commentError("dao_feature_disabled")
}

func (s *ProposalCommentService) allowWrite(user *types.UserSessInfo) error {
	if user == nil || strings.TrimSpace(user.Id) == "" || strings.TrimSpace(user.Address) == "" {
		return commentError("authentication_required")
	}

	key := strings.ToLower(user.Address)
	minute := s.now().UTC().Truncate(time.Minute)
	s.limiter.mu.Lock()
	defer s.limiter.mu.Unlock()
	window, tracked := s.limiter.windows[key]
	if !tracked && len(s.limiter.windows) >= maxCommentLimiterEntries {
		for address, candidate := range s.limiter.windows {
			if candidate.Minute.Before(minute) {
				delete(s.limiter.windows, address)
			}
		}
		if len(s.limiter.windows) >= maxCommentLimiterEntries {
			return commentError("comment_rate_limited")
		}
	}
	if !window.Minute.Equal(minute) {
		window = proposalCommentWriteWindow{Minute: minute}
	}
	if window.Count >= commentWritesPerMinute {
		return commentError("comment_rate_limited")
	}
	window.Count++
	s.limiter.windows[key] = window
	return nil
}

func proposalCommentIDCandidates(value string) ([]string, error) {
	value = strings.TrimSpace(value)
	base := 10
	if strings.HasPrefix(value, "0x") || strings.HasPrefix(value, "0X") {
		base = 16
		value = value[2:]
	}
	if value == "" {
		return nil, errors.New("empty proposal ID")
	}
	proposalID, ok := new(big.Int).SetString(value, base)
	if !ok || proposalID.Sign() < 0 || proposalID.BitLen() > 256 {
		return nil, errors.New("invalid proposal ID")
	}

	hexValue := proposalID.Text(16)
	candidates := []string{proposalID.String(), "0x" + hexValue}
	if len(hexValue) < 64 {
		candidates = append(candidates, "0x"+strings.Repeat("0", 64-len(hexValue))+hexValue)
	}
	return candidates, nil
}

func validateCommentBody(value string) (string, error) {
	body := strings.TrimSpace(value)
	if body == "" {
		return "", commentError("comment_body_required")
	}
	if !utf8.ValidString(body) || utf8.RuneCountInString(body) > maxCommentBodyRunes {
		return "", commentError("comment_body_too_long")
	}
	return body, nil
}

func encodeProposalCommentCursor(comment dbmodels.ProposalComment) (string, error) {
	payload, err := json.Marshal(proposalCommentCursor{Time: comment.CTime.UTC(), ID: comment.ID})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeProposalCommentCursor(value string) (*proposalCommentCursor, error) {
	payload, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	var cursor proposalCommentCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, err
	}
	if cursor.Time.IsZero() || cursor.ID == "" {
		return nil, errors.New("incomplete cursor")
	}
	return &cursor, nil
}

func proposalCommentToGraphQL(comment *dbmodels.ProposalComment) *gqlmodels.ProposalComment {
	state := gqlmodels.ProposalCommentState(comment.State)
	var body *string
	if comment.State == dbmodels.ProposalCommentStateActive {
		body = &comment.Body
	}
	return &gqlmodels.ProposalComment{
		ID:            comment.ID,
		DaoCode:       comment.DaoCode,
		ChainID:       int32(comment.ChainID),
		ProposalID:    comment.ProposalID,
		AuthorAddress: comment.UserAddress,
		ReplyToID:     comment.ReplyToID,
		Body:          body,
		State:         state,
		Ctime:         comment.CTime,
		Utime:         comment.UTime,
	}
}

func commentError(code string) error {
	return &ProposalCommentError{Code: code}
}
