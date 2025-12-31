package dashboard

import (
	"context"

	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/v2/api/dashboard/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func (c *ControllerV1) Models(ctx context.Context, req *v1.ModelsReq) (res *v1.ModelsRes, err error) {

	modelIds := make([]string, 0)

	if len(service.Session().GetUser(ctx).Groups) > 0 {
		ids, err := service.Group().GetModelIds(ctx, service.Session().GetUser(ctx).Groups...)
		if err != nil {
			return nil, err
		}
		modelIds = append(modelIds, ids...)
	}

	if app, err := service.App().GetCache(ctx, service.Session().GetAppId(ctx)); err != nil {
		return nil, err
	} else if len(app.Models) > 0 {
		modelIds = app.Models
	} else if app.IsBindGroup {
		if modelIds, err = service.Group().GetModelIds(ctx, app.Group); err != nil {
			return nil, err
		}
	} else if appKey, err := service.AppKey().GetCache(ctx, service.Session().GetSecretKey(ctx)); err != nil {
		return nil, err
	} else if len(appKey.Models) > 0 {
		modelIds = appKey.Models
	} else if appKey.IsBindGroup {
		if modelIds, err = service.Group().GetModelIds(ctx, appKey.Group); err != nil {
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

				provider, err := service.Provider().GetCache(ctx, m.ProviderId)
				if err != nil {
					return nil, err
				}

				modelsData := model.DashboardModelsData{
					Id:      m.Model,
					Object:  "model",
					OwnedBy: gstr.ToLower(provider.Code),
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
						Provider: provider.Name,
						Code:     provider.Code,
						Model:    m.Model,
						Type:     m.Type,
						Pricing:  m.Pricing,
						Remark:   m.Remark,
					}
				}

				modelsRes.Data = append(modelsRes.Data, modelsData)
			}
		}
	}

	g.RequestFromCtx(ctx).Response.WriteJson(modelsRes)

	return
}
