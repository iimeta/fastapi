package common

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
)

// 记录花费
func (s *sCommon) RecordSpend(ctx context.Context, totalTokens int, key string, group *model.Group) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCommon RecordSpend time: %d", gtime.TimestampMilli()-now)
	}()

	if totalTokens == 0 {
		return nil
	}

	rid := service.Session().GetRid(ctx)
	userId := service.Session().GetUserId(ctx)
	appId := service.Session().GetAppId(ctx)
	appKey := service.Session().GetSecretKey(ctx)

	logger.Infof(ctx, "sCommon RecordSpend rid: %d, userId: %d, appId: %d, appKey: %s, spendQuota: %d, key: %s", rid, userId, appId, appKey, totalTokens, key)

	usageKey := getUserUsageKey(ctx)

	currentQuota, err := redisSpendQuota(ctx, usageKey, consts.USER_QUOTA_FIELD, totalTokens)
	if err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if err = service.User().SaveCacheQuota(ctx, userId, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	if currentQuota <= config.Cfg.QuotaWarning.Threshold {
		if _, err = redis.Publish(ctx, consts.CHANGE_CHANNEL_USER, model.PubMessage{
			Action: consts.ACTION_CACHE,
			NewData: &model.UserQuota{
				UserId:       userId,
				CurrentQuota: currentQuota,
			},
		}); err != nil {
			logger.Error(ctx, err)
		}
	}

	if err = mongoSpendQuota(ctx, func() error {
		return service.User().SpendQuota(ctx, userId, totalTokens, currentQuota)
	}); err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if rid != 0 {

		currentQuota, err = redisSpendQuota(ctx, getResellerUsageKey(ctx), consts.RESELLER_QUOTA_FIELD, totalTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.Reseller().SpendQuota(ctx, rid, totalTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		currentQuota, err = redisSpendQuota(ctx, usageKey, getAppTotalTokensField(ctx), totalTokens)
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

		currentQuota, err = redisSpendQuota(ctx, usageKey, getAppKeyTotalTokensField(ctx), totalTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.AppKey().SpendQuota(ctx, appKey, totalTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

	} else {
		if err = mongoUsedQuota(ctx, func() error {
			return service.AppKey().UsedQuota(ctx, appKey, totalTokens)
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

	if group != nil {

		if group.IsLimitQuota {

			currentQuota, err = redisSpendQuota(ctx, consts.API_GROUP_USAGE_KEY, group.Id, totalTokens)
			if err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

			if err = mongoSpendQuota(ctx, func() error {
				return service.Group().SpendQuota(ctx, group.Id, totalTokens, currentQuota)
			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

		} else {
			if err = mongoUsedQuota(ctx, func() error {
				return service.Group().UsedQuota(ctx, group.Id, totalTokens)
			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}

		if group.IsEnableForward && group.ForwardConfig.ForwardRule == 4 && group.UsedQuota < group.ForwardConfig.UsedQuota {

			if group, err = service.Group().GetById(ctx, group.Id); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

			if err = service.Group().SaveCache(ctx, group); err != nil {
				logger.Error(ctx, err)
			}
		}
	}

	return nil
}

func redisSpendQuota(ctx context.Context, usageKey, field string, totalTokens int, retry ...int) (int, error) {

	currentQuota, err := redis.HIncrBy(ctx, usageKey, field, int64(-totalTokens))
	if err != nil {
		logger.Errorf(ctx, "redisSpendQuota usageKey: %s, field: %s, totalTokens: %d, error: %v", usageKey, field, totalTokens, err)

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
		logger.Errorf(ctx, "mongoSpendQuota error: %v", err)

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
		logger.Errorf(ctx, "mongoUsedQuota error: %v", err)

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

func getResellerUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.API_RESELLER_USAGE_KEY, service.Session().GetRid(ctx))
}

func getUserUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.API_USER_USAGE_KEY, service.Session().GetUserId(ctx))
}

func getAppTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_QUOTA_FIELD, service.Session().GetAppId(ctx))
}

func getAppKeyTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_KEY_QUOTA_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}
