package dashboard

import (
	"context"
	"github.com/gogf/gf/v2/container/gset"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/util/gconv"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"

	"github.com/iimeta/fastapi/api/dashboard/v1"
)

func (c *ControllerV1) Models(ctx context.Context, req *v1.ModelsReq) (res *v1.ModelsRes, err error) {

	models, err := service.Model().GetCacheList(ctx, service.Session().GetUser(ctx).Models...)
	if err != nil {
		return nil, err
	}

	modelsRes := &model.DashboardModelsRes{
		Object: "list",
	}

	ids := gset.NewStrSet()
	for _, m := range models {

		if m.Status == 1 && ids.AddIfNotExist(m.Model) {

			corp, err := service.Corp().GetCacheCorp(ctx, m.Corp)
			if err != nil {
				return nil, err
			}

			modelsRes.Data = append(modelsRes.Data, model.DashboardModelsData{
				Id:      m.Model,
				Object:  "model",
				OwnedBy: "fastapi",
				Created: gconv.Int(m.CreatedAt / 1000),
				FastAPI: &model.FastAPI{
					Corp:            corp.Name,
					Code:            corp.Code,
					Model:           m.Model,
					Type:            m.Type,
					BaseUrl:         m.BaseUrl,
					Path:            m.Path,
					BillingMethod:   m.BillingMethod,
					PromptRatio:     m.PromptRatio,
					CompletionRatio: m.CompletionRatio,
					FixedQuota:      m.FixedQuota,
				},
			})
		}
	}

	g.RequestFromCtx(ctx).Response.WriteJson(modelsRes)

	return
}
