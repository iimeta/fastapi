package app_key

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

type sAppKey struct {
	appKeyCache      *cache.Cache // [key]AppKey
	appKeyQuotaCache *cache.Cache // [key]Quota
}

func init() {
	service.RegisterAppKey(New())
}

func New() service.IAppKey {
	return &sAppKey{
		appKeyCache:      cache.New(),
		appKeyQuotaCache: cache.New(),
	}
}

// 根据secretKey获取应用密钥信息
func (s *sAppKey) GetBySecretKey(ctx context.Context, secretKey string) (*model.AppKey, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey GetBySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	key, err := dao.AppKey.FindOne(ctx, bson.M{"key": secretKey, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.AppKey{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		Key:                 key.Key,
		BillingMethods:      key.BillingMethods,
		Models:              key.Models,
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
	}, nil
}

// 应用密钥列表
func (s *sAppKey) List(ctx context.Context) ([]*model.AppKey, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey List time: %d", gtime.TimestampMilli()-now)
	}()

	results, err := dao.AppKey.Find(ctx, bson.M{}, &dao.FindOptions{SortFields: []string{"status", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.AppKey, 0)
	for _, result := range results {
		items = append(items, &model.AppKey{
			Id:                  result.Id,
			UserId:              result.UserId,
			AppId:               result.AppId,
			Key:                 result.Key,
			BillingMethods:      result.BillingMethods,
			Models:              result.Models,
			IsLimitQuota:        result.IsLimitQuota,
			Quota:               result.Quota,
			UsedQuota:           result.UsedQuota,
			QuotaExpiresRule:    result.QuotaExpiresRule,
			QuotaExpiresAt:      result.QuotaExpiresAt,
			QuotaExpiresMinutes: result.QuotaExpiresMinutes,
			IsBindGroup:         result.IsBindGroup,
			Group:               result.Group,
			IpWhitelist:         result.IpWhitelist,
			IpBlacklist:         result.IpBlacklist,
			Status:              result.Status,
			Rid:                 result.Rid,
		})
	}

	return items, nil
}

// 保存应用密钥信息到缓存
func (s *sAppKey) SaveCache(ctx context.Context, key *model.AppKey) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if key == nil {
		return errors.New("appKey is nil")
	}

	service.Session().SaveAppKey(ctx, key)

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
func (s *sAppKey) GetCache(ctx context.Context, secretKey string) (*model.AppKey, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey GetCache time: %d", gtime.TimestampMilli()-now)
	}()

	if key := service.Session().GetAppKey(ctx); key != nil {
		return key, nil
	}

	if appKeyCacheValue := s.appKeyCache.GetVal(ctx, secretKey); appKeyCacheValue != nil {
		key := appKeyCacheValue.(*model.AppKey)
		service.Session().SaveAppKey(ctx, key)
		return key, nil
	}

	return nil, errors.New("appKey is nil")
}

// 更新缓存中的应用密钥信息
func (s *sAppKey) UpdateCache(ctx context.Context, key *entity.AppKey) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey UpdateCache time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCache(ctx, &model.AppKey{
		Id:                  key.Id,
		UserId:              key.UserId,
		AppId:               key.AppId,
		Key:                 key.Key,
		BillingMethods:      key.BillingMethods,
		Models:              key.Models,
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
func (s *sAppKey) RemoveCache(ctx context.Context, secretKey string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey RemoveCache time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.appKeyCache.Remove(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := s.appKeyQuotaCache.Remove(ctx, secretKey); err != nil {
		logger.Error(ctx, err)
	}
}

// 应用密钥花费额度
func (s *sAppKey) SpendQuota(ctx context.Context, secretKey string, spendQuota, currentQuota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey SpendQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.AppKey.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
		"$inc": bson.M{
			"quota":      -spendQuota,
			"used_quota": spendQuota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := s.SaveCacheQuota(ctx, secretKey, currentQuota); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

// 应用密钥已用额度
func (s *sAppKey) UsedQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey UsedQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := dao.AppKey.UpdateOne(ctx, bson.M{"key": secretKey}, bson.M{
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
func (s *sAppKey) SaveCacheQuota(ctx context.Context, secretKey string, quota int) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey SaveCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.appKeyQuotaCache.Set(ctx, secretKey, quota, 0); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 获取缓存中的应用密钥额度
func (s *sAppKey) GetCacheQuota(ctx context.Context, secretKey string) int {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey GetCacheQuota time: %d", gtime.TimestampMilli()-now)
	}()

	if appKeyQuotaValue := s.appKeyQuotaCache.GetVal(ctx, secretKey); appKeyQuotaValue != nil {
		return appKeyQuotaValue.(int)
	}

	return 0
}

// 更新应用密钥额度过期时间
func (s *sAppKey) UpdateQuotaExpiresAt(ctx context.Context, key *model.AppKey) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey UpdateQuotaExpiresAt time: %d", gtime.TimestampMilli()-now)
	}()

	oldData, err := dao.AppKey.FindById(ctx, key.Id)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err = dao.AppKey.UpdateById(ctx, key.Id, bson.M{
		"quota_expires_rule": 1,
		"quota_expires_at":   gtime.Now().Add(time.Duration(key.QuotaExpiresMinutes) * time.Minute).TimestampMilli(),
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	newData, err := dao.AppKey.FindById(ctx, key.Id)
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
func (s *sAppKey) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sAppKey SubscribeKey time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sAppKey SubscribeKey: %s", gjson.MustEncodeString(message))

	var key *entity.AppKey
	switch message.Action {
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS, consts.ACTION_MODELS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCache(ctx, key)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCache(ctx, key.Key)
	}

	return nil
}
