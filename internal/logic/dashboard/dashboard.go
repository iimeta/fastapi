package dashboard

import (
	"context"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
)

type sDashboard struct{}

func init() {
	service.RegisterDashboard(New())
}

func New() service.IDashboard {
	return &sDashboard{}
}

// Subscription
func (s *sDashboard) Subscription(ctx context.Context) (*model.DashboardSubscriptionRes, error) {

	quota := service.Session().GetUser(ctx).Quota

	if service.Session().GetAppIsLimitQuota(ctx) {
		quota = service.Session().GetApp(ctx).Quota
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {
		quota = service.Session().GetKey(ctx).Quota
	}

	return &model.DashboardSubscriptionRes{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       (float64(quota) / consts.USD_QUOTA_UNIT) * 100,
		HardLimitUSD:       (float64(quota) / consts.USD_QUOTA_UNIT) * 100,
		SystemHardLimitUSD: (float64(quota) / consts.USD_QUOTA_UNIT) * 100,
		AccessUntil:        0,
	}, nil
}

// Usage
func (s *sDashboard) Usage(ctx context.Context) (*model.DashboardUsageRes, error) {

	quota := service.Session().GetUser(ctx).Quota

	if service.Session().GetAppIsLimitQuota(ctx) {
		quota = service.Session().GetApp(ctx).Quota
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {
		quota = service.Session().GetKey(ctx).Quota
	}

	return &model.DashboardUsageRes{
		Object:     "list",
		TotalUsage: (float64(quota) / consts.USD_QUOTA_UNIT) * 100,
	}, nil
}
