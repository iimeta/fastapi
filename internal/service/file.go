// ================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// You can delete these comments if you wish manually maintain this interface file.
// ================================================================================

package service

import (
	"context"

	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	v1 "github.com/iimeta/fastapi/v2/api/file/v1"
	"github.com/iimeta/fastapi/v2/internal/model"
)

type (
	IFile interface {
		// Upload
		Upload(ctx context.Context, params *v1.UploadReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.FileResponse, err error)
		// List
		List(ctx context.Context, params *v1.ListReq) (response smodel.FileListResponse, err error)
		// Retrieve
		Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.FileResponse, err error)
		// Delete
		Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.FileResponse, err error)
		// Content
		Content(ctx context.Context, params *v1.ContentReq) (response smodel.FileContentResponse, err error)
	}
)

var (
	localFile IFile
)

func File() IFile {
	if localFile == nil {
		panic("implement not found for interface IFile, forgot register?")
	}
	return localFile
}

func RegisterFile(i IFile) {
	localFile = i
}
