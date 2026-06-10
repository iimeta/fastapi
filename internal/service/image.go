// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	v1 "github.com/iimeta/fastapi/v2/api/image/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IImage interface {
		// Generations
		Generations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error)
		// GenerationsStream
		GenerationsStream(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
		// Edits
		Edits(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error)
		// EditsStream
		EditsStream(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error)
		// GenerationsAsync
		GenerationsAsync(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageJobResponse, err error)
		// List
		List(ctx context.Context, params *v1.ListReq) (response smodel.ImageListResponse, err error)
		// Retrieve
		Retrieve(ctx context.Context, params smodel.ImageRetrieveRequest) (response smodel.ImageJobResponse, err error)
		// Delete
		Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.ImageJobResponse, err error)
		// Content
		Content(ctx context.Context, params *v1.ContentReq) (response smodel.ImageContentResponse, err error)
	}
)

var (
	localImage IImage
)

func Image() IImage {
	if localImage == nil {
		panic("implement not found for interface IImage, forgot register?")
	}
	return localImage
}

func RegisterImage(i IImage) {
	localImage = i
}
