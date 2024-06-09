package app

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
)

type sApp struct {
	appCache    *cache.Cache // [appId]App
	appKeyCache *cache.Cache // [key]Key
}

func init() {
	service.RegisterApp(New())
}

func New() service.IApp {
	return &sApp{
		appCache:    cache.New(),
		appKeyCache: cache.New(),
	}
}

// 根据应用ID获取应用信息
func (s *sApp) GetApp(ctx context.Context, appId int) (*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetApp time: %d", gtime.TimestampMilli()-now)
	}()

	app, err := dao.App.FindOne(ctx, bson.M{"app_id": appId, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.App{
		Id:             app.Id,
		AppId:          app.AppId,
		Name:           app.Name,
		Type:           app.Type,
		Models:         app.Models,
		IsLimitQuota:   app.IsLimitQuota,
		Quota:          app.Quota,
		UsedQuota:      app.UsedQuota,
		QuotaExpiresAt: app.QuotaExpiresAt,
		IpWhitelist:    app.IpWhitelist,
		IpBlacklist:    app.IpBlacklist,
		Remark:         app.Remark,
		Status:         app.Status,
		UserId:         app.UserId,
	}, nil
}

// 应用列表
func (s *sApp) List(ctx context.Context) ([]*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.App.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.App, 0)
	for _, result := range results {
		items = append(items, &model.App{
			Id:             result.Id,
			AppId:          result.AppId,
			Name:           result.Name,
			Type:           result.Type,
			Models:         result.Models,
			IsLimitQuota:   result.IsLimitQuota,
			Quota:          result.Quota,
			UsedQuota:      result.UsedQuota,
			QuotaExpiresAt: result.QuotaExpiresAt,
			IpWhitelist:    result.IpWhitelist,
			IpBlacklist:    result.IpBlacklist,
			Remark:         result.Remark,
			Status:         result.Status,
			UserId:         result.UserId,
		})
	}

	return items, nil
}

// 应用消费额度
func (s *sApp) SpendQuota(ctx context.Context, appId, quota, currentQuota int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	app, err := s.GetCacheApp(ctx, appId)
	if err != nil {
		logger.Error(ctx, err)
	}

	app.Quota = currentQuota
	app.UsedQuota += quota

	if err = s.SaveCacheApp(ctx, app); err != nil {
		logger.Error(ctx, err)
	}

	if err = dao.App.UpdateOne(ctx, bson.M{"app_id": appId}, bson.M{
		"$inc": bson.M{
			"quota":      -quota,
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 应用已用额度
func (s *sApp) UsedQuota(ctx context.Context, appId, quota int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	app, err := s.GetCacheApp(ctx, appId)
	if err != nil {
		logger.Error(ctx, err)
	}

	app.UsedQuota += quota

	if err = s.SaveCacheApp(ctx, app); err != nil {
		logger.Error(ctx, err)
	}

	if err = dao.App.UpdateOne(ctx, bson.M{"app_id": appId}, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存应用信息到缓存
func (s *sApp) SaveCacheApp(ctx context.Context, app *model.App) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if app == nil {
		return errors.New("app is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_APP_KEY, app.AppId), app); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveApp(ctx, app)

	if err := s.appCache.Set(ctx, app.AppId, app, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用信息
func (s *sApp) GetCacheApp(ctx context.Context, appId int) (*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if app := service.Session().GetApp(ctx); app != nil {
		return app, nil
	}

	if appCacheValue := s.appCache.GetVal(ctx, appId); appCacheValue != nil {
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

	if err = s.appCache.Set(ctx, app.AppId, app, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return app, nil
}

// 更新缓存中的应用信息
func (s *sApp) UpdateCacheApp(ctx context.Context, app *entity.App) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UpdateCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheApp(ctx, &model.App{
		Id:             app.Id,
		AppId:          app.AppId,
		Name:           app.Name,
		Type:           app.Type,
		Models:         app.Models,
		IsLimitQuota:   app.IsLimitQuota,
		Quota:          app.Quota,
		UsedQuota:      app.UsedQuota,
		QuotaExpiresAt: app.QuotaExpiresAt,
		IpWhitelist:    app.IpWhitelist,
		IpBlacklist:    app.IpBlacklist,
		Status:         app.Status,
		UserId:         app.UserId,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的应用信息
func (s *sApp) RemoveCacheApp(ctx context.Context, appId int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp RemoveCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.appCache.Remove(ctx, appId); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_APP_KEY, appId)); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存应用密钥信息到缓存
func (s *sApp) SaveCacheAppKey(ctx context.Context, key *model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if key == nil {
		return errors.New("key is nil")
	}

	if _, err := redis.Set(ctx, fmt.Sprintf(consts.API_APP_KEY_KEY, key.Key), key); err != nil {
		logger.Error(ctx, err)
		return err
	}

	service.Session().SaveKey(ctx, key)

	if err := s.appKeyCache.Set(ctx, key.Key, key, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用密钥信息
func (s *sApp) GetCacheAppKey(ctx context.Context, secretKey string) (*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if key := service.Session().GetKey(ctx); key != nil {
		return key, nil
	}

	if keyCacheValue := s.appKeyCache.GetVal(ctx, secretKey); keyCacheValue != nil {
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

	if err = s.appKeyCache.Set(ctx, key.Key, key, 0); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return key, nil
}

// 更新缓存中的应用密钥信息
func (s *sApp) UpdateCacheAppKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UpdateCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheAppKey(ctx, &model.Key{
		Id:             key.Id,
		UserId:         key.UserId,
		AppId:          key.AppId,
		Corp:           key.Corp,
		Key:            key.Key,
		Type:           key.Type,
		Models:         key.Models,
		ModelAgents:    key.ModelAgents,
		IsLimitQuota:   key.IsLimitQuota,
		Quota:          key.Quota,
		UsedQuota:      key.UsedQuota,
		QuotaExpiresAt: key.QuotaExpiresAt,
		RPM:            key.RPM,
		RPD:            key.RPD,
		IpWhitelist:    key.IpWhitelist,
		IpBlacklist:    key.IpBlacklist,
		Status:         key.Status,
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 移除缓存中的应用密钥信息
func (s *sApp) RemoveCacheAppKey(ctx context.Context, secretKey string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp RemoveCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.appKeyCache.Remove(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.Del(ctx, fmt.Sprintf(consts.API_APP_KEY_KEY, secretKey)); err != nil {
		logger.Error(ctx, err)
	}
}

// 密钥消费额度
func (s *sApp) AppKeySpendQuota(ctx context.Context, secretKey string, quota, currentQuota int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp AppKeySpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := s.GetCacheAppKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
	}

	key.Quota = currentQuota
	key.UsedQuota += quota

	if err = s.SaveCacheAppKey(ctx, key); err != nil {
		logger.Error(ctx, err)
	}

	if err = dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"quota":      -quota,
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 密钥已用额度
func (s *sApp) AppKeyUsedQuota(ctx context.Context, secretKey string, quota int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp AppKeyUsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := s.GetCacheAppKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
	}

	key.UsedQuota += quota

	if err = s.SaveCacheAppKey(ctx, key); err != nil {
		logger.Error(ctx, err)
	}

	if err = dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sApp) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sApp Subscribe: %s", gjson.MustEncodeString(message))

	var app *entity.App
	switch message.Action {
	case consts.ACTION_UPDATE, consts.ACTION_STATUS, consts.ACTION_MODELS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &app); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheApp(ctx, app)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &app); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheApp(ctx, app.AppId)
	}

	return nil
}

// 应用密钥变更订阅
func (s *sApp) SubscribeKey(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SubscribeKey time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sApp SubscribeKey: %s", gjson.MustEncodeString(message))

	var key *entity.Key
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheAppKey(ctx, key)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheAppKey(ctx, key.Key)
	}

	return nil
}
