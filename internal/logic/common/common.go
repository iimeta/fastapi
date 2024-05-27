package common

import (
	"context"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"strings"
)

type sCommon struct{}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{}
}

// 核验密钥
func (s *sCommon) VerifySecretKey(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sCommon VerifySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := service.App().GetCacheAppKey(ctx, secretKey)
	if err != nil || key == nil {

		if key, err = service.Key().GetKey(ctx, secretKey); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_API_KEY
		}

		if err = service.App().SaveCacheAppKey(ctx, key); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if key == nil || key.Key != secretKey {
		err = errors.ERR_INVALID_API_KEY
		logger.Error(ctx, err)
		return err
	}

	if key.Status == 2 {
		err = errors.ERR_API_KEY_DISABLED
		logger.Error(ctx, err)
		return err
	}

	if key.IsLimitQuota && key.Quota <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	user, err := service.User().GetCacheUser(ctx, service.Session().GetUserId(ctx))
	if err != nil || user == nil {

		if user, err = service.User().GetUser(ctx, service.Session().GetUserId(ctx)); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_USER
		}

		if err = service.User().SaveCacheUser(ctx, user); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if user == nil {
		err = errors.ERR_INVALID_USER
		logger.Error(ctx, err)
		return err
	}

	if user.Status == 2 {
		err = errors.ERR_USER_DISABLED
		logger.Error(ctx, err)
		return err
	}

	if user.Quota <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	app, err := service.App().GetCacheApp(ctx, key.AppId)
	if err != nil || app == nil {

		if app, err = service.App().GetApp(ctx, key.AppId); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_APP
		}

		if err = service.App().SaveCacheApp(ctx, app); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if app == nil {
		err = errors.ERR_INVALID_APP
		logger.Error(ctx, err)
		return err
	}

	if app.Status == 2 {
		err = errors.ERR_APP_DISABLED
		logger.Error(ctx, err)
		return err
	}

	if app.IsLimitQuota && app.Quota <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveUser(ctx, user)
	service.Session().SaveIsLimitQuota(ctx, app.IsLimitQuota, key.IsLimitQuota)

	return nil
}

// 解析密钥
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

// 记录错误次数和禁用
func (s *sCommon) RecordError(ctx context.Context, model *model.Model, key *model.Key, modelAgent *model.ModelAgent) {

	if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

		if model.IsEnableModelAgent {
			service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
			service.ModelAgent().RecordErrorModelAgent(ctx, model, modelAgent)
		} else {
			service.Key().RecordErrorModelKey(ctx, model, key)
		}

	}, nil); err != nil {
		logger.Error(ctx, err)
	}
}
