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

	userId, appId, err := service.Common().ParseSecretKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	user, err := service.User().GetUserByUid(ctx, userId)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}
	fmt.Println(gjson.MustEncodeString(user))

	app, err := service.App().GetApp(ctx, appId)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	fmt.Println(gjson.MustEncodeString(app))

	r := g.RequestFromCtx(ctx)

	r.SetCtxVar(consts.USER_ID_KEY, userId)
	r.SetCtxVar(consts.APP_ID_KEY, appId)
	r.SetCtxVar(consts.SECRET_KEY, secretKey)

	pass, err := service.Common().VerifySecretKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return false, err
	}

	if pass {
		err = service.Session().SaveKey(ctx, secretKey)
		if err != nil {
			logger.Error(ctx, err)
			return false, err
		}
	}

	return pass, nil
}

func (s *sAuth) GetUserId(ctx context.Context) int {

	userId := ctx.Value(consts.USER_ID_KEY)
	if userId == nil {
		logger.Error(ctx, "user_id is nil")
		return 0
	}

	return userId.(int)
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
