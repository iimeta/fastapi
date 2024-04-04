package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/sashabaranov/go-openai"
)

// 记录使用额度
func (s *sCommon) RecordUsage(ctx context.Context, model *model.Model, usage openai.Usage) error {

	usageKey := s.GetUserUsageKey(ctx)

	promptTokens := model.PromptRatio * float64(usage.PromptTokens)
	completionTokens := model.CompletionRatio * float64(usage.CompletionTokens)
	totalTokens := promptTokens + completionTokens

	currentQuota, err := redis.HIncrBy(ctx, usageKey, consts.USER_TOTAL_TOKENS_FIELD, int64(-totalTokens))
	if err != nil {
		logger.Error(ctx, err)
	}

	if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
		if err = service.User().ChangeQuota(ctx, service.Session().GetUserId(ctx), int(-totalTokens), int(currentQuota)); err != nil {
			logger.Error(ctx, err)
		}
	}, nil); err != nil {
		logger.Error(ctx, err)
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		currentQuota, err = redis.HIncrBy(ctx, usageKey, s.GetAppTotalTokensField(ctx), int64(-totalTokens))
		if err != nil {
			logger.Error(ctx, err)
		}

		if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
			if err = service.App().ChangeQuota(ctx, service.Session().GetAppId(ctx), int(-totalTokens), int(currentQuota)); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		currentQuota, err = redis.HIncrBy(ctx, usageKey, s.GetKeyTotalTokensField(ctx), int64(-totalTokens))
		if err != nil {
			logger.Error(ctx, err)
		}

		if err = grpool.AddWithRecover(ctx, func(ctx context.Context) {
			if err = service.App().ChangeAppKeyQuota(ctx, service.Session().GetSecretKey(ctx), int(-totalTokens), int(currentQuota)); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}

	return nil
}

func (s *sCommon) GetUserTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), consts.USER_TOTAL_TOKENS_FIELD)
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
	return fmt.Sprintf(consts.APP_TOTAL_TOKENS_FIELD, service.Session().GetAppId(ctx))
}

func (s *sCommon) GetKeyTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.KEY_TOTAL_TOKENS_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}
