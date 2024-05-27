package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
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

	if err := service.Common().VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey); err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return err
	}

	return nil
}
