package batch

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/model"
	v1 "github.com/iimeta/fastapi/api/batch/v1"
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

type sBatch struct{}

func init() {
	service.RegisterBatch(New())
}

func New() service.IBatch {
	return &sBatch{}
}

// Create
func (s *sBatch) Create(ctx context.Context, params *v1.CreateReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.BatchResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBatch Create time: %d", gtime.TimestampMilli()-now)
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
					Action:       consts.ACTION_CREATE,
					IsBatch:      true,
					BatchId:      response.Id,
					FileId:       params.InputFileId,
					RequestData:  gconv.Map(params.BatchCreateRequest),
					ResponseData: gconv.Map(response.ResponseBytes),
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

	response, err = common.NewAdapter(ctx, mak, false).BatchCreate(ctx, params.BatchCreateRequest)
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
							return s.Create(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Create(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Create(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// List
func (s *sBatch) List(ctx context.Context, params *v1.ListReq) (response smodel.BatchListResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBatch List time: %d", gtime.TimestampMilli()-now)
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
					IsBatch:      true,
					RequestData:  gconv.Map(params.BatchListRequest),
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

	if limit > 1000 {
		err = errors.NewError(404, "integer_above_max_value", fmt.Sprintf("Invalid 'limit': integer above maximum value. Expected a value <= 1000, but got %d instead.", params.Limit), "invalid_request_error", "limit")
		return response, err
	} else if limit == 0 {
		limit = 1000
	}

	filter := bson.M{
		"creator":    service.Session().GetSecretKey(ctx),
		"status":     bson.M{"$nin": []string{"deleted", "expired"}},
		"created_at": bson.M{"$gt": time.Now().Add(-720 * time.Hour).UnixMilli()},
	}

	if params.After != "" {

		taskBatch, err := dao.TaskBatch.FindOne(ctx, bson.M{"batch_id": params.After, "creator": service.Session().GetSecretKey(ctx)})
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = errors.NewError(404, "invalid_request_error", "No batch found with id '"+params.After+"'.", "invalid_request_error", nil)
			}
			logger.Error(ctx, err)
			return response, err
		}

		filter["created_at"] = bson.M{"$lte": taskBatch.CreatedAt}

		filter["_id"] = bson.M{"$ne": taskBatch.Id}
	}

	sort := "-created_at"

	paging := &db.Paging{
		Page:     1,
		PageSize: limit,
	}

	results, err := dao.TaskBatch.FindByPage(ctx, paging, filter, &dao.FindOptions{SortFields: []string{sort}})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if len(results) == 0 {
		response = smodel.BatchListResponse{
			Object: "list",
			Data:   make([]any, 0),
		}
		return response, nil
	}

	mak.Model = results[0].Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.BatchListResponse{
		Object:  "list",
		FirstId: &results[0].BatchId,
		LastId:  &results[len(results)-1].BatchId,
		HasMore: paging.PageCount > 1,
	}

	for _, result := range results {
		response.Data = append(response.Data, result.ResponseData)
	}

	return response, nil
}

// Retrieve
func (s *sBatch) Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.BatchResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBatch Retrieve time: %d", gtime.TimestampMilli()-now)
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
					IsBatch:      true,
					BatchId:      params.BatchId,
					RequestData:  gconv.Map(params.BatchRetrieveRequest),
					ResponseData: gconv.Map(response.ResponseBytes),
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

	taskBatch, err := dao.TaskBatch.FindOne(ctx, bson.M{"batch_id": params.BatchId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "No batch found with id '"+params.BatchId+"'.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskBatch.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response.ResponseBytes = gconv.Bytes(taskBatch.ResponseData)

	return response, nil
}

// Cancel
func (s *sBatch) Cancel(ctx context.Context, params *v1.CancelReq) (response smodel.BatchResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBatch Cancel time: %d", gtime.TimestampMilli()-now)
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
					Action:       consts.ACTION_CANCEL,
					IsBatch:      true,
					BatchId:      params.BatchId,
					RequestData:  gconv.Map(params.BatchCancelRequest),
					ResponseData: gconv.Map(response.ResponseBytes),
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

	taskBatch, err := dao.TaskBatch.FindOne(ctx, bson.M{"batch_id": params.BatchId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "No batch found with id '"+params.BatchId+"'.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskBatch.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if err := dao.TaskBatch.UpdateById(ctx, taskBatch.Id, bson.M{"status": "cancelling"}); err != nil {
		logger.Error(ctx, err)
	}

	response.ResponseBytes = gconv.Bytes(taskBatch.ResponseData)

	return response, nil
}
