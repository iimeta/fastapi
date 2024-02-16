package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/mongo"
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

	_, err = service.User().GetUserByUid(ctx, service.Session().GetUserId(g.RequestFromCtx(ctx).GetCtx()))
	if err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, errors.ERR_INVALID_USER
		}
		return false, err
	}

	pass, err := service.Common().VerifySecretKey(g.RequestFromCtx(ctx).GetCtx(), secretKey)
	if err != nil {
		logger.Error(g.RequestFromCtx(ctx).GetCtx(), err)
		return false, err
	}

	return pass, nil
}
