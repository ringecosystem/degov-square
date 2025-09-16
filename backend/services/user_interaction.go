package services

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"

	"github.com/ringecosystem/degov-square/database"
	dbmodels "github.com/ringecosystem/degov-square/database/models"
	gqlmodels "github.com/ringecosystem/degov-square/graph/models"
	"github.com/ringecosystem/degov-square/internal/config"
	"github.com/ringecosystem/degov-square/internal/utils"
	"github.com/ringecosystem/degov-square/types"
)

type UserInteractionService struct {
	db              *gorm.DB
	daoService      *DaoService
	templateService *TemplateService
	otpCache        *cache.Cache
	rateLimitCache  *cache.Cache
	notifierService *NotifierService
	userService     *UserService
}

const ExpirationMinutes = 10

func NewUserInteractionService() *UserInteractionService {
	otpCache := cache.New(ExpirationMinutes*time.Minute, 5*time.Minute)
	rateLimitCache := cache.New(1*time.Minute, 2*time.Minute)

	return &UserInteractionService{
		db:              database.GetDB(),
		daoService:      NewDaoService(),
		templateService: NewTemplateService(),
		otpCache:        otpCache,
		rateLimitCache:  rateLimitCache,
		notifierService: NewNotifierService(),
		userService:     NewUserService(),
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

func (s *UserInteractionService) VerifyNotificationChannel(baseInput types.BasicInput[gqlmodels.VerifyNotificationChannelInput]) (*gqlmodels.VerifyNotificationChannelOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	cachedOTP, found := s.otpCache.Get(user.Id)
	if !found {
		return &gqlmodels.VerifyNotificationChannelOutput{
			Code:    1,
			Message: utils.StringPtr("OTP code has expired or does not exist"),
		}, nil
	}

	cachedOTPStr, ok := cachedOTP.(string)
	if !ok {
		return &gqlmodels.VerifyNotificationChannelOutput{
			Code:    1,
			Message: utils.StringPtr("Invalid OTP code format"),
		}, nil
	}

	if cachedOTPStr != input.OtpCode {
		return &gqlmodels.VerifyNotificationChannelOutput{
			Code:    1,
			Message: utils.StringPtr("Invalid OTP code"),
		}, nil
	}

	s.otpCache.Delete(user.Id)

	err := s.db.Delete(&dbmodels.NotificationChannel{}, "user_id = ? AND channel_type = ?", user.Id, input.Type).Error
	if err != nil {
		slog.Warn("error deleting existing unverified channel", "user_id", user.Id, "channel_type", input.Type, "err", err)
	}

	notificationChannel := dbmodels.NotificationChannel{
		ID:           utils.NextIDString(),
		UserID:       user.Id,
		UserAddress:  user.Address,
		Verified:     1,
		ChannelType:  dbmodels.NotificationChannelType(input.Type),
		ChannelValue: input.Value,
		CTime:        time.Now(),
	}
	if err := s.db.Create(&notificationChannel).Error; err != nil {
		return nil, err
	}

	return &gqlmodels.VerifyNotificationChannelOutput{
		Code: 0,
	}, nil
}

func (s *UserInteractionService) ResendOTP(baseInput types.BasicInput[gqlmodels.BaseNotificationChannelInput]) (*gqlmodels.ResendOTPOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	return s.resendOTPForChannel(types.BasicInput[*gqlmodels.BaseNotificationChannelInput]{
		User:  user,
		Input: &input,
	})
}

func (s *UserInteractionService) resendOTPForChannel(baseInput types.BasicInput[*gqlmodels.BaseNotificationChannelInput]) (*gqlmodels.ResendOTPOutput, error) {
	user := baseInput.User
	input := baseInput.Input

	if !config.GetAppEnv().IsDevelopment() {
		rateLimitKey := fmt.Sprintf("otp_rate_limit_%s", user.Id)

		if cachedTime, found := s.rateLimitCache.Get(rateLimitKey); found {
			if lastSentTime, ok := cachedTime.(time.Time); ok {
				elapsed := time.Since(lastSentTime)
				remaining := 60*time.Second - elapsed
				if remaining > 0 {
					remainingSeconds := int32(remaining.Seconds())
					return &gqlmodels.ResendOTPOutput{
						Code:      1,
						RateLimit: &remainingSeconds,
						Message:   utils.StringPtr(fmt.Sprintf("OTP can only be sent once per minute. Please try again in %d seconds", remainingSeconds)),
					}, nil
				}
			} else {
				return &gqlmodels.ResendOTPOutput{
					Code:      1,
					RateLimit: utils.Int32Ptr(60),
					Message:   utils.StringPtr("OTP can only be sent once per minute. Please try again later"),
				}, nil
			}
		}

		s.rateLimitCache.Set(rateLimitKey, time.Now(), 1*time.Minute)
	}

	ensName, err := s.userService.GetENSName(user.Address)
	if err != nil {
		slog.Warn("Failed to get ENS name", "address", user.Address, "err", err)
	}

	switch input.Type {
	case gqlmodels.NotificationChannelTypeEmail:
		otpCode, err := utils.NextOTPCode()
		if err != nil {
			return nil, fmt.Errorf("error generating OTP code: %w", err)
		}
		s.otpCache.Set(user.Id, otpCode, ExpirationMinutes*time.Minute)

		templateOutput, err := s.templateService.GenerateTemplateOTP(types.GenerateTemplateOTPInput{
			DegovSiteConfig: config.GetDegovSiteConfig(),
			OTP:             otpCode,
			Expiration:      ExpirationMinutes,
			UserAddress:     user.Address,
			EnsName:         ensName,
		})
		if err != nil {
			return nil, fmt.Errorf("error generating email content: %w", err)
		}
		if err := s.notifierService.Notify(types.NotifyInput{
			Type:     dbmodels.NotificationChannelTypeEmail,
			To:       input.Value,
			Template: templateOutput,
		}); err != nil {
			slog.Warn("Failed to notify", "err", err)
		}

		return &gqlmodels.ResendOTPOutput{
			Code:       0,
			Expiration: utils.Int32Ptr(3 * 60),
		}, nil

	case gqlmodels.NotificationChannelTypeWebhook:
		return &gqlmodels.ResendOTPOutput{
			Code:    0,
			Message: utils.StringPtr("this method do not need send OTP to verify"),
		}, nil

	default:
		return &gqlmodels.ResendOTPOutput{
			Code:    0,
			Message: utils.StringPtr("do not support this notification channel"),
		}, nil
	}

}

func (s UserInteractionService) ListChannel(
	baseInput types.BasicInput[types.ListChannelInput],
) ([]dbmodels.NotificationChannel, error) {
	user := baseInput.User
	var results []dbmodels.NotificationChannel

	query := s.db.Where("user_id = ?", user.Id)

	if baseInput.Input.Verified != nil {
		verified := 0
		if *baseInput.Input.Verified {
			verified = 1
		}
		query = query.Where("verified = ?", verified)
	}

	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}
