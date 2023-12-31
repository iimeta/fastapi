// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	"github.com/iimeta/fastapi/internal/model"
)

type (
	IModel interface {
		// 根据model获取模型信息
		GetModel(ctx context.Context, m string) (*model.Model, error)
		// 根据model和secretKey获取模型信息
		GetModelBySecretKey(ctx context.Context, m, secretKey string) (*model.Model, error)
		// 模型列表
		List(ctx context.Context, ids []string) ([]*model.Model, error)
	}
)

var (
	localModel IModel
)

func Model() IModel {
	if localModel == nil {
		panic("implement not found for interface IModel, forgot register?")
	}
	return localModel
}

func RegisterModel(i IModel) {
	localModel = i
}
