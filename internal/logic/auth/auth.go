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

func (s *sAuth) VerifySecretKey(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAuth VerifySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	if err := service.Session().Save(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
		return err
	}

	//if err := s.CheckUser(g.RequestFromCtx(ctx).GetCtx(), service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx())); err != nil {
	//	logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
	//	return err
	//}

	if err := service.Common().VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey); err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return err
	}

	return nil
}

func (s *sAuth) CheckUser(ctx context.Context, userId int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAuth CheckUser time: %d", gtime.TimestampMilli()-now)
	}()

	if userId == 0 {
		return errors.ERR_INVALID_API_KEY
	}

	user, err := service.User().GetCacheUser(ctx, userId)
	if err != nil || user == nil {

		if user, err = service.User().GetUser(ctx, userId); err != nil {
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

	service.Session().SaveUser(ctx, user)

	return nil
}
