package services

import (
	"math"
	"time"

	"gorm.io/gorm"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
)

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		db: database.GetDB(),
	}
}

func (s *NotificationService) SaveEvent(event dbmodels.NotificationEvent) error {
	return s.SaveEvents([]dbmodels.NotificationEvent{event})
}

func (s *NotificationService) SaveEvents(events []dbmodels.NotificationEvent) error {
	if len(events) == 0 {
		return nil
	}

	for i := range events {
		events[i].ID = utils.NextIDString()
		events[i].Reached = 0
		events[i].State = dbmodels.NotificationEventStatePending
		events[i].TimeNextExecute = time.Now()
	}

	if err := s.db.Create(&events).Error; err != nil {
		return err
	}

	return nil
}

func (s *NotificationService) InspectEventWithProposal(input types.InspectNotificationEventInput) (*dbmodels.NotificationEvent, error) {
	var event dbmodels.NotificationEvent
	query := s.db.Where("dao_code = ? AND proposal_id = ? AND type = ?", input.DaoCode, input.ProposalID, input.Type)

	// Add VoteID condition if provided
	if input.VoteID != nil {
		query = query.Where("vote_id = ?", *input.VoteID)
	}

	// Add States condition if provided
	if input.States != nil && len(*input.States) > 0 {
		query = query.Where("state IN ?", *input.States)
	}

	if err := query.First(&event).Error; err != nil {
		return nil, err
	}
	return &event, nil
}

func (s *NotificationService) StoreRecords(records []dbmodels.NotificationRecord) error {
	if len(records) == 0 {
		return nil
	}

	codes := make([]string, 0, len(records))
	for _, record := range records {
		if record.Code != "" {
			codes = append(codes, record.Code)
		}
	}

	if len(codes) == 0 {
		return nil
	}

	var existingCodes []string
	if err := s.db.
		Model(&dbmodels.NotificationRecord{}).
		Where("code IN ?", codes).
		Pluck("code", &existingCodes).
		Error; err != nil {
		return err
	}

	existingCodeSet := make(map[string]struct{}, len(existingCodes))
	for _, code := range existingCodes {
		existingCodeSet[code] = struct{}{}
	}

	recordsToCreate := make([]dbmodels.NotificationRecord, 0)
	for _, record := range records {
		if _, exists := existingCodeSet[record.Code]; !exists {
			recordsToCreate = append(recordsToCreate, record)
		}
	}

	if len(recordsToCreate) == 0 {
		return nil
	}

	for i := range recordsToCreate {
		recordsToCreate[i].ID = utils.NextIDString()
		recordsToCreate[i].TimeNextExecute = time.Now()
	}

	if err := s.db.Create(&recordsToCreate).Error; err != nil {
		return err
	}

	return nil
}

func (s *NotificationService) ListLimitEvents(input types.ListLimitEventsInput) ([]dbmodels.NotificationEvent, error) {
	var events []dbmodels.NotificationEvent
	query := s.db.Model(&dbmodels.NotificationEvent{})

	if input.States != nil && len(*input.States) > 0 {
		query = query.Where("state IN ?", *input.States)
	}

	query = query.Where("time_next_execute <= ?", time.Now())

	if err := query.Order("time_next_execute asc, ctime asc").Limit(input.Limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func (s *NotificationService) UpdateEventState(input types.UpdateEventStateInput) error {
	return s.db.
		Model(&dbmodels.NotificationEvent{}).Where("id = ?", input.ID).
		Updates(map[string]interface{}{
			"state": input.State,
			"utime": time.Now(),
		}).
		Error
}

func (s *NotificationService) UpdateEventRetryTimes(input types.UpdateEventRetryTimes) error {
	backoffMinutes := math.Pow(2, float64(input.TimesRetry))
	if backoffMinutes > 1440 {
		backoffMinutes = 1440
	}
	delay := time.Duration(backoffMinutes) * time.Minute
	NextExecutableTime := time.Now().Add(delay)

	updates := map[string]interface{}{
		"times_retry":       input.TimesRetry,
		"message":           input.Message,
		"utime":             time.Now(),
		"time_next_execute": NextExecutableTime,
	}

	if input.TimesRetry > 3 {
		updates["state"] = dbmodels.NotificationEventStateFailed
	}

	return s.db.Model(&dbmodels.NotificationEvent{}).Where("id = ?", input.ID).Updates(updates).Error
}

func (s *NotificationService) ListLimitRecords(input types.ListLimitRecordsInput) ([]dbmodels.NotificationRecord, error) {
	var records []dbmodels.NotificationRecord
	query := s.db.Model(&dbmodels.NotificationRecord{})

	if input.States != nil && len(*input.States) > 0 {
		query = query.Where("state IN ?", *input.States)
	}

	query = query.Where("time_next_execute <= ?", time.Now())

	if err := query.Order("time_next_execute asc, ctime asc").Limit(input.Limit).Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func (s *NotificationService) UpdateRecordState(input types.UpdateRecordStateInput) error {
	return s.db.
		Model(&dbmodels.NotificationRecord{}).
		Where("id = ?", input.ID).
		Updates(map[string]interface{}{
			"state": input.State,
			"utime": time.Now(),
		}).Error
}

func (s *NotificationService) UpdateRecordRetryTimes(input types.UpdateRecordRetryTimes) error {
	backoffMinutes := math.Pow(2, float64(input.TimesRetry))
	if backoffMinutes > 1440 {
		backoffMinutes = 1440
	}
	delay := time.Duration(backoffMinutes) * time.Minute
	NextExecutableTime := time.Now().Add(delay)

	updates := map[string]interface{}{
		"times_retry":       input.TimesRetry,
		"message":           input.Message,
		"time_next_execute": NextExecutableTime,
		"utime":             time.Now(),
	}

	if input.TimesRetry > 3 {
		updates["state"] = dbmodels.NotificationRecordStateSentFail
	}

	return s.db.Model(&dbmodels.NotificationRecord{}).Where("id = ?", input.ID).Updates(updates).Error
}
