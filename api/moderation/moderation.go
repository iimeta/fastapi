// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package moderation

import (
	"context"

	"github.com/iimeta/fastapi/v2/api/moderation/v1"
)

type IModerationV1 interface {
	Moderations(ctx context.Context, req *v1.ModerationsReq) (res *v1.ModerationsRes, err error)
}
