package session

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

type sSession struct{}

func init() {
	service.RegisterSession(New())
}

func New() service.ISession {
	return &sSession{}
}

// 保存会话
func (s *sSession) Save(ctx context.Context, secretKey string) error {

	userId, appId, err := service.Common().ParseSecretKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	r := g.RequestFromCtx(ctx)

	r.SetCtxVar(consts.USER_ID_KEY, userId)
	r.SetCtxVar(consts.APP_ID_KEY, appId)
	r.SetCtxVar(consts.SECRET_KEY, secretKey)

	return nil
}

// 保存应用和密钥是否限制额度
func (s *sSession) SaveIsLimitQuota(ctx context.Context, app, key bool) error {

	r := g.RequestFromCtx(ctx)

	r.SetCtxVar(consts.APP_IS_LIMIT_QUOTA_KEY, app)
	r.SetCtxVar(consts.KEY_IS_LIMIT_QUOTA_KEY, key)

	return nil
}

// 获取用户ID
func (s *sSession) GetUserId(ctx context.Context) int {

	userId := ctx.Value(consts.USER_ID_KEY)
	if userId == nil {
		logger.Error(ctx, "user_id is nil")
		return 0
	}

	return userId.(int)
}

// 获取应用ID
func (s *sSession) GetAppId(ctx context.Context) int {

	appId := ctx.Value(consts.APP_ID_KEY)
	if appId == nil {
		logger.Error(ctx, "app_id is nil")
		return 0
	}

	return appId.(int)
}

// 获取密钥
func (s *sSession) GetSecretKey(ctx context.Context) string {

	secretKey := ctx.Value(consts.SECRET_KEY)
	if secretKey == nil {
		logger.Error(ctx, "secret_key is nil")
		return ""
	}

	return secretKey.(string)
}

// 获取应用是否限制额度
func (s *sSession) GetAppIsLimitQuota(ctx context.Context) bool {

	isLimitQuota := ctx.Value(consts.APP_IS_LIMIT_QUOTA_KEY)
	if isLimitQuota == nil {
		logger.Error(ctx, "app isLimitQuota is nil")
		return true
	}

	return isLimitQuota.(bool)
}

// 获取密钥是否限制额度
func (s *sSession) GetKeyIsLimitQuota(ctx context.Context) bool {

	isLimitQuota := ctx.Value(consts.KEY_IS_LIMIT_QUOTA_KEY)
	if isLimitQuota == nil {
		logger.Error(ctx, "key isLimitQuota is nil")
		return true
	}

	return isLimitQuota.(bool)
}

// 保存用户信息到会话中
func (s *sSession) SaveUser(ctx context.Context, user *model.User) {
	g.RequestFromCtx(ctx).SetCtxVar(consts.SESSION_USER, user)
}

// 获取会话中的用户信息
func (s *sSession) GetUser(ctx context.Context) *model.User {

	user := ctx.Value(consts.SESSION_USER)
	if user == nil {
		logger.Debug(ctx, "user is nil")
		return nil
	}

	return user.(*model.User)
}

// 保存应用信息到会话中
func (s *sSession) SaveApp(ctx context.Context, app *model.App) {
	g.RequestFromCtx(ctx).SetCtxVar(consts.SESSION_APP, app)
}

// 获取会话中的应用信息
func (s *sSession) GetApp(ctx context.Context) *model.App {

	app := ctx.Value(consts.SESSION_APP)
	if app == nil {
		logger.Debug(ctx, "app is nil")
		return nil
	}

	return app.(*model.App)
}

// 保存密钥信息到会话中
func (s *sSession) SaveKey(ctx context.Context, key *model.Key) {
	g.RequestFromCtx(ctx).SetCtxVar(consts.SESSION_KEY, key)
}

// 获取会话中的密钥信息
func (s *sSession) GetKey(ctx context.Context) *model.Key {

	key := ctx.Value(consts.SESSION_KEY)
	if key == nil {
		logger.Debug(ctx, "key is nil")
		return nil
	}

	return key.(*model.Key)
}
