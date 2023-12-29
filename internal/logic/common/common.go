package common

import (
	"context"
	"fmt"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"time"
)

type sCommon struct{}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{}
}

func (s *sCommon) VerifyToken(ctx context.Context, secretKey string) bool {

	user, err := service.User().GetUserById(ctx, service.Auth().GetUid(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return false
	}

	if user.SecretKey != secretKey {
		logger.Errorf(ctx, "invalid user secretKey: %s", secretKey)
		return false
	}

	return true
}

func (s *sCommon) GetUidUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.UID_USAGE_KEY, service.Auth().GetUid(ctx), time.Now().Format("20060102"))
}

func (s *sCommon) RecordUsage(ctx context.Context, totalTokens int) error {

	if _, err := redis.HIncrBy(ctx, s.GetUidUsageKey(ctx), consts.USAGE_COUNT_FIELD, 1); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if _, err := redis.HIncrBy(ctx, s.GetUidUsageKey(ctx), consts.USED_TOKENS_FIELD, int64(totalTokens)); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

func (s *sCommon) GetUsageCount(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUidUsageKey(ctx), consts.USAGE_COUNT_FIELD)
}

func (s *sCommon) GetUsedTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUidUsageKey(ctx), consts.USED_TOKENS_FIELD)
}

func (s *sCommon) GetTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUidUsageKey(ctx), consts.TOTAL_TOKENS_FIELD)
}
