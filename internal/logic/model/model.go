package model

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
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

type sModel struct {
	modelCache *cache.Cache // [模型ID]Model
}

func init() {
	service.RegisterModel(New())
}

func New() service.IModel {
	return &sModel{
		modelCache: cache.New(),
	}
}

// 根据model获取模型信息
func (s *sModel) GetModel(ctx context.Context, m string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModel time: %d", gtime.TimestampMilli()-now)
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
		Prompt:             result.Prompt,
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
func (s *sModel) GetModelBySecretKey(ctx context.Context, m, secretKey string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelBySecretKey time: %d", gtime.TimestampMilli()-now)
	}()

	user, err := service.User().GetCacheUser(ctx, service.Session().GetUserId(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(user.Models) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Error(ctx, err)
		return nil, err
	}

	models, err := s.GetCacheList(ctx, user.Models...)
	if err != nil || len(models) != len(user.Models) {

		if models, err = s.List(ctx, user.Models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}

		if err = s.SaveCacheList(ctx, models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(models) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Error(ctx, err)
		return nil, err
	}

	userModelList := make([]*model.Model, 0)
	for _, v := range models {
		if v.Name == m {
			userModelList = append(userModelList, v)
			break
		}
	}

	for _, v := range models {
		if v.Model == m {
			userModelList = append(userModelList, v)
		}
	}

	if len(userModelList) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Error(ctx, err)
		return nil, err
	}

	app, err := service.App().GetCacheApp(ctx, service.Session().GetAppId(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	key, err := service.App().GetCacheAppKey(g.RequestFromCtx(ctx).GetCtx(), secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	keyModelList := make([]*model.Model, 0)
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
				keyModelList = append(keyModelList, v)
				break
			}
		}

		for _, v := range models {
			if v.Model == m {
				keyModelList = append(keyModelList, v)
			}
		}

		if len(keyModelList) == 0 {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Error(ctx, err)
			return nil, err
		}
	}

	appModelList := make([]*model.Model, 0)
	if len(app.Models) > 0 {

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

		for _, v := range models {
			if v.Name == m {
				appModelList = append(appModelList, v)
				break
			}
		}

		for _, v := range models {
			if v.Model == m {
				appModelList = append(appModelList, v)
			}
		}

		if len(appModelList) == 0 {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Error(ctx, err)
			return nil, err
		}
	}

	isModelDisabled := false
	if len(keyModelList) > 0 { // 密钥层模型权限

		for _, keyModel := range keyModelList {

			if keyModel.Name == m {

				if keyModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				if len(appModelList) > 0 {

					for _, appModel := range appModelList {
						// 应用层模型权限校验
						if keyModel.Id == appModel.Id {
							for _, userModel := range userModelList {
								// 用户层模型权限校验
								if appModel.Id == userModel.Id {
									return keyModel, nil
								}
							}
						}
					}

				} else {

					for _, userModel := range userModelList {
						// 用户层模型权限校验
						if keyModel.Id == userModel.Id {
							return keyModel, nil
						}
					}
				}
			}
		}

		for _, keyModel := range keyModelList {

			if keyModel.Model == m {

				if keyModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				if len(appModelList) > 0 {

					for _, appModel := range appModelList {
						// 应用层模型权限校验
						if keyModel.Id == appModel.Id {
							for _, userModel := range userModelList {
								// 用户层模型权限校验
								if appModel.Id == userModel.Id {
									return keyModel, nil
								}
							}
						}
					}

				} else {

					for _, userModel := range userModelList {
						// 用户层模型权限校验
						if keyModel.Id == userModel.Id {
							return keyModel, nil
						}
					}
				}
			}
		}

	} else if len(appModelList) > 0 { // 应用层模型权限

		for _, appModel := range appModelList {

			if appModel.Name == m {

				if appModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				for _, userModel := range userModelList {
					// 用户层模型权限校验
					if appModel.Id == userModel.Id {
						return appModel, nil
					}
				}
			}
		}

		for _, appModel := range appModelList {

			if appModel.Model == m {

				if appModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				for _, userModel := range userModelList {
					// 用户层模型权限校验
					if appModel.Id == userModel.Id {
						return appModel, nil
					}
				}
			}
		}

	} else if len(userModelList) > 0 { // 用户层模型权限

		for _, userModel := range userModelList {

			if userModel.Name == m {

				if userModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				return userModel, nil
			}
		}

		for _, userModel := range userModelList {

			if userModel.Model == m {

				if userModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				return userModel, nil
			}
		}
	}

	if isModelDisabled {
		err = errors.ERR_MODEL_DISABLED
		logger.Error(ctx, err)
		return nil, err
	}

	return nil, errors.ERR_MODEL_NOT_FOUND
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
			Prompt:             result.Prompt,
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
		if err := s.modelCache.Set(ctx, model.Id, model, 0); err != nil {
			logger.Error(ctx, err)
			return err
		}
	}

	if len(fields) > 0 {
		if _, err := redis.HSet(ctx, consts.API_MODELS_KEY, fields); err != nil {
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
		if modelCacheValue := s.modelCache.GetVal(ctx, id); modelCacheValue != nil {
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

		if s.modelCache.ContainsKey(ctx, result.Id) {
			continue
		}

		items = append(items, result)
		if err = s.modelCache.Set(ctx, result.Id, result, 0); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(items) == 0 {
		return nil, errors.New("models is nil")
	}

	return items, nil
}

// 更新缓存中的模型列表
func (s *sModel) UpdateCacheModel(ctx context.Context, oldData *entity.Model, newData *entity.Model) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel UpdateCacheModel time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCacheList(ctx, []*model.Model{{
		Id:                 newData.Id,
		Corp:               newData.Corp,
		Name:               newData.Name,
		Model:              newData.Model,
		Type:               newData.Type,
		Prompt:             newData.Prompt,
		PromptRatio:        newData.PromptRatio,
		CompletionRatio:    newData.CompletionRatio,
		DataFormat:         newData.DataFormat,
		IsPublic:           newData.IsPublic,
		IsEnableModelAgent: newData.IsEnableModelAgent,
		ModelAgents:        newData.ModelAgents,
		Status:             newData.Status,
	}}); err != nil {
		logger.Error(ctx, err)
	}

	// 用于处理oldData时判断作用
	newModelAgentMap := make(map[string]string)

	if newData.IsEnableModelAgent {

		for _, modelAgentId := range newData.ModelAgents {

			newModelAgentMap[modelAgentId] = modelAgentId

			modelAgent, err := service.ModelAgent().GetModelAgent(ctx, modelAgentId)
			if err != nil {
				logger.Error(ctx, err)
				continue
			}

			service.ModelAgent().UpdateCacheModelAgent(ctx, nil, modelAgent)
		}
	}

	if oldData != nil && oldData.IsEnableModelAgent {

		for _, modelAgentId := range oldData.ModelAgents {

			if newModelAgentMap[modelAgentId] == "" {

				modelAgent, err := service.ModelAgent().GetModelAgent(ctx, modelAgentId)
				if err != nil {
					logger.Error(ctx, err)
					continue
				}

				oldModelAgent := *modelAgent
				oldModelAgent.Models = append(oldModelAgent.Models, oldData.Id)

				service.ModelAgent().UpdateCacheModelAgent(ctx, &oldModelAgent, modelAgent)
			}
		}
	}
}

// 移除缓存中的模型列表
func (s *sModel) RemoveCacheModel(ctx context.Context, id string) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel RemoveCacheModel time: %d", gtime.TimestampMilli()-now)
	}()

	if _, err := s.modelCache.Remove(ctx, id); err != nil {
		logger.Error(ctx, err)
	}

	if _, err := redis.HDel(ctx, consts.API_MODELS_KEY, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 变更订阅
func (s *sModel) Subscribe(ctx context.Context, msg string) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel Subscribe time: %d", gtime.TimestampMilli()-now)
	}()

	message := new(model.SubMessage)
	if err := gjson.Unmarshal([]byte(msg), &message); err != nil {
		logger.Error(ctx, err)
		return err
	}
	logger.Infof(ctx, "sModel Subscribe: %s", gjson.MustEncodeString(message))

	var newData *entity.Model
	switch message.Action {
	case consts.ACTION_UPDATE:

		var oldData *entity.Model
		if message.OldData != nil {
			if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &oldData); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &newData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheModel(ctx, oldData, newData)

	case consts.ACTION_STATUS:

		if err := gjson.Unmarshal(gjson.MustEncode(message.NewData), &newData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.UpdateCacheModel(ctx, nil, newData)

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &newData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheModel(ctx, newData.Id)
	}

	return nil
}
