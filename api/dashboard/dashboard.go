// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package dashboard

import (
	"context"

	"github.com/iimeta/fastapi/api/dashboard/v1"
)

type IDashboardV1 interface {
	Subscription(ctx context.Context, req *v1.SubscriptionReq) (res *v1.SubscriptionRes, err error)
	Usage(ctx context.Context, req *v1.UsageReq) (res *v1.UsageRes, err error)
}
