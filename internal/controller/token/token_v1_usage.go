package token

import (
	"context"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/service"

	"github.com/iimeta/fastapi/api/token/v1"
)

func (c *ControllerV1) Usage(ctx context.Context, req *v1.UsageReq) (res *v1.UsageRes, err error) {

	usageCount, err := service.Common().GetUsageCount(ctx)
	if err != nil {
		return nil, err
	}

	usedTokens, err := service.Common().GetUsedTokens(ctx)
	if err != nil {
		return nil, err
	}

	totalTokens, err := service.Common().GetTotalTokens(ctx)
	if err != nil {
		return nil, err
	}

	res = &v1.UsageRes{
		UsageRes: &model.UsageRes{
			UsageCount:  usageCount,
			UsedTokens:  usedTokens,
			TotalTokens: totalTokens,
		},
	}

	return
}
