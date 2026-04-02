package services

import (
	"testing"

	dbmodels "github.com/ringecosystem/degov-square/database/models"
)

func TestConvertToGqlDaoMapsOffsetTrackingProposal(t *testing.T) {
	t.Parallel()

	service := &DaoService{}
	dao := service.convertToGqlDao(dbmodels.Dao{
		Code:                "ring-dao",
		OffsetTrackingBlock: 42,
		Tags:                `["governance"]`,
		Domains:             `["ringdao.com"]`,
	})

	if got, want := dao.OffsetTrackingProposal, int32(42); got != want {
		t.Fatalf("OffsetTrackingProposal = %d, want %d", got, want)
	}
	if got, want := len(dao.Tags), 1; got != want {
		t.Fatalf("len(Tags) = %d, want %d", got, want)
	}
	if got, want := len(dao.Domains), 1; got != want {
		t.Fatalf("len(Domains) = %d, want %d", got, want)
	}
}
