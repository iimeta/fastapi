package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type sAuth struct{}

func init() {
	service.RegisterAuth(New())
}

func New() service.IAuth {
	return &sAuth{}
}

// 身份核验
func (s *sAuth) Authenticator(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAuth Authenticator time: %d", gtime.TimestampMilli()-now)
	}()

	if err := service.Session().Save(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey); err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return err
	}

	return nil
}

// 核验密钥
func (s *sAuth) VerifySecretKey(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAuth VerifySecretKey time: %d", gtime.TimestampMilli()-now)
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

	if key.IsLimitQuota && (key.Quota <= 0 || (key.QuotaExpiresAt != 0 && key.QuotaExpiresAt < gtime.TimestampMilli())) {
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

	if user.Quota <= 0 || (user.QuotaExpiresAt != 0 && user.QuotaExpiresAt < gtime.TimestampMilli()) {
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

	if app.IsLimitQuota && (app.Quota <= 0 || (app.QuotaExpiresAt != 0 && app.QuotaExpiresAt < gtime.TimestampMilli())) {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveUser(ctx, user)
	service.Session().SaveIsLimitQuota(ctx, app.IsLimitQuota, key.IsLimitQuota)

	return nil
}
