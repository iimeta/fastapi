package volcengine

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi-sdk/v2/volcengine"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/db"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type sVolcEngine struct{}

func init() {
	service.RegisterVolcEngine(New())
}

func New() service.IVolcEngine {
	return &sVolcEngine{}
}

// VideoCreate
func (s *sVolcEngine) VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine VideoCreate time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convCreateRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		totalTime int64
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_CREATE,
					VideoMode:    detectVideoMode(params),
					RequestData:  util.ConvToMap(params),
					ResponseData: util.ConvToMap(responseBytes),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if params.Frames != nil && *params.Frames > 0 {
					afterHandler.Seconds = int(math.Ceil(float64(*params.Frames) / 24))
				} else if params.Duration != nil && *params.Duration > 0 {
					afterHandler.Seconds = *params.Duration
				}

				// 解析响应获取 VideoId
				if responseBytes != nil {
					var res smodel.VolcVideoTaskRes
					if e := json.Unmarshal(responseBytes, &res); e == nil {
						afterHandler.VideoId = res.Id
						if res.Duration != nil && afterHandler.Seconds == 0 {
							afterHandler.Seconds = *res.Duration
						}
					}
				}

				common.AfterHandler(ctx, mak, afterHandler)

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	body := request.GetBody()

	responseBytes, err = common.NewAdapterOfficial(ctx, mak, false).VideoCreateOfficial(ctx, body)
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

				if service.Session().GetModelAgentBillingMethod(ctx) == 2 && slices.Contains(mak.RealModel.Pricing.BillingMethods, 1) {
					service.Session().SaveModelAgentBillingMethod(ctx, 1)
					retry = []int{}
				} else {

					if mak.RealModel.IsEnableFallback {

						if mak.RealModel.FallbackConfig.ModelAgent != "" && mak.RealModel.FallbackConfig.ModelAgent != mak.ModelAgent.Id && fallbackModelAgent == nil {
							if fallbackModelAgent, _ = service.ModelAgent().GetFallback(ctx, mak.RealModel); fallbackModelAgent != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.VideoCreate(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.VideoCreate(g.RequestFromCtx(ctx).GetCtx(), request, nil, fallbackModel)
							}
						}
					}

					return nil, err
				}
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.VideoCreate(g.RequestFromCtx(ctx).GetCtx(), request, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return nil, err
	}

	return responseBytes, nil
}

// VideoList
func (s *sVolcEngine) VideoList(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine VideoList time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		modelName = convModel(request)
		mak       = &common.MAK{
			Model:              modelName,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		totalTime int64
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					Action:       consts.ACTION_LIST,
					RequestData:  map[string]any{"model": modelName},
					ResponseData: util.ConvToMap(responseBytes),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	pageSize := request.GetQuery("page_size", 20).Int64()
	if pageSize > 500 {
		pageSize = 500
	}

	pageNum := request.GetQuery("page_num", 1).Int64()
	if pageNum < 1 {
		pageNum = 1
	}

	filter := bson.M{
		"creator":    service.Session().GetSecretKey(ctx),
		"created_at": bson.M{"$gt": time.Now().Add(-7 * 24 * time.Hour).UnixMilli()},
	}

	if status := request.GetQuery("filter.status").String(); status != "" {
		filter["status"] = status
	}

	if filterModel := request.GetQuery("filter.model").String(); filterModel != "" {
		filter["model"] = filterModel
		modelName = filterModel
	}

	paging := &db.Paging{
		Page:     pageNum,
		PageSize: pageSize,
	}

	results, err := dao.TaskVideo.FindByPage(ctx, paging, filter, &dao.FindOptions{SortFields: []string{"-created_at"}})
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if len(results) > 0 && modelName == "" {
		modelName = results[0].Model
		mak.Model = modelName
	}

	if modelName != "" {
		if err = mak.InitMAK(ctx); err != nil {
			logger.Error(ctx, err)
			return nil, err
		}
	}

	listRes := smodel.VolcVideoListRes{
		Total: int(paging.Total),
	}

	for _, result := range results {
		listRes.Items = append(listRes.Items, convTaskVideoToVolcRes(result))
	}

	if listRes.Items == nil {
		listRes.Items = make([]*smodel.VolcVideoTaskRes, 0)
	}

	responseBytes, err = json.Marshal(listRes)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return responseBytes, nil
}

// VideoRetrieve
func (s *sVolcEngine) VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine VideoRetrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		totalTime int64
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					VideoId:      taskId,
					RequestData:  map[string]any{"task_id": taskId},
					ResponseData: util.ConvToMap(responseBytes),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": taskId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+taskId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return nil, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	volcRes := convTaskVideoToVolcRes(taskVideo)

	responseBytes, err = json.Marshal(volcRes)
	if err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	return responseBytes, nil
}

// VideoDelete
func (s *sVolcEngine) VideoDelete(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sVolcEngine VideoDelete time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		totalTime int64
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					Action:       consts.ACTION_DELETE,
					VideoId:      taskId,
					RequestData:  map[string]any{"task_id": taskId},
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    totalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				})

			}); err != nil {
				logger.Error(ctx, err)
			}
		}
	}()

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": taskId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+taskId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	if err := dao.TaskVideo.UpdateById(ctx, taskVideo.Id, bson.M{"status": "cancelled", "video_url": "", "file_name": "", "file_path": ""}); err != nil {
		logger.Error(ctx, err)
	}

	return nil
}

func convModel(request *ghttp.Request) string {

	m := request.GetHeader("X-Model")

	if m == "" {
		m = request.GetQuery("filter.model").String()
	}

	if m == "" {
		if j, err := request.GetJson(); err == nil {
			m = j.Get("model").String()
		}
	}

	return m
}

func convCreateRequest(request *ghttp.Request) *smodel.VolcVideoCreateReq {

	req := new(smodel.VolcVideoCreateReq)

	if j, err := request.GetJson(); err == nil {
		if err := j.Scan(req); err != nil {
			req.Model = j.Get("model").String()
		}
	}

	return req
}

func detectVideoMode(req *smodel.VolcVideoCreateReq) string {
	for _, c := range req.Content {
		if c.Type == "video_url" {
			return "has_video_input"
		}
	}
	return "no_video_input"
}

func convTaskVideoToVolcRes(task *entity.TaskVideo) *smodel.VolcVideoTaskRes {

	response := smodel.VideoJobResponse{
		Id:        task.VideoId,
		Object:    "video",
		Model:     task.Model,
		Status:    task.Status,
		CreatedAt: task.CreatedAt / 1000,
		Size:      fmt.Sprintf("%dx%d", task.Width, task.Height),
		Seconds:   gconv.String(task.Seconds),
	}

	if task.CompletedAt != 0 {
		response.CompletedAt = &task.CompletedAt
	}

	if task.VideoUrl != "" {
		response.VideoUrl = task.VideoUrl
	}

	if task.Error != nil {
		response.Error = &smodel.VideoError{
			Code:    task.Error.Code,
			Message: task.Error.Message,
		}
	}

	converter := &volcengine.VolcEngine{}
	volcRes, _ := converter.ConvVideoJobResponseOfficial(context.Background(), response)

	if task.UpdatedAt > 0 {
		volcRes.UpdatedAt = task.UpdatedAt / 1000
	}

	return volcRes
}
