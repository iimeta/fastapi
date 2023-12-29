// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"
)

type (
	IVip interface {
		CheckUserVipPermissions(ctx context.Context, secretKey, model string) bool
	}
)

var (
	localVip IVip
)

func Vip() IVip {
	if localVip == nil {
		panic("implement not found for interface IVip, forgot register?")
	}
	return localVip
}

func RegisterVip(i IVip) {
	localVip = i
}
