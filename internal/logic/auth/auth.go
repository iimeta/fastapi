package auth

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/consts"
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

func (s *sAuth) GetUid(ctx context.Context) int {

	uid := ctx.Value(consts.UID_KEY)
	if uid == nil {
		logger.Error(ctx, "uid is nil")
		return 0
	}

	return uid.(int)
}

func (s *sAuth) VerifyToken(ctx context.Context, token string) (bool, error) {

	if token == "" {
		return false, errors.New("token is nil")
	}

	if !s.CheckUsage(ctx) {
		logger.Errorf(ctx, "token: %s usage exhausted", token)
		return false, nil
	}

	return service.Vip().CheckUserVipPermissions(ctx, token, g.RequestFromCtx(ctx).GetForm("model").String()), nil
}

func (s *sAuth) GetToken(ctx context.Context) string {

	sk := ctx.Value(consts.SECRET_KEY)
	if sk == nil {
		logger.Error(ctx, "sk is nil")
		return ""
	}

	return sk.(string)
}

func (s *sAuth) CheckUsage(ctx context.Context) bool {

	usedTokens, err := service.Common().GetUsedTokens(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return false
	}

	totalTokens, err := service.Common().GetTotalTokens(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return false
	}

	return usedTokens < totalTokens
}
