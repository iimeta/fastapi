package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"time"
)

// 记录使用额度
func (s *sCommon) RecordUsage(ctx context.Context, totalTokens int, key string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCommon RecordUsage time: %d", gtime.TimestampMilli()-now)
	}()

	if totalTokens == 0 {
		return nil
	}

	userId := service.Session().GetUserId(ctx)
	appId := service.Session().GetAppId(ctx)
	appKey := service.Session().GetSecretKey(ctx)

	logger.Infof(ctx, "sCommon RecordUsage userId: %d, appId: %d, appKey: %s, spendQuota: %d, key: %s", userId, appId, appKey, totalTokens, key)

	usageKey := s.GetUserUsageKey(ctx)

	currentQuota, err := redisSpendQuota(ctx, usageKey, consts.USER_QUOTA_FIELD, totalTokens)
	if err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if err = mongoSpendQuota(ctx, func() error {
		return service.User().SpendQuota(ctx, userId, totalTokens, currentQuota)
	}); err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		currentQuota, err = redisSpendQuota(ctx, usageKey, s.GetAppTotalTokensField(ctx), totalTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.App().SpendQuota(ctx, appId, totalTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

	} else {
		if err = mongoUsedQuota(ctx, func() error {
			return service.App().UsedQuota(ctx, appId, totalTokens)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		currentQuota, err = redisSpendQuota(ctx, usageKey, s.GetKeyTotalTokensField(ctx), totalTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.App().AppKeySpendQuota(ctx, appKey, totalTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

	} else {
		if err = mongoUsedQuota(ctx, func() error {
			return service.App().AppKeyUsedQuota(ctx, appKey, totalTokens)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if err = mongoUsedQuota(ctx, func() error {
		return service.Key().UsedQuota(ctx, key, totalTokens)
	}); err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	return nil
}

func redisSpendQuota(ctx context.Context, usageKey, field string, totalTokens int, retry ...int) (int, error) {

	currentQuota, err := redis.HIncrBy(ctx, usageKey, field, int64(-totalTokens))
	if err != nil {
		logger.Errorf(ctx, "redisSpendQuota usageKey: %s, field: %s, totalTokens: %d, err: %v", usageKey, field, totalTokens, err)

		if len(retry) == 10 {
			return -1, err
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "redisSpendQuota usageKey: %s, field: %s, totalTokens: %d, retry: %d", usageKey, field, totalTokens, len(retry))

		return redisSpendQuota(ctx, usageKey, field, totalTokens, retry...)
	}

	return int(currentQuota), nil
}

func mongoSpendQuota(ctx context.Context, f func() error, retry ...int) error {

	if err := f(); err != nil {
		logger.Errorf(ctx, "mongoSpendQuota err: %v", err)

		if len(retry) == 10 {
			return err
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "mongoSpendQuota retry: %d", len(retry))

		return mongoSpendQuota(ctx, f, retry...)
	}

	return nil
}

func mongoUsedQuota(ctx context.Context, f func() error, retry ...int) error {

	if err := f(); err != nil {
		logger.Errorf(ctx, "mongoUsedQuota err: %v", err)

		if len(retry) == 10 {
			return err
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "mongoUsedQuota retry: %d", len(retry))

		return mongoUsedQuota(ctx, f, retry...)
	}

	return nil
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
