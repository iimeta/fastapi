package common

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"strings"
)

type sCommon struct {
	userCacheMap   *cache.Cache
	appCacheMap    *cache.Cache
	appKeyCacheMap *cache.Cache
}

func init() {
	service.RegisterCommon(New())
}

func New() service.ICommon {
	return &sCommon{
		userCacheMap:   cache.New(),
		appCacheMap:    cache.New(),
		appKeyCacheMap: cache.New(),
	}
}

// 核验密钥
func (s *sCommon) VerifySecretKey(ctx context.Context, secretKey string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "VerifySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := s.GetCacheAppKey(ctx, secretKey)
	if err != nil || key == nil {

		if key, err = service.Key().GetKey(ctx, secretKey); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_API_KEY
		}

		if err = s.SaveCacheAppKey(ctx, key); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if key == nil || key.Key != secretKey {
		err = errors.ERR_INVALID_API_KEY
		logger.Error(ctx, err)
		return err
	}

	if key.Status == 2 {
		err = errors.ERR_API_KEY_DISABLED
		logger.Error(ctx, err)
		return err
	}

	getUserTotalTokensTime := gtime.TimestampMilli()
	userTotalTokens, err := s.GetUserTotalTokens(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Debugf(ctx, "GetUserTotalTokens time: %d", gtime.TimestampMilli()-getUserTotalTokensTime)

	if userTotalTokens <= 0 {
		err = errors.ERR_INSUFFICIENT_QUOTA
		logger.Error(ctx, err)
		return err
	}

	app, err := s.GetCacheApp(ctx, key.AppId)
	if err != nil || app == nil {

		if app, err = service.App().GetApp(ctx, key.AppId); err != nil {
			logger.Error(ctx, err)
			return errors.ERR_INVALID_APP
		}

		if err = s.SaveCacheApp(ctx, app); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if app.Status == 2 {
		err = errors.ERR_APP_DISABLED
		logger.Error(ctx, err)
		return err
	}

	if key.IsLimitQuota {

		getKeyTotalTokensTime := gtime.TimestampMilli()
		keyTotalTokens, err := s.GetKeyTotalTokens(ctx)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
		logger.Debugf(ctx, "GetKeyTotalTokens time: %d", gtime.TimestampMilli()-getKeyTotalTokensTime)

		if keyTotalTokens <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}
	}

	if app.IsLimitQuota {

		getAppTotalTokensTime := gtime.TimestampMilli()
		appTotalTokens, err := s.GetAppTotalTokens(ctx)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
		logger.Debugf(ctx, "GetAppTotalTokens time: %d", gtime.TimestampMilli()-getAppTotalTokensTime)

		if appTotalTokens <= 0 {
			err = errors.ERR_INSUFFICIENT_QUOTA
			logger.Error(ctx, err)
			return err
		}
	}

	if err = service.Session().SaveIsLimitQuota(ctx, app.IsLimitQuota, key.IsLimitQuota); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 解析密钥
func (s *sCommon) ParseSecretKey(ctx context.Context, secretKey string) (int, int, error) {

	secretKey = strings.TrimPrefix(secretKey, "sk-FastAPI")

	userId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[:len(secretKey)/2])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	appId, err := gregex.ReplaceString("[a-zA-Z-]*", "", secretKey[len(secretKey)/2:])
	if err != nil {
		logger.Error(ctx, err)
		return 0, 0, err
	}

	return gconv.Int(userId), gconv.Int(appId), nil
}

// 保存用户信息到缓存
func (s *sCommon) SaveCacheUser(ctx context.Context, user *model.User) error {

	if user == nil {
		return errors.New("user is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_USER_KEY, user.UserId), user); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveUser(ctx, user)

	if err := s.userCacheMap.Set(ctx, user.UserId, user, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的用户信息
func (s *sCommon) GetCacheUser(ctx context.Context, userId int) (*model.User, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetCacheUser time: %d", gtime.TimestampMilli()-now)
	}()

	if user := service.Session().GetUser(ctx); user != nil {
		return user, nil
	}

	if userCacheValue := s.userCacheMap.GetVal(ctx, userId); userCacheValue != nil {
		return userCacheValue.(*model.User), nil
	}

	reply, err := redis.Get(ctx, fmt.Sprintf(consts.API_USER_KEY, userId))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || reply.IsNil() {
		return nil, errors.New("user is nil")
	}

	user := new(model.User)
	if err = reply.Struct(&user); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	service.Session().SaveUser(ctx, user)

	if err = s.userCacheMap.Set(ctx, user.UserId, user, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return user, nil
}

// 更新缓存中的用户信息
func (s *sCommon) UpdateCacheUser(ctx context.Context, user *entity.User) {
	if err := s.SaveCacheUser(ctx, &model.User{
		Id:     user.Id,
		UserId: user.UserId,
		Name:   user.Name,
		Avatar: user.Avatar,
		Gender: user.Gender,
		Phone:  user.Phone,
		Email:  user.Email,
		Quota:  user.Quota,
		Status: user.Status,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的用户信息
func (s *sCommon) RemoveCacheUser(ctx context.Context, userId int) {

	if _, err := s.userCacheMap.Remove(ctx, userId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_USER_KEY, userId)); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存应用信息到缓存
func (s *sCommon) SaveCacheApp(ctx context.Context, app *model.App) error {

	if app == nil {
		return errors.New("app is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_APP_KEY, app.AppId), app); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveApp(ctx, app)

	if err := s.appCacheMap.Set(ctx, app.AppId, app, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用信息
func (s *sCommon) GetCacheApp(ctx context.Context, appId int) (*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if app := service.Session().GetApp(ctx); app != nil {
		return app, nil
	}

	if appCacheValue := s.appCacheMap.GetVal(ctx, appId); appCacheValue != nil {
		return appCacheValue.(*model.App), nil
	}

	reply, err := redis.Get(ctx, fmt.Sprintf(consts.API_APP_KEY, appId))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || reply.IsNil() {
		return nil, errors.New("app is nil")
	}

	app := new(model.App)
	if err = reply.Struct(&app); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	service.Session().SaveApp(ctx, app)

	if err = s.appCacheMap.Set(ctx, app.AppId, app, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return app, nil
}

// 更新缓存中的应用信息
func (s *sCommon) UpdateCacheApp(ctx context.Context, app *entity.App) {
	if err := s.SaveCacheApp(ctx, &model.App{
		Id:           app.Id,
		AppId:        app.AppId,
		Name:         app.Name,
		Type:         app.Type,
		Models:       app.Models,
		IsLimitQuota: app.IsLimitQuota,
		Quota:        app.Quota,
		IpWhitelist:  app.IpWhitelist,
		IpBlacklist:  app.IpBlacklist,
		Status:       app.Status,
		UserId:       app.UserId,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的应用信息
func (s *sCommon) RemoveCacheApp(ctx context.Context, appId int) {

	if _, err := s.appCacheMap.Remove(ctx, appId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_APP_KEY, appId)); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存应用密钥信息到缓存
func (s *sCommon) SaveCacheAppKey(ctx context.Context, key *model.Key) error {

	if key == nil {
		return errors.New("key is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_APP_KEY_KEY, key.Key), key); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveKey(ctx, key)

	if err := s.appKeyCacheMap.Set(ctx, key.Key, key, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用密钥信息
func (s *sCommon) GetCacheAppKey(ctx context.Context, secretKey string) (*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if key := service.Session().GetKey(ctx); key != nil {
		return key, nil
	}

	if keyCacheValue := s.appKeyCacheMap.GetVal(ctx, secretKey); keyCacheValue != nil {
		return keyCacheValue.(*model.Key), nil
	}

	reply, err := redis.Get(ctx, fmt.Sprintf(consts.API_APP_KEY_KEY, secretKey))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || reply.IsNil() {
		return nil, errors.New("key is nil")
	}

	key := new(model.Key)
	if err = reply.Struct(&key); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	service.Session().SaveKey(ctx, key)

	if err = s.appKeyCacheMap.Set(ctx, key.Key, key, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return key, nil
}

// 更新缓存中的应用密钥信息
func (s *sCommon) UpdateCacheAppKey(ctx context.Context, key *entity.Key) {
	if key.Type == 1 {
		if err := s.SaveCacheAppKey(ctx, &model.Key{
			Id:           key.Id,
			UserId:       key.UserId,
			AppId:        key.AppId,
			Corp:         key.Corp,
			Key:          key.Key,
			Type:         key.Type,
			Models:       key.Models,
			ModelAgents:  key.ModelAgents,
			IsLimitQuota: key.IsLimitQuota,
			Quota:        key.Quota,
			RPM:          key.RPM,
			RPD:          key.RPD,
			IpWhitelist:  key.IpWhitelist,
			IpBlacklist:  key.IpBlacklist,
			Status:       key.Status,
		}); err != nil {
			logger.Error(ctx, err)
		}
	}
}

// 移除缓存中的应用密钥信息
func (s *sCommon) RemoveCacheAppKey(ctx context.Context, secretKey string) {

	if _, err := s.appKeyCacheMap.Remove(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_APP_KEY_KEY, secretKey)); err != nil {
		logger.Error(ctx, err)
	}
}

// 记录错误次数和禁用
func (s *sCommon) RecordError(ctx context.Context, model *model.Model, key *model.Key, modelAgent *model.ModelAgent) {

	if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

		if model.IsEnableModelAgent {
			service.ModelAgent().RecordErrorModelAgentKey(ctx, modelAgent, key)
			service.ModelAgent().RecordErrorModelAgent(ctx, model, modelAgent)
		} else {
			service.Key().RecordErrorModelKey(ctx, model, key)
		}

	}, nil); err != nil {
		logger.Error(ctx, err)
	}
}
