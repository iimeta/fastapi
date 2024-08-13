package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
)

// 记录使用额度
func (s *sCommon) RecordUsage(ctx context.Context, totalTokens int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCommon RecordUsage time: %d", gtime.TimestampMilli()-now)
	}()

	if totalTokens == 0 {
		return nil
	}

	usageKey := s.GetUserUsageKey(ctx)

	currentQuota, err := redis.HIncrBy(ctx, usageKey, consts.USER_QUOTA_FIELD, int64(-totalTokens))
	if err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if err = grpool.Add(ctx, func(ctx context.Context) {
		service.User().SpendQuota(ctx, service.Session().GetUserId(ctx), totalTokens, int(currentQuota))
	}); err != nil {
		logger.Error(ctx, err)
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		currentQuota, err = redis.HIncrBy(ctx, usageKey, s.GetAppTotalTokensField(ctx), int64(-totalTokens))
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = grpool.Add(ctx, func(ctx context.Context) {
			service.App().SpendQuota(ctx, service.Session().GetAppId(ctx), totalTokens, int(currentQuota))
		}); err != nil {
			logger.Error(ctx, err)
		}

	} else {
		if err = grpool.Add(ctx, func(ctx context.Context) {
			service.App().UsedQuota(ctx, service.Session().GetAppId(ctx), totalTokens)
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		currentQuota, err = redis.HIncrBy(ctx, usageKey, s.GetKeyTotalTokensField(ctx), int64(-totalTokens))
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = grpool.Add(ctx, func(ctx context.Context) {
			service.App().AppKeySpendQuota(ctx, service.Session().GetSecretKey(ctx), totalTokens, int(currentQuota))
		}); err != nil {
			logger.Error(ctx, err)
		}

	} else {
		if err = grpool.Add(ctx, func(ctx context.Context) {
			service.App().AppKeyUsedQuota(ctx, service.Session().GetSecretKey(ctx), totalTokens)
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	return err
}

func (s *sCommon) GetUserTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), consts.USER_QUOTA_FIELD)
}

func (s *sCommon) GetAppTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetAppTotalTokensField(ctx))
}

func (s *sCommon) GetKeyTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetKeyTotalTokensField(ctx))
}

func (s *sCommon) GetUserUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.API_USAGE_KEY, service.Session().GetUserId(ctx))
}

func (s *sCommon) GetAppTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_QUOTA_FIELD, service.Session().GetAppId(ctx))
}

func (s *sCommon) GetKeyTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.KEY_QUOTA_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}
