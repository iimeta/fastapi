package model

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
)

type sApp struct{}

func init() {
	service.RegisterApp(New())
}

func New() service.IApp {
	return &sApp{}
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

// 变更订阅
func (s *sApp) Subscribe(ctx context.Context, msg string) error {

	app := new(entity.App)
	err := gjson.Unmarshal([]byte(msg), &app)
	if err != nil {
		logger.Error(ctx, err)
		return err
	}
	fmt.Println(gjson.MustEncodeString(app))

	return nil
}
