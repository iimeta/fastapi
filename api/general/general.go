// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package general

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/general/v1"
)

type IGeneralV1 interface {
	General(ctx context.Context, req *v1.GeneralReq) (res *v1.GeneralRes, err error)
}
