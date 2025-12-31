package common

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/redis"
)

// 记录花费
func RecordSpend(ctx context.Context, spend common.Spend, mak *MAK) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "RecordSpend time: %d", gtime.TimestampMilli()-now)
	}()

	if spend.TotalSpendTokens == 0 {
		return nil
	}

	rid := service.Session().GetRid(ctx)
	userId := service.Session().GetUserId(ctx)
	appId := service.Session().GetAppId(ctx)
	appKey := service.Session().GetSecretKey(ctx)

	logger.Infof(ctx, "RecordSpend rid: %d, userId: %d, appId: %d, appKey: %s, totalSpendTokens: %d, key: %s", rid, userId, appId, appKey, spend.TotalSpendTokens, mak.Key.Key)

	usageKey := getUserUsageKey(ctx)

	currentQuota, err := redisSpendQuota(ctx, usageKey, consts.USER_QUOTA_FIELD, spend.TotalSpendTokens)
	if err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if err = service.User().SaveCacheQuota(ctx, userId, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	if currentQuota <= config.Cfg.Quota.Threshold {
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
		return service.User().SpendQuota(ctx, userId, spend.TotalSpendTokens, currentQuota)
	}); err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if rid != 0 {

		currentQuota, err = redisSpendQuota(ctx, getResellerUsageKey(ctx), consts.RESELLER_QUOTA_FIELD, spend.TotalSpendTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.Reseller().SpendQuota(ctx, rid, spend.TotalSpendTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		currentQuota, err = redisSpendQuota(ctx, usageKey, getAppTotalTokensField(ctx), spend.TotalSpendTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.App().SpendQuota(ctx, appId, spend.TotalSpendTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

	} else {
		if err = mongoUsedQuota(ctx, func() error {
			return service.App().UsedQuota(ctx, appId, spend.TotalSpendTokens)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		currentQuota, err = redisSpendQuota(ctx, usageKey, getAppKeyTotalTokensField(ctx), spend.TotalSpendTokens)
		if err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

		if err = mongoSpendQuota(ctx, func() error {
			return service.AppKey().SpendQuota(ctx, appKey, spend.TotalSpendTokens, currentQuota)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}

	} else {
		if err = mongoUsedQuota(ctx, func() error {
			return service.AppKey().UsedQuota(ctx, appKey, spend.TotalSpendTokens)
		}); err != nil {
			logger.Error(ctx, err)
			panic(err)
		}
	}

	if err = mongoUsedQuota(ctx, func() error {
		return service.Key().UsedQuota(ctx, mak.Key.Key, spend.TotalSpendTokens)
	}); err != nil {
		logger.Error(ctx, err)
		panic(err)
	}

	if mak.Group != nil {

		if mak.Group.IsLimitQuota {

			currentQuota, err = redisSpendQuota(ctx, consts.API_GROUP_USAGE_KEY, mak.Group.Id, spend.TotalSpendTokens)
			if err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

			if err = mongoSpendQuota(ctx, func() error {
				return service.Group().SpendQuota(ctx, mak.Group.Id, spend.TotalSpendTokens, currentQuota)
			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

		} else {
			if err = mongoUsedQuota(ctx, func() error {
				return service.Group().UsedQuota(ctx, mak.Group.Id, spend.TotalSpendTokens)
			}); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}
		}

		if mak.Group.IsEnableForward && mak.Group.ForwardConfig.ForwardRule == 4 && mak.Group.UsedQuota < mak.Group.ForwardConfig.UsedQuota {

			if mak.Group, err = service.Group().GetById(ctx, mak.Group.Id); err != nil {
				logger.Error(ctx, err)
				panic(err)
			}

			if err = service.Group().SaveCache(ctx, mak.Group); err != nil {
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
