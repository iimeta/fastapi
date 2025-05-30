package model

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/entity"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/cache"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
	"slices"
	"strconv"
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
		Id:                   result.Id,
		Corp:                 result.Corp,
		Name:                 result.Name,
		Model:                result.Model,
		Type:                 result.Type,
		BaseUrl:              result.BaseUrl,
		Path:                 result.Path,
		IsEnablePresetConfig: result.IsEnablePresetConfig,
		PresetConfig:         result.PresetConfig,
		TextQuota:            result.TextQuota,
		ImageQuota:           result.ImageQuota,
		AudioQuota:           result.AudioQuota,
		MultimodalQuota:      result.MultimodalQuota,
		RealtimeQuota:        result.RealtimeQuota,
		MultimodalAudioQuota: result.MultimodalAudioQuota,
		MidjourneyQuotas:     result.MidjourneyQuotas,
		DataFormat:           result.DataFormat,
		IsPublic:             result.IsPublic,
		IsEnableModelAgent:   result.IsEnableModelAgent,
		LbStrategy:           result.LbStrategy,
		ModelAgents:          result.ModelAgents,
		IsEnableForward:      result.IsEnableForward,
		ForwardConfig:        result.ForwardConfig,
		IsEnableFallback:     result.IsEnableFallback,
		FallbackConfig:       result.FallbackConfig,
		Remark:               result.Remark,
		Status:               result.Status,
	}, nil
}

// 根据模型ID获取模型信息
func (s *sModel) GetModelById(ctx context.Context, id string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelById time: %d", gtime.TimestampMilli()-now)
	}()

	result, err := dao.Model.FindById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return &model.Model{
		Id:                   result.Id,
		Corp:                 result.Corp,
		Name:                 result.Name,
		Model:                result.Model,
		Type:                 result.Type,
		BaseUrl:              result.BaseUrl,
		Path:                 result.Path,
		IsEnablePresetConfig: result.IsEnablePresetConfig,
		PresetConfig:         result.PresetConfig,
		TextQuota:            result.TextQuota,
		ImageQuota:           result.ImageQuota,
		AudioQuota:           result.AudioQuota,
		MultimodalQuota:      result.MultimodalQuota,
		RealtimeQuota:        result.RealtimeQuota,
		MultimodalAudioQuota: result.MultimodalAudioQuota,
		MidjourneyQuotas:     result.MidjourneyQuotas,
		DataFormat:           result.DataFormat,
		IsPublic:             result.IsPublic,
		IsEnableModelAgent:   result.IsEnableModelAgent,
		LbStrategy:           result.LbStrategy,
		ModelAgents:          result.ModelAgents,
		IsEnableForward:      result.IsEnableForward,
		ForwardConfig:        result.ForwardConfig,
		IsEnableFallback:     result.IsEnableFallback,
		FallbackConfig:       result.FallbackConfig,
		Remark:               result.Remark,
		Status:               result.Status,
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
		if models, err = s.GetModelListAndSaveCacheList(ctx, user.Models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(models) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Error(ctx, err)
		return nil, err
	}

	userModels := make([]*model.Model, 0)
	userWildcardModels := make([]*model.Model, 0)

	for _, v := range models {

		if gstr.Contains(v.Name, "*") {
			userWildcardModels = append(userWildcardModels, v)
		}

		if v.Name == m {
			userModels = append(userModels, v)
			break
		}
	}

	for _, v := range models {

		if gstr.Contains(v.Model, "*") {
			userWildcardModels = append(userWildcardModels, v)
		}

		if v.Model == m {
			userModels = append(userModels, v)
		}
	}

	if len(userModels) == 0 && len(userWildcardModels) > 0 {

		for _, v := range userWildcardModels {
			if gregex.IsMatchString(v.Name, m) {
				userModels = append(userModels, v)
			}
		}

		for _, v := range userWildcardModels {
			if gregex.IsMatchString(v.Model, m) {
				userModels = append(userModels, v)
			}
		}
	}

	if len(userModels) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Error(ctx, err)
		return nil, err
	}

	app, err := service.App().GetCacheApp(ctx, service.Session().GetAppId(ctx))
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	key, err := service.App().GetCacheAppKey(ctx, secretKey)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	keyModels := make([]*model.Model, 0)
	keyWildcardModels := make([]*model.Model, 0)

	if len(key.Models) > 0 {

		models, err = s.GetCacheList(ctx, key.Models...)
		if err != nil || len(models) != len(key.Models) {
			if models, err = s.GetModelListAndSaveCacheList(ctx, key.Models); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		for _, v := range models {

			if gstr.Contains(v.Name, "*") {
				keyWildcardModels = append(keyWildcardModels, v)
			}

			if v.Name == m {
				keyModels = append(keyModels, v)
				break
			}
		}

		for _, v := range models {

			if gstr.Contains(v.Model, "*") {
				keyWildcardModels = append(keyWildcardModels, v)
			}

			if v.Model == m {
				keyModels = append(keyModels, v)
			}
		}

		if len(keyModels) == 0 && len(keyWildcardModels) > 0 {

			for _, v := range keyWildcardModels {
				if gregex.IsMatchString(v.Name, m) {
					keyModels = append(keyModels, v)
				}
			}

			for _, v := range keyWildcardModels {
				if gregex.IsMatchString(v.Model, m) {
					keyModels = append(keyModels, v)
				}
			}
		}

		if len(keyModels) == 0 {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Error(ctx, err)
			return nil, err
		}
	}

	appModels := make([]*model.Model, 0)
	appWildcardModels := make([]*model.Model, 0)

	if len(app.Models) > 0 {

		models, err = s.GetCacheList(ctx, app.Models...)
		if err != nil || len(models) != len(app.Models) {
			if models, err = s.GetModelListAndSaveCacheList(ctx, app.Models); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

		for _, v := range models {

			if gstr.Contains(v.Name, "*") {
				appWildcardModels = append(appWildcardModels, v)
			}

			if v.Name == m {
				appModels = append(appModels, v)
				break
			}
		}

		for _, v := range models {

			if gstr.Contains(v.Model, "*") {
				appWildcardModels = append(appWildcardModels, v)
			}

			if v.Model == m {
				appModels = append(appModels, v)
			}
		}

		if len(appModels) == 0 && len(appWildcardModels) > 0 {

			for _, v := range appWildcardModels {
				if gregex.IsMatchString(v.Name, m) {
					appModels = append(appModels, v)
				}
			}

			for _, v := range appWildcardModels {
				if gregex.IsMatchString(v.Model, m) {
					appModels = append(appModels, v)
				}
			}
		}

		if len(appModels) == 0 {
			err = errors.ERR_MODEL_NOT_FOUND
			logger.Error(ctx, err)
			return nil, err
		}
	}

	isModelDisabled := false
	if len(keyModels) > 0 { // 密钥层模型权限

		for _, keyModel := range keyModels {

			if keyModel.Name == m || (len(keyWildcardModels) > 0 && gregex.IsMatchString(keyModel.Name, m)) {

				if keyModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				if len(appModels) > 0 {

					for _, appModel := range appModels {
						// 应用层模型权限校验
						if keyModel.Id == appModel.Id {
							for _, userModel := range userModels {
								// 用户层模型权限校验
								if appModel.Id == userModel.Id {
									return keyModel, nil
								}
							}
						}
					}

				} else {

					for _, userModel := range userModels {
						// 用户层模型权限校验
						if keyModel.Id == userModel.Id {
							return keyModel, nil
						}
					}
				}
			}
		}

		for _, keyModel := range keyModels {

			if keyModel.Model == m || (len(keyWildcardModels) > 0 && gregex.IsMatchString(keyModel.Model, m)) {

				if keyModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				if len(appModels) > 0 {

					for _, appModel := range appModels {
						// 应用层模型权限校验
						if keyModel.Id == appModel.Id {
							for _, userModel := range userModels {
								// 用户层模型权限校验
								if appModel.Id == userModel.Id {
									return keyModel, nil
								}
							}
						}
					}

				} else {

					for _, userModel := range userModels {
						// 用户层模型权限校验
						if keyModel.Id == userModel.Id {
							return keyModel, nil
						}
					}
				}
			}
		}

	} else if len(appModels) > 0 { // 应用层模型权限

		for _, appModel := range appModels {

			if appModel.Name == m || (len(appWildcardModels) > 0 && gregex.IsMatchString(appModel.Name, m)) {

				if appModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				for _, userModel := range userModels {
					// 用户层模型权限校验
					if appModel.Id == userModel.Id {
						return appModel, nil
					}
				}
			}
		}

		for _, appModel := range appModels {

			if appModel.Model == m || (len(appWildcardModels) > 0 && gregex.IsMatchString(appModel.Model, m)) {

				if appModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				for _, userModel := range userModels {
					// 用户层模型权限校验
					if appModel.Id == userModel.Id {
						return appModel, nil
					}
				}
			}
		}

	} else if len(userModels) > 0 { // 用户层模型权限

		for _, userModel := range userModels {

			if userModel.Name == m || (len(userWildcardModels) > 0 && gregex.IsMatchString(userModel.Name, m)) {

				if userModel.Status == 2 {
					isModelDisabled = true
					continue
				}

				return userModel, nil
			}
		}

		for _, userModel := range userModels {

			if userModel.Model == m || (len(userWildcardModels) > 0 && gregex.IsMatchString(userModel.Model, m)) {

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

// 根据model和group获取模型信息
func (s *sModel) GetModelByGroup(ctx context.Context, m string, group *model.Group) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelByGroup time: %d", gtime.TimestampMilli()-now)
	}()

	if len(group.Models) == 0 {
		err := errors.ERR_MODEL_NOT_FOUND
		logger.Info(ctx, err)
		return nil, err
	}

	models, err := s.GetCacheList(ctx, group.Models...)
	if err != nil || len(models) != len(group.Models) {
		if models, err = s.GetModelListAndSaveCacheList(ctx, group.Models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if len(models) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Info(ctx, err)
		return nil, err
	}

	groupModels := make([]*model.Model, 0)
	groupWildcardModels := make([]*model.Model, 0)

	for _, v := range models {

		if gstr.Contains(v.Name, "*") {
			groupWildcardModels = append(groupWildcardModels, v)
		}

		if v.Name == m {
			groupModels = append(groupModels, v)
			break
		}
	}

	for _, v := range models {

		if gstr.Contains(v.Model, "*") {
			groupWildcardModels = append(groupWildcardModels, v)
		}

		if v.Model == m {
			groupModels = append(groupModels, v)
		}
	}

	if len(groupModels) == 0 && len(groupWildcardModels) > 0 {

		for _, v := range groupWildcardModels {
			if gregex.IsMatchString(v.Name, m) {
				groupModels = append(groupModels, v)
			}
		}

		for _, v := range groupWildcardModels {
			if gregex.IsMatchString(v.Model, m) {
				groupModels = append(groupModels, v)
			}
		}
	}

	if len(groupModels) == 0 {
		err = errors.ERR_MODEL_NOT_FOUND
		logger.Info(ctx, err)
		return nil, err
	}

	isModelDisabled := false

	for _, groupModel := range groupModels {

		if groupModel.Name == m || (len(groupWildcardModels) > 0 && gregex.IsMatchString(groupModel.Name, m)) {

			if groupModel.Status == 2 {
				isModelDisabled = true
				continue
			}

			return groupModel, nil
		}
	}

	for _, groupModel := range groupModels {

		if groupModel.Model == m || (len(groupWildcardModels) > 0 && gregex.IsMatchString(groupModel.Model, m)) {

			if groupModel.Status == 2 {
				isModelDisabled = true
				continue
			}

			return groupModel, nil
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
	}

	results, err := dao.Model.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {
		items = append(items, &model.Model{
			Id:                   result.Id,
			Corp:                 result.Corp,
			Name:                 result.Name,
			Model:                result.Model,
			Type:                 result.Type,
			BaseUrl:              result.BaseUrl,
			Path:                 result.Path,
			IsEnablePresetConfig: result.IsEnablePresetConfig,
			PresetConfig:         result.PresetConfig,
			TextQuota:            result.TextQuota,
			ImageQuota:           result.ImageQuota,
			AudioQuota:           result.AudioQuota,
			MultimodalQuota:      result.MultimodalQuota,
			RealtimeQuota:        result.RealtimeQuota,
			MultimodalAudioQuota: result.MultimodalAudioQuota,
			MidjourneyQuotas:     result.MidjourneyQuotas,
			DataFormat:           result.DataFormat,
			IsPublic:             result.IsPublic,
			IsEnableModelAgent:   result.IsEnableModelAgent,
			LbStrategy:           result.LbStrategy,
			ModelAgents:          result.ModelAgents,
			IsEnableForward:      result.IsEnableForward,
			ForwardConfig:        result.ForwardConfig,
			IsEnableFallback:     result.IsEnableFallback,
			FallbackConfig:       result.FallbackConfig,
			Remark:               result.Remark,
			Status:               result.Status,
			CreatedAt:            result.CreatedAt,
		})
	}

	return items, nil
}

// 全部模型列表
func (s *sModel) ListAll(ctx context.Context) ([]*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel ListAll time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{}

	results, err := dao.Model.Find(ctx, filter, &dao.FindOptions{SortFields: []string{"status", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {
		items = append(items, &model.Model{
			Id:                   result.Id,
			Corp:                 result.Corp,
			Name:                 result.Name,
			Model:                result.Model,
			Type:                 result.Type,
			BaseUrl:              result.BaseUrl,
			Path:                 result.Path,
			IsEnablePresetConfig: result.IsEnablePresetConfig,
			PresetConfig:         result.PresetConfig,
			TextQuota:            result.TextQuota,
			ImageQuota:           result.ImageQuota,
			AudioQuota:           result.AudioQuota,
			MultimodalQuota:      result.MultimodalQuota,
			RealtimeQuota:        result.RealtimeQuota,
			MultimodalAudioQuota: result.MultimodalAudioQuota,
			MidjourneyQuotas:     result.MidjourneyQuotas,
			DataFormat:           result.DataFormat,
			IsPublic:             result.IsPublic,
			IsEnableModelAgent:   result.IsEnableModelAgent,
			LbStrategy:           result.LbStrategy,
			ModelAgents:          result.ModelAgents,
			IsEnableForward:      result.IsEnableForward,
			ForwardConfig:        result.ForwardConfig,
			IsEnableFallback:     result.IsEnableFallback,
			FallbackConfig:       result.FallbackConfig,
			Remark:               result.Remark,
			Status:               result.Status,
			CreatedAt:            result.CreatedAt,
		})
	}

	return items, nil
}

// 获取模型与密钥列表
func (s *sModel) GetModelsAndKeys(ctx context.Context) ([]*model.Model, map[string][]*model.Key, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelsAndKeys time: %d", gtime.TimestampMilli()-now)
	}()

	models, err := s.ListAll(ctx)
	if err != nil {
		logger.Error(ctx, err)
		return nil, nil, err
	}

	results, err := dao.Key.Find(ctx, bson.M{"type": 2, "is_agents_only": false}, &dao.FindOptions{SortFields: []string{"status", "-weight", "-updated_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, nil, err
	}

	modelKeyMap := make(map[string][]*model.Key)
	for _, result := range results {

		key := &model.Key{
			Id:                  result.Id,
			UserId:              result.UserId,
			AppId:               result.AppId,
			Corp:                result.Corp,
			Key:                 result.Key,
			Type:                result.Type,
			Weight:              result.Weight,
			Models:              result.Models,
			ModelAgents:         result.ModelAgents,
			IsLimitQuota:        result.IsLimitQuota,
			Quota:               result.Quota,
			UsedQuota:           result.UsedQuota,
			QuotaExpiresRule:    result.QuotaExpiresRule,
			QuotaExpiresAt:      result.QuotaExpiresAt,
			QuotaExpiresMinutes: result.QuotaExpiresMinutes,
			IpWhitelist:         result.IpWhitelist,
			IpBlacklist:         result.IpBlacklist,
			Status:              result.Status,
		}

		for _, modelId := range result.Models {
			modelKeyMap[modelId] = append(modelKeyMap[modelId], key)
		}
	}

	return models, modelKeyMap, nil
}

// 根据模型ID获取模型信息并保存到缓存
func (s *sModel) GetModelAndSaveCache(ctx context.Context, id string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelAndSaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	model, err := s.GetModelById(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if model != nil {
		if err = s.SaveCache(ctx, model); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return model, nil
}

// 获取模型列表并保存到缓存
func (s *sModel) GetModelListAndSaveCacheList(ctx context.Context, ids []string) ([]*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetModelListAndSaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	models, err := s.List(ctx, ids)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(models) > 0 {
		if err = s.SaveCacheList(ctx, models); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	return models, nil
}

// 保存模型到缓存
func (s *sModel) SaveCache(ctx context.Context, m *model.Model) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel SaveCache time: %d", gtime.TimestampMilli()-now)
	}()

	return s.SaveCacheList(ctx, []*model.Model{m})
}

// 保存模型列表到缓存
func (s *sModel) SaveCacheList(ctx context.Context, models []*model.Model) error {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel SaveCacheList time: %d", gtime.TimestampMilli()-now)
	}()

	for _, model := range models {
		if err := s.modelCache.Set(ctx, model.Id, model, 0); err != nil {
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

	if len(items) == 0 {
		return nil, errors.New("models is nil")
	}

	return items, nil
}

// 获取缓存中的模型信息
func (s *sModel) GetCacheModel(ctx context.Context, id string) (*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetCacheModel time: %d", gtime.TimestampMilli()-now)
	}()

	models, err := s.GetCacheList(ctx, id)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(models) == 0 {
		return nil, errors.New("model is nil")
	}

	return models[0], nil
}

// 更新缓存中的模型列表
func (s *sModel) UpdateCacheModel(ctx context.Context, oldData *entity.Model, newData *entity.Model) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel UpdateCacheModel time: %d", gtime.TimestampMilli()-now)
	}()

	if err := s.SaveCache(ctx, &model.Model{
		Id:                   newData.Id,
		Corp:                 newData.Corp,
		Name:                 newData.Name,
		Model:                newData.Model,
		Type:                 newData.Type,
		BaseUrl:              newData.BaseUrl,
		Path:                 newData.Path,
		IsEnablePresetConfig: newData.IsEnablePresetConfig,
		PresetConfig:         newData.PresetConfig,
		TextQuota:            newData.TextQuota,
		ImageQuota:           newData.ImageQuota,
		AudioQuota:           newData.AudioQuota,
		MultimodalQuota:      newData.MultimodalQuota,
		RealtimeQuota:        newData.RealtimeQuota,
		MultimodalAudioQuota: newData.MultimodalAudioQuota,
		MidjourneyQuotas:     newData.MidjourneyQuotas,
		DataFormat:           newData.DataFormat,
		IsPublic:             newData.IsPublic,
		IsEnableModelAgent:   newData.IsEnableModelAgent,
		LbStrategy:           newData.LbStrategy,
		ModelAgents:          newData.ModelAgents,
		IsEnableForward:      newData.IsEnableForward,
		ForwardConfig:        newData.ForwardConfig,
		IsEnableFallback:     newData.IsEnableFallback,
		FallbackConfig:       newData.FallbackConfig,
		Status:               newData.Status,
	}); err != nil {
		logger.Error(ctx, err)
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
}

// 获取目标模型
func (s *sModel) GetTargetModel(ctx context.Context, model *model.Model, messages []sdkm.ChatCompletionMessage) (targetModel *model.Model, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetTargetModel time: %d", gtime.TimestampMilli()-now)
	}()

	if !model.IsEnableForward {
		return model, nil
	}

	if model.ForwardConfig.ForwardRule == 1 {

		if targetModel, err = s.GetCacheModel(ctx, model.ForwardConfig.TargetModel); err != nil || targetModel == nil {
			if targetModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.TargetModel); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else if model.ForwardConfig.ForwardRule == 3 {

		contentLength := 0
		for _, message := range messages {
			contentLength += len(gconv.String(message.Content))
		}

		if contentLength < model.ForwardConfig.ContentLength {
			return model, nil
		}

		if targetModel, err = s.GetCacheModel(ctx, model.ForwardConfig.TargetModel); err != nil || targetModel == nil {
			if targetModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.TargetModel); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else {

		prompt := gconv.String(messages[len(messages)-1].Content)

		keywords := model.ForwardConfig.Keywords
		if slices.Contains(model.ForwardConfig.MatchRule, 2) {

			prompt := gstr.ToLower(gstr.TrimAll(prompt))

			for i, keyword := range keywords {

				if gregex.IsMatchString(gstr.ToLower(gstr.TrimAll(keyword)), prompt) {

					if targetModel, err = s.GetCacheModel(ctx, model.ForwardConfig.TargetModels[i]); err != nil || targetModel == nil {
						if targetModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.TargetModels[i]); err != nil {
							logger.Error(ctx, err)
							return nil, err
						}
					}

					if targetModel != nil {
						break
					}
				}
			}
		}

		if targetModel == nil && slices.Contains(model.ForwardConfig.MatchRule, 1) {

			decisionModel, err := s.GetCacheModel(ctx, model.ForwardConfig.DecisionModel)
			if err != nil || decisionModel == nil {
				decisionModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.DecisionModel)
			}

			if decisionModel == nil || decisionModel.Status != 1 {
				return model, nil
			}

			systemPrompt := "You are an emotionless question judgment expert. You will only return the content within the options and will not add any other information. The enumerated content options that can be returned are as follows: [-1%s]"
			systemEnum := ""
			decisionPrompt := "please only return the value that include in enum: [-1%s]; Return the result based on the nature of the conversation: %s Other: -1. The question you need to decide is: '%s'"
			decisionEnum := ""

			for i, keyword := range keywords {
				if i == 0 {
					systemEnum = fmt.Sprintf(",%d", i)
					decisionEnum = fmt.Sprintf("About %s, return %d;", gstr.Replace(keyword, "|", " or about "), i)
				} else {
					systemEnum += fmt.Sprintf(",%d", i)
					decisionEnum += fmt.Sprintf(" About %s, return %d;", gstr.Replace(keyword, "|", " or about "), i)
				}
			}

			systemPrompt = fmt.Sprintf(systemPrompt, systemEnum)
			decisionPrompt = fmt.Sprintf(decisionPrompt, systemEnum, decisionEnum, prompt)

			messages := []sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: systemPrompt,
			}, {
				Role:    consts.ROLE_USER,
				Content: decisionPrompt,
			}}

			response, err := service.Chat().SmartCompletions(ctx, sdkm.ChatCompletionRequest{
				Model:    decisionModel.Model,
				Messages: messages,
			}, decisionModel, nil, nil)

			if err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			logger.Infof(ctx, "sModel GetTargetModel SmartCompletions response: %s", gjson.MustEncodeString(response))

			if len(response.Choices) > 0 {

				if index, err := strconv.Atoi(gconv.String(response.Choices[0].Message.Content)); err == nil {

					if index == -1 {
						return model, nil
					}

					targetModel, err = s.GetCacheModel(ctx, model.ForwardConfig.TargetModels[index])
					if err != nil || targetModel == nil {
						if targetModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.TargetModels[index]); err != nil {
							logger.Error(ctx, err)
							return nil, err
						}
					}
				}
			}
		}
	}

	if targetModel == nil || targetModel.Status != 1 {
		return model, nil
	}

	return s.GetTargetModel(ctx, targetModel, messages)
}

// 获取分组目标模型
func (s *sModel) GetGroupTargetModel(ctx context.Context, group *model.Group, model *model.Model, messages []sdkm.ChatCompletionMessage) (targetModel *model.Model, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetGroupTargetModel time: %d", gtime.TimestampMilli()-now)
	}()

	if !group.IsEnableForward {
		return model, nil
	}

	if group.ForwardConfig.ForwardRule == 1 {

		if targetModel, err = s.GetCacheModel(ctx, group.ForwardConfig.TargetModel); err != nil || targetModel == nil {
			if targetModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.TargetModel); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else if group.ForwardConfig.ForwardRule == 3 {

		contentLength := 0
		for _, message := range messages {
			contentLength += len(gconv.String(message.Content))
		}

		if contentLength < group.ForwardConfig.ContentLength {
			return model, nil
		}

		if targetModel, err = s.GetCacheModel(ctx, group.ForwardConfig.TargetModel); err != nil || targetModel == nil {
			if targetModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.TargetModel); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else if group.ForwardConfig.ForwardRule == 4 {

		if group.UsedQuota < group.ForwardConfig.UsedQuota {
			return model, nil
		}

		if targetModel, err = s.GetCacheModel(ctx, group.ForwardConfig.TargetModel); err != nil || targetModel == nil {
			if targetModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.TargetModel); err != nil {
				logger.Error(ctx, err)
				return nil, err
			}
		}

	} else {

		prompt := gconv.String(messages[len(messages)-1].Content)

		keywords := group.ForwardConfig.Keywords
		if slices.Contains(group.ForwardConfig.MatchRule, 2) {

			prompt := gstr.ToLower(gstr.TrimAll(prompt))

			for i, keyword := range keywords {

				if gregex.IsMatchString(gstr.ToLower(gstr.TrimAll(keyword)), prompt) {

					if targetModel, err = s.GetCacheModel(ctx, group.ForwardConfig.TargetModels[i]); err != nil || targetModel == nil {
						if targetModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.TargetModels[i]); err != nil {
							logger.Error(ctx, err)
							return nil, err
						}
					}

					if targetModel != nil {
						break
					}
				}
			}
		}

		if targetModel == nil && slices.Contains(group.ForwardConfig.MatchRule, 1) {

			decisionModel, err := s.GetCacheModel(ctx, group.ForwardConfig.DecisionModel)
			if err != nil || decisionModel == nil {
				decisionModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.DecisionModel)
			}

			if decisionModel == nil || decisionModel.Status != 1 {
				return model, nil
			}

			systemPrompt := "You are an emotionless question judgment expert. You will only return the content within the options and will not add any other information. The enumerated content options that can be returned are as follows: [-1%s]"
			systemEnum := ""
			decisionPrompt := "please only return the value that include in enum: [-1%s]; Return the result based on the nature of the conversation: %s Other: -1. The question you need to decide is: '%s'"
			decisionEnum := ""

			for i, keyword := range keywords {
				if i == 0 {
					systemEnum = fmt.Sprintf(",%d", i)
					decisionEnum = fmt.Sprintf("About %s, return %d;", gstr.Replace(keyword, "|", " or about "), i)
				} else {
					systemEnum += fmt.Sprintf(",%d", i)
					decisionEnum += fmt.Sprintf(" About %s, return %d;", gstr.Replace(keyword, "|", " or about "), i)
				}
			}

			systemPrompt = fmt.Sprintf(systemPrompt, systemEnum)
			decisionPrompt = fmt.Sprintf(decisionPrompt, systemEnum, decisionEnum, prompt)

			messages := []sdkm.ChatCompletionMessage{{
				Role:    consts.ROLE_SYSTEM,
				Content: systemPrompt,
			}, {
				Role:    consts.ROLE_USER,
				Content: decisionPrompt,
			}}

			response, err := service.Chat().SmartCompletions(ctx, sdkm.ChatCompletionRequest{
				Model:    decisionModel.Model,
				Messages: messages,
			}, decisionModel, nil, nil)

			if err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			logger.Infof(ctx, "sModel GetGroupTargetModel SmartCompletions response: %s", gjson.MustEncodeString(response))

			if len(response.Choices) > 0 {

				if index, err := strconv.Atoi(gconv.String(response.Choices[0].Message.Content)); err == nil {

					if index == -1 {
						return model, nil
					}

					targetModel, err = s.GetCacheModel(ctx, group.ForwardConfig.TargetModels[index])
					if err != nil || targetModel == nil {
						if targetModel, err = s.GetModelAndSaveCache(ctx, group.ForwardConfig.TargetModels[index]); err != nil {
							logger.Error(ctx, err)
							return nil, err
						}
					}
				}
			}
		}
	}

	if targetModel == nil || targetModel.Status != 1 {
		return model, nil
	}

	return s.GetTargetModel(ctx, targetModel, messages)
}

// 获取后备模型
func (s *sModel) GetFallbackModel(ctx context.Context, model *model.Model) (fallbackModel *model.Model, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetFallbackModel time: %d", gtime.TimestampMilli()-now)
	}()

	if fallbackModel, err = s.GetCacheModel(ctx, model.FallbackConfig.Model); err != nil || fallbackModel == nil {
		if fallbackModel, err = s.GetModelAndSaveCache(ctx, model.FallbackConfig.Model); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	if fallbackModel.Status != 1 {
		err = errors.ERR_MODEL_HAS_BEEN_DISABLED
		logger.Error(ctx, err)
		return nil, err
	}

	return fallbackModel, nil
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
	case consts.ACTION_CREATE, consts.ACTION_UPDATE, consts.ACTION_STATUS:

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

	case consts.ACTION_DELETE:

		if err := gjson.Unmarshal(gjson.MustEncode(message.OldData), &newData); err != nil {
			logger.Error(ctx, err)
			return err
		}

		s.RemoveCacheModel(ctx, newData.Id)
	}

	return nil
}
