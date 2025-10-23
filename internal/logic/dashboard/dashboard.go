package dashboard

import (
	"context"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
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

	quota := 0

	if service.Session().GetAppIsLimitQuota(ctx) {

		app, err := service.App().GetByAppId(ctx, service.Session().GetAppId(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Subscription GetByAppId error: %v", err)
			return nil, err
		}

		quota = app.Quota
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		key, err := service.AppKey().GetBySecretKey(ctx, service.Session().GetSecretKey(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Subscription GetBySecretKey error: %v", err)
			return nil, err
		}

		quota = key.Quota
	}

	if quota == 0 {

		user, err := service.User().GetByUserId(ctx, service.Session().GetUserId(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Subscription GetByUserId error: %v", err)
			return nil, err
		}

		quota = user.Quota
	}

	return &model.DashboardSubscriptionRes{
		Object:             "billing_subscription",
		HasPaymentMethod:   true,
		SoftLimitUSD:       common.ConvQuota(quota, 4),
		HardLimitUSD:       common.ConvQuota(quota, 4),
		SystemHardLimitUSD: common.ConvQuota(quota, 4),
		AccessUntil:        0,
	}, nil
}

// Usage
func (s *sDashboard) Usage(ctx context.Context) (*model.DashboardUsageRes, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sDashboard Usage time: %d", gtime.TimestampMilli()-now)
	}()

	usedQuota := 0

	if service.Session().GetAppIsLimitQuota(ctx) {

		app, err := service.App().GetByAppId(ctx, service.Session().GetAppId(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Usage GetByAppId error: %v", err)
			return nil, err
		}

		usedQuota = app.UsedQuota
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		key, err := service.AppKey().GetBySecretKey(ctx, service.Session().GetSecretKey(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Usage GetBySecretKey error: %v", err)
			return nil, err
		}

		usedQuota = key.UsedQuota
	}

	if usedQuota == 0 {

		user, err := service.User().GetByUserId(ctx, service.Session().GetUserId(ctx))
		if err != nil {
			logger.Errorf(ctx, "sDashboard Usage GetByUserId error: %v", err)
			return nil, err
		}

		usedQuota = user.UsedQuota
	}

	return &model.DashboardUsageRes{
		Object:     "list",
		TotalUsage: common.ConvQuota(usedQuota, 4),
	}, nil
}
