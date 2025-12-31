// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	v1 "github.com/iimeta/fastapi/v2/api/video/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IVideo interface {
		// Create
		Create(ctx context.Context, params *v1.CreateReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.VideoJobResponse, err error)
		// Remix
		Remix(ctx context.Context, params *v1.RemixReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.VideoJobResponse, err error)
		// List
		List(ctx context.Context, params *v1.ListReq) (response smodel.VideoListResponse, err error)
		// Retrieve
		Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.VideoJobResponse, err error)
		// Delete
		Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.VideoJobResponse, err error)
		// Content
		Content(ctx context.Context, params *v1.ContentReq) (response smodel.VideoContentResponse, err error)
	}
)

var (
	localVideo IVideo
)

func Video() IVideo {
	if localVideo == nil {
		panic("implement not found for interface IVideo, forgot register?")
	}
	return localVideo
}

func RegisterVideo(i IVideo) {
	localVideo = i
}
