package app

import (
	"context"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/redis"
	"go.mongodb.org/mongo-driver/bson"
)

type sApp struct {
	appCacheMap *gmap.StrAnyMap
}

func init() {
	service.RegisterApp(New())
}

func New() service.IApp {
	return &sApp{
		appCacheMap: gmap.NewStrAnyMap(true),
	}
}

// 根据应用ID获取应用信息
func (s *sApp) GetApp(ctx context.Context, appId int) (*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetApp time: %d", gtime.TimestampMilli()-now)
	}()

	app, err := dao.App.FindOne(ctx, bson.M{"app_id": appId, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.App{
		Id:           app.Id,
		AppId:        app.AppId,
		Name:         app.Name,
		Type:         app.Type,
		Models:       app.Models,
		IsLimitQuota: app.IsLimitQuota,
		Quota:        app.Quota,
		IpWhitelist:  app.IpWhitelist,
		IpBlacklist:  app.IpBlacklist,
		Remark:       app.Remark,
		Status:       app.Status,
		UserId:       app.UserId,
	}, nil
}

// 应用列表
func (s *sApp) List(ctx context.Context) ([]*model.App, error) {

	filter := bson.M{
		"status": 1,
	}

	results, err := dao.App.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.App, 0)
	for _, result := range results {
		items = append(items, &model.App{
			Id:           result.Id,
			AppId:        result.AppId,
			Name:         result.Name,
			Type:         result.Type,
			Models:       result.Models,
			IsLimitQuota: result.IsLimitQuota,
			Quota:        result.Quota,
			IpWhitelist:  result.IpWhitelist,
			IpBlacklist:  result.IpBlacklist,
			Remark:       result.Remark,
			Status:       result.Status,
			UserId:       result.UserId,
		})
	}

	return items, nil
}

// 更改应用额度
func (s *sApp) ChangeQuota(ctx context.Context, appId, quota int) error {

	if err := dao.App.UpdateOne(ctx, bson.M{"app_id": appId}, bson.M{
		"$inc": bson.M{
			"quota": quota,
		},
	}); err != nil {
		logger.Error(ctx, err)
		return err
	}

	return nil
}

// 保存应用列表到缓存
func (s *sApp) SaveCacheList(ctx context.Context, apps []*model.App) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, app := range apps {
		fields[gconv.String(app.AppId)] = app
		s.appCacheMap.Set(gconv.String(app.AppId), app)
	}

	if len(fields) > 0 {
		if _, err := redis.HSet(ctx, consts.API_APPS_KEY, fields); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的应用列表
func (s *sApp) GetCacheList(ctx context.Context, appIds ...string) ([]*model.App, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sApp GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.App, 0)

	for _, id := range appIds {
		if appCacheValue := s.appCacheMap.Get(id); appCacheValue != nil {
			items = append(items, appCacheValue.(*model.App))
		}
	}

	if len(items) == len(appIds) {
		return items, nil
	}

	reply, err := redis.HMGet(ctx, consts.API_APPS_KEY, appIds...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		if len(items) != 0 {
			return items, nil
		}
		return nil, errors.New("apps is nil")
	}

	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.App)
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if s.appCacheMap.Get(gconv.String(result.AppId)) != nil {
			continue
		}

		if result.Status == 1 {
			items = append(items, result)
			s.appCacheMap.Set(gconv.String(result.AppId), result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("apps is nil")
	}

	return items, nil
}

// 变更订阅
func (s *sApp) Subscribe(ctx context.Context, msg string) error {

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sApp Subscribe: %s", gjson.MustEncodeString(message))

	var app *entity.App
	switch message.Action {
	case consts.ACTION_UPDATE, consts.ACTION_STATUS:
		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &app); err != nil {
			logger.Error(ctx, err)
			return err
		}
		service.Common().UpdateCacheApp(ctx, app)
	case consts.ACTION_DELETE:
		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &app); err != nil {
			logger.Error(ctx, err)
			return err
		}
		service.Common().RemoveCacheApp(ctx, app.AppId)
	}

	return nil
}

// 应用密钥变更订阅
func (s *sApp) SubscribeKey(ctx context.Context, msg string) error {

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sApp SubscribeKey: %s", gjson.MustEncodeString(message))

	var key *entity.Key
	switch message.Action {
	case consts.ACTION_UPDATE, consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		service.Common().UpdateCacheAppKey(ctx, key)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &key); err != nil {
			logger.Error(ctx, err)
			return err
		}

		service.Common().RemoveCacheAppKey(ctx, key.Key)
	}

	return nil
}
