package session

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
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

	if userId == 0 || appId == 0 {
		return errors.ERR_INVALID_API_KEY
	}

	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.USER_ID_KEY, userId)
		r.SetCtxVar(consts.APP_ID_KEY, appId)
		r.SetCtxVar(consts.SECRET_KEY, secretKey)
		r.SetCtxVar(consts.MODEL_AGENT_HEADER, r.GetHeader(consts.MODEL_AGENT_HEADER))
	}

	return nil
}

// 保存应用和密钥是否限制额度
func (s *sSession) SaveIsLimitQuota(ctx context.Context, app, key bool) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.APP_IS_LIMIT_QUOTA_KEY, app)
		r.SetCtxVar(consts.KEY_IS_LIMIT_QUOTA_KEY, key)
	}
}

// 保存代理商ID到会话中
func (s *sSession) SaveRid(ctx context.Context, rid int) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.RID_KEY, rid)
	}
}

// 获取代理商ID
func (s *sSession) GetRid(ctx context.Context) int {

	rid := ctx.Value(consts.RID_KEY)
	if rid == nil {
		return 0
	}

	return rid.(int)
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

// 保存代理商信息到会话中
func (s *sSession) SaveReseller(ctx context.Context, reseller *model.Reseller) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_RESELLER, reseller)
	}
}

// 获取会话中的代理商信息
func (s *sSession) GetReseller(ctx context.Context) *model.Reseller {

	reseller := ctx.Value(consts.SESSION_RESELLER)
	if reseller == nil {
		return nil
	}

	return reseller.(*model.Reseller)
}

// 保存用户信息到会话中
func (s *sSession) SaveUser(ctx context.Context, user *model.User) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_USER, user)
	}
}

// 获取会话中的用户信息
func (s *sSession) GetUser(ctx context.Context) *model.User {

	user := ctx.Value(consts.SESSION_USER)
	if user == nil {
		return nil
	}

	return user.(*model.User)
}

// 保存应用信息到会话中
func (s *sSession) SaveApp(ctx context.Context, app *model.App) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_APP, app)
	}
}

// 获取会话中的应用信息
func (s *sSession) GetApp(ctx context.Context) *model.App {

	app := ctx.Value(consts.SESSION_APP)
	if app == nil {
		return nil
	}

	return app.(*model.App)
}

// 保存应用密钥信息到会话中
func (s *sSession) SaveAppKey(ctx context.Context, key *model.AppKey) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_APP_KEY, key)
	}
}

// 获取会话中的应用密钥信息
func (s *sSession) GetAppKey(ctx context.Context) *model.AppKey {

	key := ctx.Value(consts.SESSION_APP_KEY)
	if key == nil {
		return nil
	}

	return key.(*model.AppKey)
}

// 记录错误模型代理ID到会话中
func (s *sSession) RecordErrorModelAgent(ctx context.Context, id string) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_ERROR_MODEL_AGENTS, append(s.GetErrorModelAgents(ctx), id))
	}
}

// 获取会话中的错误模型代理Ids
func (s *sSession) GetErrorModelAgents(ctx context.Context) []string {

	modelAgents := ctx.Value(consts.SESSION_ERROR_MODEL_AGENTS)
	if modelAgents == nil {
		return []string{}
	}

	return modelAgents.([]string)
}

// 记录错误密钥ID到会话中
func (s *sSession) RecordErrorKey(ctx context.Context, id string) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_ERROR_KEYS, append(s.GetErrorModelAgents(ctx), id))
	}
}

// 获取会话中的错误密钥Ids
func (s *sSession) GetErrorKeys(ctx context.Context) []string {

	keys := ctx.Value(consts.SESSION_ERROR_KEYS)
	if keys == nil {
		return []string{}
	}

	return keys.([]string)
}

// 是否已选定模型代理
func (s *sSession) IsSelectedModelAgent(ctx context.Context) (string, bool) {

	modelAgentId := ctx.Value(consts.MODEL_AGENT_HEADER)

	return modelAgentId.(string), modelAgentId != ""
}

// 保存模型代理计费方式
func (s *sSession) SaveModelAgentBillingMethod(ctx context.Context, billingMethod int) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(consts.SESSION_MODEL_AGENT_BILLING_METHOD, billingMethod)
	}
}

// 获取模型代理计费方式
func (s *sSession) GetModelAgentBillingMethod(ctx context.Context) int {

	billingMethod := ctx.Value(consts.SESSION_MODEL_AGENT_BILLING_METHOD)
	if billingMethod == nil {
		return 0
	}

	return billingMethod.(int)
}
