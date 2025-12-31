// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	v1 "github.com/iimeta/fastapi/v2/api/batch/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IBatch interface {
		// Create
		Create(ctx context.Context, params *v1.CreateReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.BatchResponse, err error)
		// List
		List(ctx context.Context, params *v1.ListReq) (response smodel.BatchListResponse, err error)
		// Retrieve
		Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.BatchResponse, err error)
		// Cancel
		Cancel(ctx context.Context, params *v1.CancelReq) (response smodel.BatchResponse, err error)
	}
)

var (
	localBatch IBatch
)

func Batch() IBatch {
	if localBatch == nil {
		panic("implement not found for interface IBatch, forgot register?")
	}
	return localBatch
}

func RegisterBatch(i IBatch) {
	localBatch = i
}
