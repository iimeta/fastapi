package auth

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/consts"
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

	uid, appid, err := service.Common().ParseSecretKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	user, err := service.User().GetUserByUid(ctx, uid)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}
	fmt.Println(gjson.MustEncodeString(user))

	app, err := service.App().GetAppByAppid(ctx, appid)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	fmt.Println(gjson.MustEncodeString(app))

	r := g.RequestFromCtx(ctx)

	r.SetCtxVar(consts.UID_KEY, uid)
	r.SetCtxVar(consts.APPID_KEY, appid)
	r.SetCtxVar(consts.SECRET_KEY, secretKey)

	return service.Common().VerifySecretKey(ctx, secretKey)
}

func (s *sAuth) GetUid(ctx context.Context) int {

	uid := ctx.Value(consts.UID_KEY)
	if uid == nil {
		logger.Error(ctx, "uid is nil")
		return 0
	}

	return uid.(int)
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
