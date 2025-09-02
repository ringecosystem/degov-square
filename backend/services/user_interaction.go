package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-apps/database"
	dbmodels "github.com/ringecosystem/degov-apps/database/models"
	gqlmodels "github.com/ringecosystem/degov-apps/graph/models"
	"github.com/ringecosystem/degov-apps/internal/utils"
	"github.com/ringecosystem/degov-apps/types"
)

type UserInteractionService struct {
	db         *gorm.DB
	daoService *DaoService
	otpCache   *cache.Cache
}

func NewUserInteractionService() *UserInteractionService {
	otpCache := cache.New(3*time.Minute, 5*time.Minute)

	return &UserInteractionService{
		db:         database.GetDB(),
		daoService: NewDaoService(),
		otpCache:   otpCache,
	}
}

func (s *UserInteractionService) ModifyLikeDao(baseInput types.BasicInput[gqlmodels.ModifyLikeDaoInput]) (bool, error) {
	user := baseInput.User
	input := baseInput.Input

	_, err := s.daoService.Inspect(types.BasicInput[string]{
		User:  user,
		Input: input.DaoCode,
	})
	if err != nil {
		return false, fmt.Errorf("error inspecting DAO: %w", err)
	}

	var existingLikedDao dbmodels.UserLikedDao
	err = s.db.Where("user_id = ? AND dao_code = ?", user.Id, input.DaoCode).First(&existingLikedDao).Error
	isNotFoundError := false
	if err != nil {
		isNotFoundError = errors.Is(err, gorm.ErrRecordNotFound)
		if !isNotFoundError {
			return false, fmt.Errorf("error checking existing like: %w", err)
		}
	}

	switch baseInput.Input.Action {
	case gqlmodels.LikeActionLike:
		if isNotFoundError {
			// Create new like
			like := &dbmodels.UserLikedDao{
				ID:          utils.NextIDString(),
				DaoCode:     input.DaoCode,
				UserID:      user.Id,
				UserAddress: user.Address,
				CTime:       time.Now(),
			}

			if err := s.db.Create(like).Error; err != nil {
				return false, fmt.Errorf("error creating like: %w", err)
			}
			return true, nil
		}
	case gqlmodels.LikeActionUnlike:
		if !isNotFoundError {
			// Remove existing like
			result := s.db.Where("user_id = ? AND dao_code = ?", user.Id, input.DaoCode).Delete(&dbmodels.UserLikedDao{})
			if result.Error != nil {
				return false, fmt.Errorf("error removing like: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return false, errors.New("like not found")
			}
			return true, nil
		}
	}

	return true, nil
}

func (s *UserInteractionService) BindNotifyChannel(baseInput types.BasicInput[gqlmodels.BindNotifyChannelInput]) (*gqlmodels.BindNotifyChannelOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	var existingChannel dbmodels.NotificationChannel
	err := s.db.Where("user_id = ? AND channel_type = ?", user.Id, input.Type).First(&existingChannel).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("error checking existing channel: %w", err)
	}

	if err == nil {
		return nil, errors.New("channel type already exists for this user")
	}

	channelID := utils.NextIDString()

	channel := &dbmodels.NotificationChannel{
		ID:           channelID,
		UserID:       user.Id,
		UserAddress:  user.Address,
		Verified:     0,
		ChannelType:  dbmodels.NotificationChannelType(input.Type),
		ChannelValue: input.Value,
		CTime:        time.Now(),
	}

	if err := s.db.Create(channel).Error; err != nil {
		return nil, fmt.Errorf("error creating notification channel: %w", err)
	}

	sendOutput, err := s.resendOTPForChannel(types.BasicInput[*dbmodels.NotificationChannel]{
		User:  user,
		Input: channel,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to send otp: %w", err)
	}

	return &gqlmodels.BindNotifyChannelOutput{
		Expiration: sendOutput.Expiration,
	}, nil
}

func (s *UserInteractionService) VerifyNotififyChannel(baseInput types.BasicInput[gqlmodels.VerifyNotififyChannelInput]) (*gqlmodels.VerifyNotififyChannelOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	cachedOTP, found := s.otpCache.Get(input.ID)
	if !found {
		return &gqlmodels.VerifyNotififyChannelOutput{
			Code:    1,
			Message: utils.StringPtr("OTP code has expired or does not exist"),
		}, nil
	}

	cachedOTPStr, ok := cachedOTP.(string)
	if !ok {
		return &gqlmodels.VerifyNotififyChannelOutput{
			Code:    1,
			Message: utils.StringPtr("Invalid OTP code format"),
		}, nil
	}

	if cachedOTPStr != input.OtpCode {
		return &gqlmodels.VerifyNotififyChannelOutput{
			Code:    1,
			Message: utils.StringPtr("Invalid OTP code"),
		}, nil
	}

	s.otpCache.Delete(input.ID)

	var channel dbmodels.NotificationChannel
	err := s.db.Where("id = ? AND user_id = ?", input.ID, user.Id).First(&channel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &gqlmodels.VerifyNotififyChannelOutput{
				Code:    1,
				Message: utils.StringPtr("Notification channel not found"),
			}, nil
		}
		return nil, fmt.Errorf("error finding notification channel: %w", err)
	}

	if err := s.db.Model(&channel).Update("verified", 1).Error; err != nil {
		return nil, fmt.Errorf("error updating channel verification status: %w", err)
	}

	return &gqlmodels.VerifyNotififyChannelOutput{
		Code: 0,
	}, nil
}

func (s *UserInteractionService) ResendOTP(baseInput types.BasicInput[gqlmodels.ResendOTPInput]) (*gqlmodels.ResendOTPOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	var existingChannel dbmodels.NotificationChannel
	err := s.db.Where("user_id = ? AND channel_type = ? and channel_value = ?", user.Id, input.Type, input.Value).First(&existingChannel).Error
	if err != nil {
		return nil, fmt.Errorf("error checking existing channel: %w", err)
	}
	return s.resendOTPForChannel(types.BasicInput[*dbmodels.NotificationChannel]{
		User:  user,
		Input: &existingChannel,
	})
}

func (s *UserInteractionService) resendOTPForChannel(baseInput types.BasicInput[*dbmodels.NotificationChannel]) (*gqlmodels.ResendOTPOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	otpCode, err := utils.NextOTPCode()
	if err != nil {
		return nil, fmt.Errorf("error generating OTP code: %w", err)
	}
	s.otpCache.Set(input.ID, otpCode, 3*time.Minute)

	return &gqlmodels.ResendOTPOutput{
		Expiration: 3 * 60, // 180 seconds
	}, nil
}
