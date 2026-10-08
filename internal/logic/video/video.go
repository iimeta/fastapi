package video

import (
	"context"
	"fmt"
	"time"

	"io"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
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
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
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
			Endpoint:           consts.ENDPOINT_VIDEO_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo      *mcommon.Retry
		videoId        string
		requestData    map[string]any
		inputFilePaths []string
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:         consts.ACTION_CREATE,
					IsAsync:        true,
					VideoId:        videoId,
					Prompt:         params.Prompt,
					Seconds:        gconv.Int(params.Seconds),
					Size:           params.Size,
					RequestData:    requestData,
					InputFilePaths: inputFilePaths,
					ResponseData:   util.ConvToMap(response),
					Error:          err,
					RetryInfo:      retryInfo,
					TotalTime:      response.TotalTime,
					InternalTime:   internalTime,
					EnterTime:      enterTime,
				}

				if params.InputReference != nil {
					afterHandler.VideoMode = "has_video_input"
				} else {
					afterHandler.VideoMode = "no_video_input"
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

	videoId = "video_" + gtrace.GetTraceID(ctx)

	requestData, inputFilePaths, err = persistVideoCreateInput(ctx, params)
	if err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	response = smodel.VideoJobResponse{
		Id:        videoId,
		Object:    "video",
		Model:     mak.ReqModel.Name,
		Status:    "queued",
		Progress:  0,
		CreatedAt: time.Now().Unix(),
		Prompt:    params.Prompt,
		Seconds:   params.Seconds,
		Size:      params.Size,
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
			Endpoint:           consts.ENDPOINT_VIDEO_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo   *mcommon.Retry
		videoId     string
		requestData map[string]any
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:             consts.ACTION_REMIX,
					IsAsync:            true,
					VideoId:            videoId,
					RemixedFromVideoId: params.VideoId,
					RequestData:        requestData,
					ResponseData:       util.ConvToMap(response),
					Error:              err,
					RetryInfo:          retryInfo,
					TotalTime:          response.TotalTime,
					InternalTime:       internalTime,
					EnterTime:          enterTime,
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	origin, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": params.VideoId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+params.VideoId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return response, err
	}

	mak.Model = origin.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	videoId = "video_" + gtrace.GetTraceID(ctx)
	requestData = util.ConvToMap(g.RequestFromCtx(ctx).GetBody())
	if requestData == nil {
		requestData = map[string]any{}
	}

	if origin.JobId != "" && origin.JobId != "-" {
		requestData["upstream_video_id"] = origin.JobId
	} else {
		requestData["upstream_video_id"] = origin.VideoId
	}

	response = smodel.VideoJobResponse{
		Id:                 videoId,
		Object:             "video",
		Model:              mak.ReqModel.Name,
		Status:             "queued",
		Progress:           0,
		CreatedAt:          time.Now().Unix(),
		Prompt:             params.Prompt,
		RemixedFromVideoId: &params.VideoId,
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

		// 失败任务的错误信息按 errors.ShieldError 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
		if result.Error != nil {
			result.Error.Code, result.Error.Message = errors.ShieldError(ctx, result.Error.Code, result.Error.Message)
		}

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

		if result.VideoUrl != "" {
			videoJobResponse.VideoUrl = common.ResolveVideoUrl(result.VideoUrl)
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

	// 失败任务的错误信息按 errors.ShieldError 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
	if taskVideo.Error != nil {
		taskVideo.Error.Code, taskVideo.Error.Message = errors.ShieldError(ctx, taskVideo.Error.Code, taskVideo.Error.Message)
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

	if taskVideo.VideoUrl != "" {
		response.VideoUrl = common.ResolveVideoUrl(taskVideo.VideoUrl)
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

	// 失败任务的错误信息按 errors.ShieldError 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
	if taskVideo.Error != nil {
		taskVideo.Error.Code, taskVideo.Error.Message = errors.ShieldError(ctx, taskVideo.Error.Code, taskVideo.Error.Message)
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

	if response, err = adapter.VideoContent(ctx, smodel.VideoContentRequest{VideoId: pollContentVideoId(taskVideo.VideoId, taskVideo.JobId)}); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	return response, nil
}

func pollContentVideoId(videoId, jobId string) string {
	if jobId != "" && jobId != "-" {
		return jobId
	}
	return videoId
}

func persistVideoCreateInput(ctx context.Context, params *v1.CreateReq) (requestData map[string]any, inputFilePaths []string, err error) {

	requestData = util.ConvToMap(g.RequestFromCtx(ctx).GetBody())
	if requestData == nil {
		requestData = map[string]any{}
	}
	delete(requestData, "input_reference")

	if params.InputReference == nil {
		return requestData, nil, nil
	}

	storageDir := "./resource/public/video/input/"
	if config.Cfg.VideoTask != nil && config.Cfg.VideoTask.StorageDir != "" {
		storageDir = config.Cfg.VideoTask.StorageDir
		if !gstr.HasSuffix(storageDir, "/") {
			storageDir += "/"
		}
		storageDir += "input/"
	}

	if err = gfile.Mkdir(storageDir); err != nil {
		return nil, nil, err
	}

	src, err := params.InputReference.Open()
	if err != nil {
		return nil, nil, err
	}
	defer src.Close()

	content, err := io.ReadAll(src)
	if err != nil {
		return nil, nil, err
	}

	filename := gtrace.GetTraceID(ctx) + "_" + gfile.Basename(params.InputReference.Filename)
	filePath := storageDir + filename
	if err = gfile.PutBytes(filePath, content); err != nil {
		return nil, nil, err
	}

	publicUrl := filename
	if gstr.HasPrefix(storageDir, "./resource/public/") {
		publicUrl = "/public/" + gstr.TrimLeftStr(storageDir, "./resource/public/") + filename
	}

	if config.Cfg.VideoTask != nil && config.Cfg.VideoTask.StorageBaseUrl != "" {
		base := config.Cfg.VideoTask.StorageBaseUrl
		if gstr.HasSuffix(base, "/") {
			publicUrl = base + gstr.TrimLeftStr(publicUrl, "/")
		} else if !gstr.HasPrefix(publicUrl, "/") {
			publicUrl = base + "/" + publicUrl
		} else {
			publicUrl = base + publicUrl
		}
	}

	if _, ok := requestData["input_reference"]; !ok {
		requestData["input_reference"] = publicUrl
	}

	return requestData, []string{filePath}, nil
}
