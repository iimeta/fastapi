package dashboard

import (
	"context"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/api/dashboard/v1"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
)

func (c *ControllerV1) Models(ctx context.Context, req *v1.ModelsReq) (res *v1.ModelsRes, err error) {

	modelIds := service.Session().GetUser(ctx).Models

	if len(service.Session().GetUser(ctx).Groups) > 0 {
		ids, err := service.Group().GetGroupsModelIds(ctx, service.Session().GetUser(ctx).Groups...)
		if err != nil {
			return nil, err
		}
		modelIds = append(modelIds, ids...)
	}

	if app, err := service.App().GetCacheApp(ctx, service.Session().GetAppId(ctx)); err != nil {
		logger.Error(ctx, err)
		return nil, err
	} else if len(app.Models) > 0 {
		modelIds = app.Models
	} else if app.IsBindGroup {
		if modelIds, err = service.Group().GetGroupsModelIds(ctx, app.Group); err != nil {
			return nil, err
		}
	} else if appKey, err := service.App().GetCacheAppKey(ctx, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return nil, err
	} else if len(appKey.Models) > 0 {
		modelIds = appKey.Models
	} else if appKey.IsBindGroup {
		if modelIds, err = service.Group().GetGroupsModelIds(ctx, appKey.Group); err != nil {
			return nil, err
		}
	}

	modelsRes := &model.DashboardModelsRes{
		Object: "list",
		Data:   []model.DashboardModelsData{},
	}

	if len(modelIds) > 0 {

		models, err := service.Model().GetCacheList(ctx, modelIds...)
		if err != nil {
			return nil, err
		}

		ids := gset.NewStrSet()
		for _, m := range models {

			if m.Status == 1 && ids.AddIfNotExist(m.Model) {

				corp, err := service.Corp().GetCacheCorp(ctx, m.Corp)
				if err != nil {
					return nil, err
				}

				modelsData := model.DashboardModelsData{
					Id:      m.Model,
					Object:  "model",
					OwnedBy: gstr.ToLower(corp.Code),
					Created: gconv.Int(m.CreatedAt / 1000),
					Root:    m.Model,
					Permission: []model.Permission{{
						Id:                "modelperm-" + m.Model,
						Object:            "model_permission",
						Created:           gconv.Int(m.CreatedAt / 1000),
						AllowCreateEngine: true,
						AllowSampling:     true,
						AllowLogprobs:     true,
						AllowView:         true,
						Organization:      "*",
					}},
				}

				if req.IsFastAPI {
					modelsData.FastAPI = &model.FastAPI{
						Corp:                 corp.Name,
						Code:                 corp.Code,
						Model:                m.Model,
						Type:                 m.Type,
						BaseUrl:              m.BaseUrl,
						Path:                 m.Path,
						TextQuota:            m.TextQuota,
						ImageQuota:           m.ImageQuota,
						AudioQuota:           m.AudioQuota,
						MultimodalQuota:      m.MultimodalQuota,
						RealtimeQuota:        m.RealtimeQuota,
						MultimodalAudioQuota: m.MultimodalAudioQuota,
						MidjourneyQuotas:     m.MidjourneyQuotas,
						Remark:               m.Remark,
					}
				}

				modelsRes.Data = append(modelsRes.Data, modelsData)
			}
		}
	}

	g.RequestFromCtx(ctx).Response.WriteJson(modelsRes)

	return
}
