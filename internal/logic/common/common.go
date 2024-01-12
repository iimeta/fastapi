package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"strings"
)

type sCommon struct{}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{}
}

func (s *sCommon) VerifySecretKey(ctx context.Context, secretKey string) (bool, error) {

	key, err := service.Key().GetKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	if key == nil || key.Key != secretKey {
		err = errors.Newf("invalid secretKey: %s", secretKey)
		logger.Error(ctx, err)
		return false, err
	}

	return true, nil
}

func (s *sCommon) GetUidUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.API_USAGE_KEY, service.Auth().GetUserId(ctx))
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

func (s *sCommon) ParseSecretKey(ctx context.Context, secretKey string) (int, int, error) {

	secretKey = strings.TrimPrefix(secretKey, "sk-FastAPI")

	userId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[:len(secretKey)/2])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	appId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[len(secretKey)/2:])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	return gconv.Int(userId), gconv.Int(appId), nil
}
