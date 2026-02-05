package auth

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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
		logger.Debugf(g.RequestFromCtx(ctx).GetCtx(), "sAuth Authenticator time: %d", gtime.TimestampMilli()-now)
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

	path := g.RequestFromCtx(ctx).URL.Path
	modelsPath := "/v1/models"

	key, err := service.AppKey().GetCache(ctx, secretKey)
	if err != nil || key == nil {
		if key, err = service.AppKey().GetBySecretKey(ctx, secretKey); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_API_KEY
		}

		if err = service.AppKey().SaveCache(ctx, key); err != nil {
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

	if err = common.CheckIp(ctx, key.IpWhitelist, key.IpBlacklist); err != nil {
		logger.Errorf(ctx, "sAuth Key CheckIp ClientIp: %s, RemoteIp: %s, error: %v", g.RequestFromCtx(ctx).GetClientIp(), g.RequestFromCtx(ctx).GetRemoteIp(), err)
		return err
	}

	if key.IsLimitQuota {
		if path != modelsPath && service.AppKey().GetCacheQuota(ctx, key.Key) <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}

		if key.QuotaExpiresAt != 0 && key.QuotaExpiresAt < gtime.TimestampMilli() {
			err = errors.ERR_KEY_QUOTA_EXPIRED
			logger.Error(ctx, err)
			return err
		}
	}

	user, err := service.User().GetCache(ctx, service.Session().GetUserId(ctx))
	if err != nil || user == nil {
		if user, err = service.User().GetByUserId(ctx, service.Session().GetUserId(ctx)); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_USER
		}

		if err = service.User().SaveCache(ctx, user); err != nil {
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

	if path != modelsPath && service.User().GetCacheQuota(ctx, user.UserId) <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	if user.QuotaExpiresAt != 0 && user.QuotaExpiresAt < gtime.TimestampMilli() {
		err = errors.ERR_ACCOUNT_QUOTA_EXPIRED
		logger.Error(ctx, err)
		return err
	}

	if user.Rid != 0 {

		reseller, err := service.Reseller().GetCache(ctx, user.Rid)
		if err != nil || reseller == nil {

			if reseller, err = service.Reseller().GetByUserId(ctx, user.Rid); err != nil {
				logger.Error(ctx, err)
				return errors.ERR_INVALID_RESELLER
			}

			if err = service.Reseller().SaveCache(ctx, reseller); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}

		if reseller == nil {
			err = errors.ERR_INVALID_RESELLER
			logger.Error(ctx, err)
			return err
		}

		if reseller.Status == 2 {
			err = errors.ERR_RESELLER_DISABLED
			logger.Error(ctx, err)
			return err
		}

		if service.Reseller().GetCacheQuota(ctx, reseller.UserId) <= 0 {
			err = errors.ERR_RESELLER_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}

		if reseller.QuotaExpiresAt != 0 && reseller.QuotaExpiresAt < gtime.TimestampMilli() {
			err = errors.ERR_RESELLER_QUOTA_EXPIRED
			logger.Error(ctx, err)
			return err
		}

		service.Session().SaveRid(ctx, reseller.UserId)
		service.Session().SaveReseller(ctx, reseller)
	}

	app, err := service.App().GetCache(ctx, key.AppId)
	if err != nil || app == nil {
		if app, err = service.App().GetByAppId(ctx, key.AppId); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_APP
		}

		if err = service.App().SaveCache(ctx, app); err != nil {
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

	if err = common.CheckIp(ctx, app.IpWhitelist, app.IpBlacklist); err != nil {
		logger.Errorf(ctx, "sAuth App CheckIp ClientIp: %s, RemoteIp: %s, error: %v", g.RequestFromCtx(ctx).GetClientIp(), g.RequestFromCtx(ctx).GetRemoteIp(), err)
		return err
	}

	if app.IsLimitQuota {
		if path != modelsPath && service.App().GetCacheQuota(ctx, app.AppId) <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}

		if app.QuotaExpiresAt != 0 && app.QuotaExpiresAt < gtime.TimestampMilli() {
			err = errors.ERR_APP_QUOTA_EXPIRED
			logger.Error(ctx, err)
			return err
		}
	}

	service.Session().SaveUser(ctx, user)
	service.Session().SaveIsLimitQuota(ctx, app.IsLimitQuota, key.IsLimitQuota)

	if key.QuotaExpiresRule == 2 {
		if err = service.AppKey().UpdateQuotaExpiresAt(ctx, key); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}
