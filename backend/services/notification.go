package services

import (
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
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
	}

	if err := s.db.Create(&events).Error; err != nil {
		return err
	}

	return nil
}

func (s *NotificationService) InspectEvent(input types.InspectNotificationEventInput) (*dbmodels.NotificationEvent, error) {
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

	for i := range records {
		records[i].ID = utils.NextIDString()
	}

	if err := s.db.Create(&records).Error; err != nil {
		return err
	}

	return nil
}
