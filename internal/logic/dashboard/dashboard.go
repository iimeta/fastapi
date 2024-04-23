package dashboard

import (
	"context"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"math"
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

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sDashboard Subscription time: %d", gtime.TimestampMilli()-now)
	}()

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
		SoftLimitUSD:       round(float64(quota)/consts.QUOTA_USD_UNIT, 4),
		HardLimitUSD:       round(float64(quota)/consts.QUOTA_USD_UNIT, 4),
		SystemHardLimitUSD: round(float64(quota)/consts.QUOTA_USD_UNIT, 4),
		AccessUntil:        0,
	}, nil
}

// Usage
func (s *sDashboard) Usage(ctx context.Context) (*model.DashboardUsageRes, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sDashboard Usage time: %d", gtime.TimestampMilli()-now)
	}()

	usedQuota := service.Session().GetUser(ctx).UsedQuota

	if service.Session().GetAppIsLimitQuota(ctx) {
		usedQuota = service.Session().GetApp(ctx).UsedQuota
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {
		usedQuota = service.Session().GetKey(ctx).UsedQuota
	}

	return &model.DashboardUsageRes{
		Object:     "list",
		TotalUsage: round(float64(usedQuota)/consts.QUOTA_USD_UNIT, 4),
	}, nil
}

func round(f float64, n int) float64 {
	n10 := math.Pow10(n)
	return math.Trunc((f+0.5/n10)*n10) / n10
}
