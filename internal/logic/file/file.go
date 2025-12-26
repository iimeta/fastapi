package file

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	sdk "github.com/iimeta/fastapi-sdk"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/options"
	v1 "github.com/iimeta/fastapi/api/file/v1"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/db"
	"github.com/iimeta/fastapi/utility/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type sFile struct{}

func init() {
	service.RegisterFile(New())
}

func New() service.IFile {
	return &sFile{}
}

// Upload
func (s *sFile) Upload(ctx context.Context, params *v1.UploadReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.FileResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile Upload time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_UPLOAD,
					IsFile:       true,
					FileId:       response.Id,
					FileRes:      response,
					RequestData:  gconv.Map(params.FileUploadRequest),
					ResponseData: gconv.Map(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	request := params

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sFile Upload request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	response, err = common.NewAdapter(ctx, mak, false).FileUpload(ctx, request.FileUploadRequest)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {
				if mak.RealModel.IsEnableModelAgent {
					service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())
				} else {
					service.Key().Disabled(ctx, mak.Key, err.Error())
				}
			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.RealModel.IsEnableModelAgent, mak.AgentTotal, mak.KeyTotal, len(retry)) {

				if mak.RealModel.IsEnableFallback {

					if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id {
						if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Upload(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Upload(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
						}
					}
				}

				return response, err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.Upload(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// List
func (s *sFile) List(ctx context.Context, params *v1.ListReq) (response smodel.FileListResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile List time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_LIST,
					IsFile:       true,
					RequestData:  gconv.Map(params.FileListRequest),
					ResponseData: gconv.Map(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	limit := params.Limit

	if limit > 10000 {
		err = errors.NewError(404, "integer_above_max_value", fmt.Sprintf("Invalid 'limit': integer above maximum value. Expected a value <= 10000, but got %d instead.", params.Limit), "invalid_request_error", "limit")
		return response, err
	} else if limit == 0 {
		limit = 1000
	}

	filter := bson.M{
		"creator":    service.Session().GetSecretKey(ctx),
		"status":     bson.M{"$nin": []string{"deleted", "expired"}},
		"created_at": bson.M{"$gt": time.Now().Add(-24 * time.Hour).UnixMilli()},
	}

	if params.Purpose != "" {
		filter["purpose"] = params.Purpose
	}

	if params.After != "" {

		taskFileFilter := bson.M{
			"file_id": params.After,
			"creator": service.Session().GetSecretKey(ctx),
		}

		if params.Purpose != "" {
			taskFileFilter["purpose"] = params.Purpose
		}

		taskFile, err := dao.TaskFile.FindOne(ctx, taskFileFilter)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = errors.NewError(404, "invalid_request_error", "No such File object: "+params.After, "invalid_request_error", "id")
			}
			logger.Error(ctx, err)
			return response, err
		}

		filter["created_at"] = bson.M{"$lte": taskFile.CreatedAt}

		if params.Order == "asc" {
			filter["created_at"] = bson.M{"$gte": taskFile.CreatedAt}
		}

		filter["_id"] = bson.M{"$ne": taskFile.Id}
	}

	sort := "-created_at"
	if params.Order == "asc" {
		sort = "created_at"
	}

	paging := &db.Paging{
		Page:     1,
		PageSize: limit,
	}

	results, err := dao.TaskFile.FindByPage(ctx, paging, filter, &dao.FindOptions{SortFields: []string{sort}})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if len(results) == 0 {
		response = smodel.FileListResponse{
			Object: "list",
			Data:   make([]smodel.FileResponse, 0),
		}
		return response, nil
	}

	mak.Model = results[0].Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.FileListResponse{
		Object:  "list",
		FirstId: &results[0].FileId,
		LastId:  &results[len(results)-1].FileId,
		HasMore: paging.PageCount > 1,
	}

	for _, result := range results {

		fileResponse := smodel.FileResponse{
			Id:        result.FileId,
			Object:    "file",
			Purpose:   result.Purpose,
			Filename:  result.FileName,
			Bytes:     result.Bytes,
			CreatedAt: result.CreatedAt / 1000,
			ExpiresAt: result.ExpiresAt,
			Status:    result.Status,
		}

		response.Data = append(response.Data, fileResponse)
	}

	return response, nil
}

// Retrieve
func (s *sFile) Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.FileResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile Retrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					IsFile:       true,
					RequestData:  gconv.Map(params.FileRetrieveRequest),
					ResponseData: gconv.Map(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskFile, err := dao.TaskFile.FindOne(ctx, bson.M{"file_id": params.FileId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "No such File object: "+params.FileId, "invalid_request_error", "id")
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskFile.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.FileResponse{
		Id:        taskFile.FileId,
		Object:    "file",
		Purpose:   taskFile.Purpose,
		Filename:  taskFile.FileName,
		Bytes:     taskFile.Bytes,
		CreatedAt: taskFile.CreatedAt / 1000,
		ExpiresAt: taskFile.ExpiresAt,
		Status:    taskFile.Status,
	}

	return response, nil
}

// Delete
func (s *sFile) Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.FileResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile Delete time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_DELETE,
					IsFile:       true,
					RequestData:  gconv.Map(params.FileDeleteRequest),
					ResponseData: gconv.Map(response),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskFile, err := dao.TaskFile.FindOne(ctx, bson.M{"file_id": params.FileId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "No such File object: "+params.FileId, "invalid_request_error", "id")
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskFile.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if err := dao.TaskFile.UpdateById(ctx, taskFile.Id, bson.M{"status": "deleted"}); err != nil {
		logger.Error(ctx, err)
	}

	response = smodel.FileResponse{
		Id:        taskFile.FileId,
		Object:    "file",
		Purpose:   taskFile.Purpose,
		Filename:  taskFile.FileName,
		Bytes:     taskFile.Bytes,
		CreatedAt: taskFile.CreatedAt / 1000,
		ExpiresAt: taskFile.ExpiresAt,
		Status:    "deleted",
		Deleted:   true,
	}

	return response, nil
}

// Content
func (s *sFile) Content(ctx context.Context, params *v1.ContentReq) (response smodel.FileContentResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sFile Content time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_CONTENT,
					IsFile:       true,
					RequestData:  gconv.Map(params.FileContentRequest),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskFile, err := dao.TaskFile.FindOne(ctx, bson.M{"file_id": params.FileId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "No such File object: "+params.FileId, "invalid_request_error", "id")
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskFile.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	var adapter sdk.AdapterGroup

	if taskFile.Purpose != "batch_output" {

		logFile, err := dao.LogFile.FindOne(ctx, bson.M{"trace_id": taskFile.TraceId, "status": 1})
		if err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		adapter = sdk.NewAdapter(ctx, &options.AdapterOptions{
			Provider: common.GetProviderCode(ctx, logFile.ModelAgent.ProviderId),
			Model:    logFile.Model,
			Key:      logFile.Key,
			BaseUrl:  logFile.ModelAgent.BaseUrl,
			Path:     logFile.ModelAgent.Path,
			Timeout:  config.Cfg.Base.ShortTimeout * time.Second,
			ProxyUrl: config.Cfg.Http.ProxyUrl,
		})

	} else {

		logBatch, err := dao.LogBatch.FindOne(ctx, bson.M{"trace_id": taskFile.TraceId, "status": 1})
		if err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		adapter = sdk.NewAdapter(ctx, &options.AdapterOptions{
			Provider: common.GetProviderCode(ctx, logBatch.ModelAgent.ProviderId),
			Model:    logBatch.Model,
			Key:      logBatch.Key,
			BaseUrl:  logBatch.ModelAgent.BaseUrl,
			Path:     logBatch.ModelAgent.Path,
			Timeout:  config.Cfg.Base.ShortTimeout * time.Second,
			ProxyUrl: config.Cfg.Http.ProxyUrl,
		})
	}

	if response, err = adapter.FileContent(ctx, smodel.FileContentRequest{FileId: taskFile.FileId}); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	return response, nil
}
