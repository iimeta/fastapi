package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
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

func (s *sAuth) VerifySecretKey(ctx context.Context, secretKey string) (bool, error) {

	err := service.Session().Save(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	pass, err := service.Common().VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey)
	if err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return false, err
	}

	return pass, nil
}
