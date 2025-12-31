// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IImage interface {
		// Generations
		Generations(ctx context.Context, data []byte, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error)
		// Edits
		Edits(ctx context.Context, params smodel.ImageEditRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ImageResponse, err error)
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
