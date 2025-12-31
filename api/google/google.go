// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package google

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/google/v1"
)

type IGoogleV1 interface {
	Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error)
}
