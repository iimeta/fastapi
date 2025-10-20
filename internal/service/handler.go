// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

type (
	IHandler interface {
		Init() error
	}
)

var (
	localHandler IHandler
)

func Handler() IHandler {
	if localHandler == nil {
		panic("implement not found for interface IHandler, forgot register?")
	}
	return localHandler
}

func RegisterHandler(i IHandler) {
	localHandler = i
}
