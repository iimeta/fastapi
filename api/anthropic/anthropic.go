// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package anthropic

import (
	"context"

	"github.com/iimeta/fastapi/api/anthropic/v1"
)

type IAnthropicV1 interface {
	Completions(ctx context.Context, req *v1.CompletionsReq) (res *v1.CompletionsRes, err error)
}
