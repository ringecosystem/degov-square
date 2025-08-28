package types

import dbmodels "github.com/ringecosystem/degov-apps/database/models"

type ListSubscribeUserInput struct {
	Feature    dbmodels.SubscribeFeatureName
	Strategies []string
	DaoCode    string
	ProposalId *string
	Limit      int
	Offset     int
}

type ListSubscribedUserOutput struct {
	UserID      string
	UserAddress string
	ChainID     int
	DaoCode     string
}
