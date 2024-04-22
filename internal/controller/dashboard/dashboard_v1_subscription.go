package dashboard

import (
	"context"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/iimeta/fastapi/internal/service"

	"github.com/iimeta/fastapi/api/dashboard/v1"
)

func (c *ControllerV1) Subscription(ctx context.Context, req *v1.SubscriptionReq) (res *v1.SubscriptionRes, err error) {

	subscription, err := service.Dashboard().Subscription(ctx)
	if err != nil {
		return nil, err
	}

	g.RequestFromCtx(ctx).Response.WriteJson(subscription)

	return
}
