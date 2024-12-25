// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	ICore interface {
		// 刷新缓存
		Refresh(ctx context.Context) error
	}
)

var (
	localCore ICore
)

func Core() ICore {
	if localCore == nil {
		panic("implement not found for interface ICore, forgot register?")
	}
	return localCore
}

func RegisterCore(i ICore) {
	localCore = i
}
