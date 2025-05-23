// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package openai

import (
	"context"

	"github.com/iimeta/fastapi/api/openai/v1"
)

type IOpenaiV1 interface {
	Responses(ctx context.Context, req *v1.ResponsesReq) (res *v1.ResponsesRes, err error)
	ResponsesChatCompletions(ctx context.Context, req *v1.ResponsesChatCompletionsReq) (res *v1.ResponsesChatCompletionsRes, err error)
}
