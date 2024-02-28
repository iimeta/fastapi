package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
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

func (s *sAuth) VerifySecretKey(ctx context.Context, secretKey string) error {

	err := service.Session().Save(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	err = s.CheckUser(g.RequestFromCtx(ctx).GetCtx(), service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx()))
	if err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return err
	}

	err = service.Common().VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey)
	if err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return err
	}

	return nil
}

func (s *sAuth) CheckUser(ctx context.Context, userId int) error {

	user, err := service.Common().GetCacheUser(ctx, userId)
	if err != nil || user == nil {
		user, err = service.User().GetUserByUserId(ctx, userId)
		if err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_USER
		}
		if err = service.Common().SaveCacheUser(ctx, user); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if user == nil || user.Status == 2 {
		return errors.ERR_INVALID_USER
	}

	return nil
}
