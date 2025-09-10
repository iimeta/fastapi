package app

import (
	"context"
	"time"

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
	appCache         *cache.Cache // [appId]App
	appKeyCache      *cache.Cache // [key]Key
	appQuotaCache    *cache.Cache // [appId]Quota
	appKeyQuotaCache *cache.Cache // [key]Quota
}

func init() {
	service.RegisterApp(New())
}

func New() service.IApp {
	return &sApp{
		appCache:         cache.New(),
		appKeyCache:      cache.New(),
		appQuotaCache:    cache.New(),
		appKeyQuotaCache: cache.New(),
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
		UserId:         app.UserId,
		AppId:          app.AppId,
		Name:           app.Name,
		Models:         app.Models,
		IsLimitQuota:   app.IsLimitQuota,
		Quota:          app.Quota,
		UsedQuota:      app.UsedQuota,
		QuotaExpiresAt: app.QuotaExpiresAt,
		IsBindGroup:    app.IsBindGroup,
		Group:          app.Group,
		IpWhitelist:    app.IpWhitelist,
		IpBlacklist:    app.IpBlacklist,
		Remark:         app.Remark,
		Status:         app.Status,
		Rid:            app.Rid,
	}, nil
}

// 应用列表
func (s *sApp) List(ctx context.Context) ([]*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.App.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.App, 0)
	for _, result := range results {
		items = append(items, &model.App{
			Id:             result.Id,
			UserId:         result.UserId,
			AppId:          result.AppId,
			Name:           result.Name,
			Models:         result.Models,
			IsLimitQuota:   result.IsLimitQuota,
			Quota:          result.Quota,
			UsedQuota:      result.UsedQuota,
			QuotaExpiresAt: result.QuotaExpiresAt,
			IsBindGroup:    result.IsBindGroup,
			Group:          result.Group,
			IpWhitelist:    result.IpWhitelist,
			IpBlacklist:    result.IpBlacklist,
			Remark:         result.Remark,
			Status:         result.Status,
			Rid:            result.Rid,
		})
	}

	return items, nil
}

// 应用花费额度
func (s *sApp) SpendQuota(ctx context.Context, appId, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.App.UpdateOne(ctx, bson.M{"app_id": appId}, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.SaveCacheAppQuota(ctx, appId, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 应用已用额度
func (s *sApp) UsedQuota(ctx context.Context, appId, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.App.UpdateOne(ctx, bson.M{"app_id": appId}, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
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

	service.Session().SaveApp(ctx, app)

	if err := s.appCache.Set(ctx, app.AppId, app, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.appQuotaCache.Set(ctx, app.AppId, app.Quota, 0); err != nil {
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
		app := appCacheValue.(*model.App)
		service.Session().SaveApp(ctx, app)
		return app, nil
	}

	return nil, errors.New("app is nil")
}

// 更新缓存中的应用信息
func (s *sApp) UpdateCacheApp(ctx context.Context, app *entity.App) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UpdateCacheApp time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheApp(ctx, &model.App{
		Id:             app.Id,
		UserId:         app.UserId,
		AppId:          app.AppId,
		Name:           app.Name,
		Models:         app.Models,
		IsLimitQuota:   app.IsLimitQuota,
		Quota:          app.Quota,
		UsedQuota:      app.UsedQuota,
		QuotaExpiresAt: app.QuotaExpiresAt,
		IsBindGroup:    app.IsBindGroup,
		Group:          app.Group,
		IpWhitelist:    app.IpWhitelist,
		IpBlacklist:    app.IpBlacklist,
		Status:         app.Status,
		Rid:            app.Rid,
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

	if _, err := s.appQuotaCache.Remove(ctx, appId); err != nil {
		logger.Error(ctx, err)
	}
}

// 保存应用额度到缓存
func (s *sApp) SaveCacheAppQuota(ctx context.Context, appId, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheAppQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.appQuotaCache.Set(ctx, appId, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用额度
func (s *sApp) GetCacheAppQuota(ctx context.Context, appId int) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetCacheAppQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if appQuotaValue := s.appQuotaCache.GetVal(ctx, appId); appQuotaValue != nil {
		return appQuotaValue.(int)
	}

	return 0
}

// 保存应用密钥信息到缓存
func (s *sApp) SaveCacheAppKey(ctx context.Context, key *model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if key == nil {
		return errors.New("appKey is nil")
	}

	service.Session().SaveKey(ctx, key)

	if err := s.appKeyCache.Set(ctx, key.Key, key, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.appKeyQuotaCache.Set(ctx, key.Key, key.Quota, 0); err != nil {
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

	if appKeyCacheValue := s.appKeyCache.GetVal(ctx, secretKey); appKeyCacheValue != nil {
		key := appKeyCacheValue.(*model.Key)
		service.Session().SaveKey(ctx, key)
		return key, nil
	}

	return nil, errors.New("appKey is nil")
}

// 更新缓存中的应用密钥信息
func (s *sApp) UpdateCacheAppKey(ctx context.Context, key *entity.Key) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UpdateCacheAppKey time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheAppKey(ctx, &model.Key{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		ProviderId:          key.ProviderId,
		Key:                 key.Key,
		Type:                key.Type,
		Models:              key.Models,
		ModelAgents:         key.ModelAgents,
		IsLimitQuota:        key.IsLimitQuota,
		Quota:               key.Quota,
		UsedQuota:           key.UsedQuota,
		QuotaExpiresRule:    key.QuotaExpiresRule,
		QuotaExpiresAt:      key.QuotaExpiresAt,
		QuotaExpiresMinutes: key.QuotaExpiresMinutes,
		IsBindGroup:         key.IsBindGroup,
		Group:               key.Group,
		IpWhitelist:         key.IpWhitelist,
		IpBlacklist:         key.IpBlacklist,
		Status:              key.Status,
		Rid:                 key.Rid,
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

	if _, err := s.appKeyQuotaCache.Remove(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
	}
}

// 应用密钥花费额度
func (s *sApp) AppKeySpendQuota(ctx context.Context, secretKey string, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp AppKeySpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.SaveCacheAppKeyQuota(ctx, secretKey, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 应用密钥已用额度
func (s *sApp) AppKeyUsedQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp AppKeyUsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.Key.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"used_quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存应用密钥额度到缓存
func (s *sApp) SaveCacheAppKeyQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheAppKeyQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.appKeyQuotaCache.Set(ctx, secretKey, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用密钥额度
func (s *sApp) GetCacheAppKeyQuota(ctx context.Context, secretKey string) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetCacheAppKeyQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if appKeyQuotaValue := s.appKeyQuotaCache.GetVal(ctx, secretKey); appKeyQuotaValue != nil {
		return appKeyQuotaValue.(int)
	}

	return 0
}

// 更新应用密钥额度过期时间
func (s *sApp) UpdateAppKeyQuotaExpiresAt(ctx context.Context, key *model.Key) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp UpdateAppKeyQuotaExpiresAt time: %d", gtime.TimestampMilli()-now)
	}()

	oldData, err := dao.Key.FindById(ctx, key.Id)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err = dao.Key.UpdateById(ctx, key.Id, bson.M{
		"quota_expires_rule": 1,
		"quota_expires_at":   gtime.Now().Add(time.Duration(key.QuotaExpiresMinutes) * time.Minute).TimestampMilli(),
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	newData, err := dao.Key.FindById(ctx, key.Id)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if _, err = redis.Publish(ctx, consts.CHANGE_CHANNEL_APP_KEY, model.PubMessage{
		Action:  consts.ACTION_UPDATE,
		OldData: oldData,
		NewData: newData,
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
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
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS, consts.ACTION_MODELS:

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
