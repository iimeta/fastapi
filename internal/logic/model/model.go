package model

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gregex"
	"github.com/gogf/gf/v2/text/gstr"
	sdkm "github.com/iimeta/fastapi-sdk/model"
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

	detail := &model.Model{
		Id:                 result.Id,
		Corp:               result.Corp,
		Name:               result.Name,
		Model:              result.Model,
		Type:               result.Type,
		BaseUrl:            result.BaseUrl,
		Path:               result.Path,
		Prompt:             result.Prompt,
		BillingMethod:      result.BillingMethod,
		PromptRatio:        result.PromptRatio,
		CompletionRatio:    result.CompletionRatio,
		FixedQuota:         result.FixedQuota,
		DataFormat:         result.DataFormat,
		IsPublic:           result.IsPublic,
		IsEnableModelAgent: result.IsEnableModelAgent,
		ModelAgents:        result.ModelAgents,
		IsEnableForward:    result.IsEnableForward,
		IsEnableFallback:   result.IsEnableFallback,
		Remark:             result.Remark,
		Status:             result.Status,
	}

	if result.ForwardConfig != nil {
		detail.ForwardConfig = &model.ForwardConfig{
			ForwardRule:   result.ForwardConfig.ForwardRule,
			MatchRule:     result.ForwardConfig.MatchRule,
			TargetModel:   result.ForwardConfig.TargetModel,
			DecisionModel: result.ForwardConfig.DecisionModel,
			Keywords:      result.ForwardConfig.Keywords,
			TargetModels:  result.ForwardConfig.TargetModels,
		}
	}

	if result.FallbackConfig != nil {
		detail.FallbackConfig = &model.FallbackConfig{
			FallbackModel: result.FallbackConfig.FallbackModel,
		}
	}

	return detail, nil
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

	detail := &model.Model{
		Id:                 result.Id,
		Corp:               result.Corp,
		Name:               result.Name,
		Model:              result.Model,
		Type:               result.Type,
		BaseUrl:            result.BaseUrl,
		Path:               result.Path,
		Prompt:             result.Prompt,
		BillingMethod:      result.BillingMethod,
		PromptRatio:        result.PromptRatio,
		CompletionRatio:    result.CompletionRatio,
		FixedQuota:         result.FixedQuota,
		DataFormat:         result.DataFormat,
		IsPublic:           result.IsPublic,
		IsEnableModelAgent: result.IsEnableModelAgent,
		ModelAgents:        result.ModelAgents,
		IsEnableForward:    result.IsEnableForward,
		IsEnableFallback:   result.IsEnableFallback,
		Remark:             result.Remark,
		Status:             result.Status,
	}

	if result.ForwardConfig != nil {
		detail.ForwardConfig = &model.ForwardConfig{
			ForwardRule:   result.ForwardConfig.ForwardRule,
			MatchRule:     result.ForwardConfig.MatchRule,
			TargetModel:   result.ForwardConfig.TargetModel,
			DecisionModel: result.ForwardConfig.DecisionModel,
			Keywords:      result.ForwardConfig.Keywords,
			TargetModels:  result.ForwardConfig.TargetModels,
		}
	}

	if result.FallbackConfig != nil {
		detail.FallbackConfig = &model.FallbackConfig{
			FallbackModel: result.FallbackConfig.FallbackModel,
		}
	}

	return detail, nil
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

		models, err = s.GetCacheList(ctx, key.Models...)
		if err != nil || len(models) != len(key.Models) {
			if models, err = s.GetModelListAndSaveCacheList(ctx, key.Models); err != nil {
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

		models, err = s.GetCacheList(ctx, app.Models...)
		if err != nil || len(models) != len(app.Models) {
			if models, err = s.GetModelListAndSaveCacheList(ctx, app.Models); err != nil {
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

		m := &model.Model{
			Id:                 result.Id,
			Corp:               result.Corp,
			Name:               result.Name,
			Model:              result.Model,
			Type:               result.Type,
			BaseUrl:            result.BaseUrl,
			Path:               result.Path,
			Prompt:             result.Prompt,
			BillingMethod:      result.BillingMethod,
			PromptRatio:        result.PromptRatio,
			CompletionRatio:    result.CompletionRatio,
			FixedQuota:         result.FixedQuota,
			DataFormat:         result.DataFormat,
			IsPublic:           result.IsPublic,
			IsEnableModelAgent: result.IsEnableModelAgent,
			ModelAgents:        result.ModelAgents,
			IsEnableForward:    result.IsEnableForward,
			IsEnableFallback:   result.IsEnableFallback,
			Remark:             result.Remark,
			Status:             result.Status,
		}

		if result.ForwardConfig != nil {
			m.ForwardConfig = &model.ForwardConfig{
				ForwardRule:   result.ForwardConfig.ForwardRule,
				MatchRule:     result.ForwardConfig.MatchRule,
				TargetModel:   result.ForwardConfig.TargetModel,
				DecisionModel: result.ForwardConfig.DecisionModel,
				Keywords:      result.ForwardConfig.Keywords,
				TargetModels:  result.ForwardConfig.TargetModels,
			}
		}

		if result.FallbackConfig != nil {
			m.FallbackConfig = &model.FallbackConfig{
				FallbackModel: result.FallbackConfig.FallbackModel,
			}
		}

		items = append(items, m)
	}

	return items, nil
}

// 全部模型列表
func (s *sModel) ListAll(ctx context.Context) ([]*model.Model, error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel ListAll time: %d", gtime.TimestampMilli()-now)
	}()

	filter := bson.M{
		"status": 1,
	}

	results, err := dao.Model.Find(ctx, filter, "-updated_at")
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	items := make([]*model.Model, 0)
	for _, result := range results {

		m := &model.Model{
			Id:                 result.Id,
			Corp:               result.Corp,
			Name:               result.Name,
			Model:              result.Model,
			Type:               result.Type,
			BaseUrl:            result.BaseUrl,
			Path:               result.Path,
			Prompt:             result.Prompt,
			BillingMethod:      result.BillingMethod,
			PromptRatio:        result.PromptRatio,
			CompletionRatio:    result.CompletionRatio,
			FixedQuota:         result.FixedQuota,
			DataFormat:         result.DataFormat,
			IsPublic:           result.IsPublic,
			IsEnableModelAgent: result.IsEnableModelAgent,
			ModelAgents:        result.ModelAgents,
			IsEnableForward:    result.IsEnableForward,
			IsEnableFallback:   result.IsEnableFallback,
			Remark:             result.Remark,
			Status:             result.Status,
		}

		if result.ForwardConfig != nil {
			m.ForwardConfig = &model.ForwardConfig{
				ForwardRule:   result.ForwardConfig.ForwardRule,
				MatchRule:     result.ForwardConfig.MatchRule,
				TargetModel:   result.ForwardConfig.TargetModel,
				DecisionModel: result.ForwardConfig.DecisionModel,
				Keywords:      result.ForwardConfig.Keywords,
				TargetModels:  result.ForwardConfig.TargetModels,
			}
		}

		if result.FallbackConfig != nil {
			m.FallbackConfig = &model.FallbackConfig{
				FallbackModel: result.FallbackConfig.FallbackModel,
			}
		}

		items = append(items, m)
	}

	return items, nil
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

	m := &model.Model{
		Id:                 newData.Id,
		Corp:               newData.Corp,
		Name:               newData.Name,
		Model:              newData.Model,
		Type:               newData.Type,
		BaseUrl:            newData.BaseUrl,
		Path:               newData.Path,
		Prompt:             newData.Prompt,
		BillingMethod:      newData.BillingMethod,
		PromptRatio:        newData.PromptRatio,
		CompletionRatio:    newData.CompletionRatio,
		FixedQuota:         newData.FixedQuota,
		DataFormat:         newData.DataFormat,
		IsPublic:           newData.IsPublic,
		IsEnableModelAgent: newData.IsEnableModelAgent,
		ModelAgents:        newData.ModelAgents,
		IsEnableForward:    newData.IsEnableForward,
		IsEnableFallback:   newData.IsEnableFallback,
		Status:             newData.Status,
	}

	if newData.ForwardConfig != nil {
		m.ForwardConfig = &model.ForwardConfig{
			ForwardRule:   newData.ForwardConfig.ForwardRule,
			MatchRule:     newData.ForwardConfig.MatchRule,
			TargetModel:   newData.ForwardConfig.TargetModel,
			DecisionModel: newData.ForwardConfig.DecisionModel,
			Keywords:      newData.ForwardConfig.Keywords,
			TargetModels:  newData.ForwardConfig.TargetModels,
		}
	}

	if newData.FallbackConfig != nil {
		m.FallbackConfig = &model.FallbackConfig{
			FallbackModel: newData.FallbackConfig.FallbackModel,
		}
	}

	if err := s.SaveCache(ctx, m); err != nil {
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

	if _, err := redis.HDel(ctx, consts.API_MODELS_KEY, id); err != nil {
		logger.Error(ctx, err)
	}
}

// 获取目标模型
func (s *sModel) GetTargetModel(ctx context.Context, model *model.Model, prompt string) (targetModel *model.Model, err error) {

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

	} else {

		keywords := model.ForwardConfig.Keywords
		if slices.Contains(model.ForwardConfig.MatchRule, 2) {

			for i, keyword := range keywords {

				if gregex.IsMatchString(gstr.ToLower(gstr.TrimAll(keyword)), gstr.ToLower(gstr.TrimAll(prompt))) {

					if targetModel, err = s.GetCacheModel(ctx, model.ForwardConfig.TargetModels[i]); err != nil || targetModel == nil {
						if targetModel, err = s.GetModelAndSaveCache(ctx, model.ForwardConfig.TargetModels[i]); err != nil {
							logger.Error(ctx, err)
							return nil, err
						}
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
					decisionEnum = fmt.Sprintf("About %s, return %d;", gstr.Replace(keyword, "|", " or "), i)
				} else {
					systemEnum += fmt.Sprintf(",%d", i)
					decisionEnum += fmt.Sprintf(" About %s, return %d;", gstr.Replace(keyword, "|", " or "), i)
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
			}, decisionModel, nil)

			if err != nil {
				logger.Error(ctx, err)
				return nil, err
			}

			logger.Infof(ctx, "sModel GetTargetModel SmartCompletions response: %s", gjson.MustEncodeString(response))

			if len(response.Choices) > 0 {

				if index, err := strconv.Atoi(response.Choices[0].Message.Content); err == nil {

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

	return s.GetTargetModel(ctx, targetModel, prompt)
}

// 获取后备模型
func (s *sModel) GetFallbackModel(ctx context.Context, model *model.Model) (fallbackModel *model.Model, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sModel GetFallbackModel time: %d", gtime.TimestampMilli()-now)
	}()

	if fallbackModel, err = s.GetCacheModel(ctx, model.FallbackConfig.FallbackModel); err != nil || fallbackModel == nil {
		if fallbackModel, err = s.GetModelAndSaveCache(ctx, model.FallbackConfig.FallbackModel); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
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
