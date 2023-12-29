// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package token

import (
	"context"

	"github.com/iimeta/fastapi/api/token/v1"
)

type ITokenV1 interface {
	Usage(ctx context.Context, req *v1.UsageReq) (res *v1.UsageRes, err error)
}
