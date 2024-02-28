package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"github.com/sashabaranov/go-openai"
	"go.mongodb.org/mongo-driver/mongo"
	"strings"
)

type sCommon struct{}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{}
}

func (s *sCommon) VerifySecretKey(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "VerifySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	getKeyTime := gtime.TimestampMilli()
	key, err := service.Key().GetKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.ERR_INVALID_API_KEY
		}
		return err
	}
	logger.Debugf(ctx, "GetKey time: %d", gtime.TimestampMilli()-getKeyTime)

	if key == nil || key.Key != secretKey {
		err = errors.ERR_INVALID_API_KEY
		logger.Error(ctx, err)
		return err
	}

	getUserTotalTokensTime := gtime.TimestampMilli()
	userTotalTokens, err := s.GetUserTotalTokens(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Debugf(ctx, "GetUserTotalTokens time: %d", gtime.TimestampMilli()-getUserTotalTokensTime)

	if userTotalTokens <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	getAppTime := gtime.TimestampMilli()
	app, err := service.App().GetApp(ctx, key.AppId)
	if err != nil {
		logger.Error(ctx, err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return errors.ERR_INVALID_APP
		}
		return err
	}
	logger.Debugf(ctx, "GetApp time: %d", gtime.TimestampMilli()-getAppTime)

	if key.IsLimitQuota {

		getKeyTotalTokensTime := gtime.TimestampMilli()
		keyTotalTokens, err := s.GetKeyTotalTokens(ctx)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
		logger.Debugf(ctx, "GetKeyTotalTokens time: %d", gtime.TimestampMilli()-getKeyTotalTokensTime)

		if keyTotalTokens <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}
	}

	if app.IsLimitQuota {

		getAppTotalTokensTime := gtime.TimestampMilli()
		appTotalTokens, err := s.GetAppTotalTokens(ctx)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
		logger.Debugf(ctx, "GetAppTotalTokens time: %d", gtime.TimestampMilli()-getAppTotalTokensTime)

		if appTotalTokens <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}
	}

	err = service.Session().SaveIsLimitQuota(ctx, app.IsLimitQuota, key.IsLimitQuota)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

func (s *sCommon) RecordUsage(ctx context.Context, model *model.Model, usage openai.Usage) error {

	usageKey := s.GetUserUsageKey(ctx)

	promptTokens := model.PromptRatio * float64(usage.PromptTokens)
	completionTokens := model.CompletionRatio * float64(usage.CompletionTokens)
	totalTokens := promptTokens + completionTokens

	if _, err := redis.HIncrBy(ctx, usageKey, consts.USER_USAGE_COUNT_FIELD, 1); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HIncrBy(ctx, usageKey, consts.USER_USED_TOKENS_FIELD, int64(totalTokens)); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HIncrBy(ctx, usageKey, consts.USER_TOTAL_TOKENS_FIELD, int64(-totalTokens)); err != nil {
		logger.Error(ctx, err)
	}

	if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
		if err := service.User().ChangeQuota(ctx, service.Session().GetUserId(ctx), int(-totalTokens)); err != nil {
			logger.Error(ctx, err)
		}
	}, nil); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HIncrBy(ctx, usageKey, s.GetAppUsageCountField(ctx), 1); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HIncrBy(ctx, usageKey, s.GetAppUsedTokensField(ctx), int64(totalTokens)); err != nil {
		logger.Error(ctx, err)
	}

	if service.Session().GetAppIsLimitQuota(ctx) {

		if _, err := redis.HIncrBy(ctx, usageKey, s.GetAppTotalTokensField(ctx), int64(-totalTokens)); err != nil {
			logger.Error(ctx, err)
		}

		if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
			if err := service.App().ChangeQuota(ctx, service.Session().GetAppId(ctx), int(-totalTokens)); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}

	if _, err := redis.HIncrBy(ctx, usageKey, s.GetKeyUsageCountField(ctx), 1); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HIncrBy(ctx, usageKey, s.GetKeyUsedTokensField(ctx), int64(totalTokens)); err != nil {
		logger.Error(ctx, err)
	}

	if service.Session().GetKeyIsLimitQuota(ctx) {

		if _, err := redis.HIncrBy(ctx, usageKey, s.GetKeyTotalTokensField(ctx), int64(-totalTokens)); err != nil {
			logger.Error(ctx, err)
		}

		if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
			if err := service.Key().ChangeQuota(ctx, service.Session().GetSecretKey(ctx), int(-totalTokens)); err != nil {
				logger.Error(ctx, err)
			}
		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}

	return nil
}

func (s *sCommon) GetUserUsageKey(ctx context.Context) string {
	return fmt.Sprintf(consts.API_USAGE_KEY, service.Session().GetUserId(ctx))
}

func (s *sCommon) GetAppUsageCountField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_USAGE_COUNT_FIELD, service.Session().GetAppId(ctx))
}

func (s *sCommon) GetAppUsedTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_USED_TOKENS_FIELD, service.Session().GetAppId(ctx))
}

func (s *sCommon) GetAppTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.APP_TOTAL_TOKENS_FIELD, service.Session().GetAppId(ctx))
}

func (s *sCommon) GetKeyUsageCountField(ctx context.Context) string {
	return fmt.Sprintf(consts.KEY_USAGE_COUNT_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}

func (s *sCommon) GetKeyUsedTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.KEY_USED_TOKENS_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}

func (s *sCommon) GetKeyTotalTokensField(ctx context.Context) string {
	return fmt.Sprintf(consts.KEY_TOTAL_TOKENS_FIELD, service.Session().GetAppId(ctx), service.Session().GetSecretKey(ctx))
}

func (s *sCommon) GetUserUsageCount(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), consts.USER_USAGE_COUNT_FIELD)
}

func (s *sCommon) GetUserUsedTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), consts.USER_USED_TOKENS_FIELD)
}

func (s *sCommon) GetUserTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), consts.USER_TOTAL_TOKENS_FIELD)
}

func (s *sCommon) GetAppUsageCount(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetAppUsageCountField(ctx))
}

func (s *sCommon) GetAppUsedTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetAppUsedTokensField(ctx))
}

func (s *sCommon) GetAppTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetAppTotalTokensField(ctx))
}

func (s *sCommon) GetKeyUsageCount(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetKeyUsageCountField(ctx))
}

func (s *sCommon) GetKeyUsedTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetKeyUsedTokensField(ctx))
}

func (s *sCommon) GetKeyTotalTokens(ctx context.Context) (int, error) {
	return redis.HGetInt(ctx, s.GetUserUsageKey(ctx), s.GetKeyTotalTokensField(ctx))
}

// 分析密钥
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

// 保存用户信息到缓存
func (s *sCommon) SaveCacheUser(ctx context.Context, user *model.User) error {

	if user == nil {
		return errors.New("user is nil")
	}

	_, err := redis.Set(ctx, fmt.Sprintf(consts.API_USER_KEY, user.UserId), user)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的用户信息
func (s *sCommon) GetCacheUser(ctx context.Context, userId int) (*model.User, error) {

	reply, err := redis.Get(ctx, fmt.Sprintf(consts.API_USER_KEY, userId))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || reply.IsNil() {
		return nil, errors.New("user is nil")
	}

	user := new(model.User)

	err = reply.Struct(&user)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return user, nil
}
