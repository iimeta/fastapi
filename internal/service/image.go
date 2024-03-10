// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/sashabaranov/go-openai"
)

type (
	IImage interface {
		// Generations
		Generations(ctx context.Context, params openai.ImageRequest, retry ...int) (response sdkm.ImageResponse, err error)
		// 保存文生图聊天数据
		SaveImage(ctx context.Context, model *model.Model, key *model.Key, imageReq *openai.ImageRequest, imageRes *model.ImageRes)
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
