package video

import (
	"context"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sdk "github.com/iimeta/fastapi-sdk/v2"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi-sdk/v2/options"
	v1 "github.com/iimeta/fastapi/v2/api/video/v1"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/db"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type sVideo struct{}

func init() {
	service.RegisterVideo(New())
}

func New() service.IVideo {
	return &sVideo{}
}

// Create
func (s *sVideo) Create(ctx context.Context, params *v1.CreateReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.VideoJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo Create time: %d", gtime.TimestampMilli()-now)
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
					VideoId:      response.Id,
					Prompt:       params.Prompt,
					Seconds:      gconv.Int(params.Seconds),
					Size:         params.Size,
					RequestData:  util.ConvToMap(params.VideoCreateRequest),
					ResponseData: util.ConvToMap(response),
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
				logger.Infof(ctx, "sVideo Create request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	response, err = common.NewAdapter(ctx, mak, false).VideoCreate(ctx, request.VideoCreateRequest)
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

// Remix
func (s *sVideo) Remix(ctx context.Context, params *v1.RemixReq, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.VideoJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo Remix time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
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
					Action:       consts.ACTION_REMIX,
					VideoId:      response.Id,
					RequestData:  util.ConvToMap(params.VideoRemixRequest),
					ResponseData: util.ConvToMap(response),
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

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.VideoId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.VideoId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response, err = common.NewAdapter(ctx, mak, false).VideoRemix(ctx, params.VideoRemixRequest)
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
							return s.Remix(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Remix(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Remix(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// List
func (s *sVideo) List(ctx context.Context, params *v1.ListReq) (response smodel.VideoListResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo List time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_LIST,
					RequestData:  util.ConvToMap(params.VideoListRequest),
					ResponseData: util.ConvToMap(response),
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
		"created_at": bson.M{"$gt": time.Now().Add(-24 * time.Hour).UnixMilli()},
	}

	if params.After != "" {

		taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.After, "creator": service.Session().GetSecretKey(ctx)})
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.After+"' not found.", "invalid_request_error", nil)
			}
			logger.Error(ctx, err)
			return response, err
		}

		filter["created_at"] = bson.M{"$lte": taskVideo.CreatedAt}

		if params.Order == "asc" {
			filter["created_at"] = bson.M{"$gte": taskVideo.CreatedAt}
		}

		filter["_id"] = bson.M{"$ne": taskVideo.Id}
	}

	sort := "-created_at"
	if params.Order == "asc" {
		sort = "created_at"
	}

	paging := &db.Paging{
		Page:     1,
		PageSize: limit,
	}

	results, err := dao.TaskVideo.FindByPage(ctx, paging, filter, &dao.FindOptions{SortFields: []string{sort}})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if len(results) == 0 {
		response = smodel.VideoListResponse{
			Object: "list",
			Data:   make([]smodel.VideoJobResponse, 0),
		}
		return response, nil
	}

	mak.Model = results[0].Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.VideoListResponse{
		Object:  "list",
		FirstId: &results[0].VideoId,
		LastId:  &results[len(results)-1].VideoId,
		HasMore: paging.PageCount > 1,
	}

	for _, result := range results {

		videoJobResponse := smodel.VideoJobResponse{
			Id:        result.VideoId,
			Object:    "video",
			Model:     result.Model,
			Status:    result.Status,
			Progress:  result.Progress,
			CreatedAt: result.CreatedAt / 1000,
			Size:      fmt.Sprintf("%dx%d", result.Width, result.Height),
			Prompt:    result.Prompt,
			Seconds:   gconv.String(result.Seconds),
			Error:     result.Error,
		}

		if result.CompletedAt != 0 {
			videoJobResponse.CompletedAt = &result.CompletedAt
		}

		if result.ExpiresAt != 0 {
			videoJobResponse.ExpiresAt = &result.ExpiresAt
		}

		if result.RemixedFromVideoId != "" {
			videoJobResponse.RemixedFromVideoId = &result.RemixedFromVideoId
		}

		if config.Cfg.VideoTask.IsEnableStorage && result.VideoUrl != "" {

			if config.Cfg.VideoTask.StorageBaseUrl != "" {
				if gstr.HasSuffix(config.Cfg.VideoTask.StorageBaseUrl, "/") {
					result.VideoUrl = gstr.TrimLeft(result.VideoUrl, "/")
				} else if !gstr.HasPrefix(result.VideoUrl, "/") {
					result.VideoUrl = "/" + result.VideoUrl
				}
			}

			videoJobResponse.VideoUrl = config.Cfg.VideoTask.StorageBaseUrl + result.VideoUrl
		}

		response.Data = append(response.Data, videoJobResponse)
	}

	return response, nil
}

// Retrieve
func (s *sVideo) Retrieve(ctx context.Context, params *v1.RetrieveReq) (response smodel.VideoJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo Retrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					VideoId:      params.VideoId,
					RequestData:  util.ConvToMap(params.VideoRetrieveRequest),
					ResponseData: util.ConvToMap(response),
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

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.VideoId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.VideoId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.VideoJobResponse{
		Id:        taskVideo.VideoId,
		Object:    "video",
		Model:     taskVideo.Model,
		Status:    taskVideo.Status,
		Progress:  taskVideo.Progress,
		CreatedAt: taskVideo.CreatedAt / 1000,
		Size:      fmt.Sprintf("%dx%d", taskVideo.Width, taskVideo.Height),
		Prompt:    taskVideo.Prompt,
		Seconds:   gconv.String(taskVideo.Seconds),
		Error:     taskVideo.Error,
	}

	if taskVideo.CompletedAt != 0 {
		response.CompletedAt = &taskVideo.CompletedAt
	}

	if taskVideo.ExpiresAt != 0 {
		response.ExpiresAt = &taskVideo.ExpiresAt
	}

	if taskVideo.RemixedFromVideoId != "" {
		response.RemixedFromVideoId = &taskVideo.RemixedFromVideoId
	}

	if config.Cfg.VideoTask.IsEnableStorage && taskVideo.VideoUrl != "" {

		if config.Cfg.VideoTask.StorageBaseUrl != "" {
			if gstr.HasSuffix(config.Cfg.VideoTask.StorageBaseUrl, "/") {
				taskVideo.VideoUrl = gstr.TrimLeft(taskVideo.VideoUrl, "/")
			} else if !gstr.HasPrefix(taskVideo.VideoUrl, "/") {
				taskVideo.VideoUrl = "/" + taskVideo.VideoUrl
			}
		}

		response.VideoUrl = config.Cfg.VideoTask.StorageBaseUrl + taskVideo.VideoUrl
	}

	return response, nil
}

// Delete
func (s *sVideo) Delete(ctx context.Context, params *v1.DeleteReq) (response smodel.VideoJobResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo Delete time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		response.TotalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_DELETE,
					VideoId:      params.VideoId,
					RequestData:  util.ConvToMap(params.VideoDeleteRequest),
					ResponseData: util.ConvToMap(response),
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

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.VideoId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.VideoId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if err := dao.TaskVideo.UpdateById(ctx, taskVideo.Id, bson.M{"status": "deleted", "video_url": "", "file_name": "", "file_path": ""}); err != nil {
		logger.Error(ctx, err)
	}

	response = smodel.VideoJobResponse{
		Id:        taskVideo.VideoId,
		Object:    "video.deleted",
		Model:     taskVideo.Model,
		Status:    "deleted",
		Progress:  taskVideo.Progress,
		CreatedAt: taskVideo.CreatedAt / 1000,
		Size:      fmt.Sprintf("%dx%d", taskVideo.Width, taskVideo.Height),
		Prompt:    taskVideo.Prompt,
		Seconds:   gconv.String(taskVideo.Seconds),
		Error:     taskVideo.Error,
		Deleted:   true,
	}

	if taskVideo.CompletedAt != 0 {
		response.CompletedAt = &taskVideo.CompletedAt
	}

	if taskVideo.ExpiresAt != 0 {
		response.ExpiresAt = &taskVideo.ExpiresAt
	}

	if taskVideo.RemixedFromVideoId != "" {
		response.RemixedFromVideoId = &taskVideo.RemixedFromVideoId
	}

	return response, nil
}

// Content
func (s *sVideo) Content(ctx context.Context, params *v1.ContentReq) (response smodel.VideoContentResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVideo Content time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak       = &common.MAK{}
		retryInfo *mcommon.Retry
	)

	defer func() {

		if response.TotalTime == 0 {
			response.TotalTime = gtime.TimestampMilli() - now
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_CONTENT,
					VideoId:      params.VideoId,
					RequestData:  util.ConvToMap(params.VideoContentRequest),
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

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.VideoId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.VideoId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	if taskVideo.Status != "completed" {
		err = errors.NewError(404, "invalid_request_error", "Video is not ready yet, use GET /v1/videos/{video_id} to check status.", "invalid_request_error", nil)
		return response, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if config.Cfg.VideoTask.IsEnableStorage && taskVideo.FilePath != "" {
		if bytes := gfile.GetBytes(taskVideo.FilePath); bytes != nil {
			response = smodel.VideoContentResponse{Data: bytes}
			return response, nil
		}
	}

	logVideo, err := dao.LogVideo.FindOne(ctx, bson.M{"trace_id": taskVideo.TraceId, "status": 1})
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	adapter := sdk.NewAdapter(ctx, &options.AdapterOptions{
		Provider: common.GetProviderCode(ctx, logVideo.ModelAgent.ProviderId),
		Model:    logVideo.Model,
		Key:      logVideo.Key,
		BaseUrl:  logVideo.ModelAgent.BaseUrl,
		Path:     logVideo.ModelAgent.Path,
		Timeout:  config.Cfg.Base.ShortTimeout * time.Second,
		ProxyUrl: config.Cfg.Http.ProxyUrl,
	})

	if response, err = adapter.VideoContent(ctx, smodel.VideoContentRequest{VideoId: taskVideo.VideoId}); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	return response, nil
}
