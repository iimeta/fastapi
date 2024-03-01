package model

import (
	"context"
	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
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

type sModel struct {
	modelCacheMap *gmap.StrAnyMap
}

func init() {
	service.RegisterModel(New())
}

func New() service.IModel {
	return &sModel{
		modelCacheMap: gmap.NewStrAnyMap(true),
	}
}

// 根据model获取模型信息
func (s *sModel) GetModel(ctx context.Context, m string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModel time: %d", gtime.TimestampMilli()-now)
	}()

	result, err := dao.Model.FindOne(ctx, bson.M{"model": m, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Model{
		Id:                 result.Id,
		Corp:               result.Corp,
		Name:               result.Name,
		Model:              result.Model,
		Type:               result.Type,
		PromptRatio:        result.PromptRatio,
		CompletionRatio:    result.CompletionRatio,
		DataFormat:         result.DataFormat,
		IsEnableModelAgent: result.IsEnableModelAgent,
		ModelAgents:        result.ModelAgents,
		IsPublic:           result.IsPublic,
		Remark:             result.Remark,
		Status:             result.Status,
	}, nil
}

// 根据model和secretKey获取模型信息
func (s *sModel) GetModelBySecretKey(ctx context.Context, m, secretKey string) (md *model.Model, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "GetModelBySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	app, err := service.Common().GetCacheApp(ctx, service.Session().GetAppId(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(app.Models) == 0 {
		err = errors.ERR_PERMISSION_DENIED
		logger.Error(ctx, err)
		return nil, err
	}

	key, err := service.Common().GetCacheKey(g.RequestFromCtx(ctx).GetCtx(), secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(key.Models) > 0 {

		models, err := s.GetCacheList(ctx, key.Models...)
		if err != nil || len(models) != len(key.Models) {

			if models, err = s.List(ctx, key.Models); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			if err = s.SaveCacheList(ctx, models); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		for _, v := range models {
			if v.Name == m {
				md = &model.Model{
					Id:                 v.Id,
					Corp:               v.Corp,
					Name:               v.Name,
					Model:              v.Model,
					Type:               v.Type,
					PromptRatio:        v.PromptRatio,
					CompletionRatio:    v.CompletionRatio,
					DataFormat:         v.DataFormat,
					IsEnableModelAgent: v.IsEnableModelAgent,
					ModelAgents:        v.ModelAgents,
					IsPublic:           v.IsPublic,
					Remark:             v.Remark,
					Status:             v.Status,
				}
				break
			}
		}

		if md == nil {
			for _, v := range models {
				if v.Model == m {
					md = &model.Model{
						Id:                 v.Id,
						Corp:               v.Corp,
						Name:               v.Name,
						Model:              v.Model,
						Type:               v.Type,
						PromptRatio:        v.PromptRatio,
						CompletionRatio:    v.CompletionRatio,
						DataFormat:         v.DataFormat,
						IsEnableModelAgent: v.IsEnableModelAgent,
						ModelAgents:        v.ModelAgents,
						IsPublic:           v.IsPublic,
						Remark:             v.Remark,
						Status:             v.Status,
					}
					break
				}
			}
		}

		if md == nil {
			return nil, errors.ERR_PERMISSION_DENIED
		}
	}

	models, err := s.GetCacheList(ctx, app.Models...)
	if err != nil || len(models) != len(app.Models) {

		if models, err = s.List(ctx, app.Models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if err = s.SaveCacheList(ctx, models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(models) == 0 {
		err = errors.ERR_PERMISSION_DENIED
		logger.Error(ctx, err)
		return nil, err
	}

	for _, v := range models {

		if md != nil && md.Id == v.Id {
			return md, nil
		}

		if v.Name == m {
			return &model.Model{
				Id:                 v.Id,
				Corp:               v.Corp,
				Name:               v.Name,
				Model:              v.Model,
				Type:               v.Type,
				PromptRatio:        v.PromptRatio,
				CompletionRatio:    v.CompletionRatio,
				DataFormat:         v.DataFormat,
				IsEnableModelAgent: v.IsEnableModelAgent,
				ModelAgents:        v.ModelAgents,
				IsPublic:           v.IsPublic,
				Remark:             v.Remark,
				Status:             v.Status,
			}, nil
		}
	}

	for _, v := range models {

		if md != nil && md.Id == v.Id {
			return md, nil
		}

		if v.Model == m {
			return &model.Model{
				Id:                 v.Id,
				Corp:               v.Corp,
				Name:               v.Name,
				Model:              v.Model,
				Type:               v.Type,
				PromptRatio:        v.PromptRatio,
				CompletionRatio:    v.CompletionRatio,
				DataFormat:         v.DataFormat,
				IsEnableModelAgent: v.IsEnableModelAgent,
				ModelAgents:        v.ModelAgents,
				IsPublic:           v.IsPublic,
				Remark:             v.Remark,
				Status:             v.Status,
			}, nil
		}
	}

	return nil, errors.ERR_PERMISSION_DENIED
}

// 模型列表
func (s *sModel) List(ctx context.Context, ids []string) ([]*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel List time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{
		"_id": bson.M{
			"$in": ids,
		},
		"status": 1,
	}

	results, err := dao.Model.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {
		items = append(items, &model.Model{
			Id:                 result.Id,
			Corp:               result.Corp,
			Name:               result.Name,
			Model:              result.Model,
			Type:               result.Type,
			PromptRatio:        result.PromptRatio,
			CompletionRatio:    result.CompletionRatio,
			DataFormat:         result.DataFormat,
			IsEnableModelAgent: result.IsEnableModelAgent,
			ModelAgents:        result.ModelAgents,
			Remark:             result.Remark,
			Status:             result.Status,
		})
	}

	return items, nil
}

// 保存模型列表到缓存
func (s *sModel) SaveCacheList(ctx context.Context, models []*model.Model) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	fields := g.Map{}
	for _, model := range models {
		fields[model.Id] = model
		s.modelCacheMap.Set(model.Id, model)
	}

	if len(fields) > 0 {
		_, err := redis.HSet(ctx, consts.API_MODELS_KEY, fields)
		if err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	return nil
}

// 获取缓存中的模型列表
func (s *sModel) GetCacheList(ctx context.Context, ids ...string) ([]*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	items := make([]*model.Model, 0)

	for _, id := range ids {
		modelCacheValue := s.modelCacheMap.Get(id)
		if modelCacheValue != nil {
			items = append(items, modelCacheValue.(*model.Model))
		}
	}

	if len(items) == len(ids) {
		return items, nil
	}

	reply, err := redis.HMGet(ctx, consts.API_MODELS_KEY, ids...)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if reply == nil || len(reply) == 0 {
		if len(items) != 0 {
			return items, nil
		}
		return nil, errors.New("models is nil")
	}

	for _, str := range reply.Strings() {

		if str == "" {
			continue
		}

		result := new(model.Model)
		if err = gjson.Unmarshal([]byte(str), &result); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if s.modelCacheMap.Get(result.Id) != nil {
			continue
		}

		if result.Status == 1 {
			items = append(items, result)
			s.modelCacheMap.Set(result.Id, result)
		}
	}

	if len(items) == 0 {
		return nil, errors.New("models is nil")
	}

	return items, nil
}

// 移除缓存中的模型列表
func (s *sModel) RemoveCacheModel(ctx context.Context, id string) {

	s.modelCacheMap.Remove(id)

	if _, err := redis.HDel(ctx, consts.API_MODELS_KEY, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sModel) Subscribe(ctx context.Context, msg string) error {

	model := new(entity.Model)
	if err := gjson.Unmarshal([]byte(msg), &model); err != nil {
		logger.Error(ctx, err)
		return err
	}

	logger.Infof(ctx, "sModel Subscribe: %s", gjson.MustEncodeString(model))

	s.RemoveCacheModel(ctx, model.Id)

	return nil
}
