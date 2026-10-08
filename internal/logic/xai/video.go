package xai

import (
	"context"
	"encoding/json"
	"net/http"
	"slices"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/dao"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/internal/model"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/model/entity"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
	"github.com/iimeta/fastapi/v2/utility/util"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type sXAI struct{}

func init() {
	service.RegisterXAI(New())
}

func New() service.IXAI {
	return &sXAI{}
}

// VideoCreate
func (s *sXAI) VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sXAI VideoCreate time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		params = convVideoCreateRequest(request)
		mak    = &common.MAK{
			Model:              params.Model,
			Endpoint:           consts.ENDPOINT_VIDEO_GENERATIONS,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo *mcommon.Retry
		totalTime int64
		videoId   string
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				afterHandler := &mcommon.AfterHandler{
					Action:       consts.ACTION_CREATE,
					IsAsync:      true,
					VideoId:      videoId,
					Prompt:       params.Prompt,
					Seconds:      videoSeconds(params),
					Size:         videoSize(params),
					VideoMode:    detectVideoMode(params),
					RequestData:  util.ConvToMap(request.GetBody()),
					ResponseData: util.ConvToMap(responseBytes),
					Error:        err,
					RetryInfo:    retryInfo,
					TotalTime:    totalTime,
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
		return nil, err
	}

	videoId = "video_" + gtrace.GetTraceID(ctx)

	responseBytes = gjson.MustEncode(map[string]any{"request_id": videoId})

	return responseBytes, nil
}

// VideoRetrieve
func (s *sXAI) VideoRetrieve(ctx context.Context, request *ghttp.Request, requestId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sXAI VideoRetrieve time: %d", gtime.TimestampMilli()-now)
	}()

	var (
		mak = &common.MAK{
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo      *mcommon.Retry
		totalTime      int64
		responseHeader http.Header
	)

	defer func() {

		totalTime = gtime.TimestampMilli() - now
		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				common.AfterHandler(ctx, mak, &mcommon.AfterHandler{
					Action:       consts.ACTION_RETRIEVE,
					VideoId:      requestId,
					RequestData:  map[string]any{"request_id": requestId},
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

	taskVideo, err := dao.TaskVideo.FindOne(ctx, bson.M{"video_id": requestId, "creator": service.Session().GetSecretKey(ctx)})
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			err = errors.NewError(404, "invalid_request_error", "Video with id '"+requestId+"' not found.", "invalid_request_error", nil)
		}
		logger.Error(ctx, err)
		return nil, err
	}

	mak.Model = taskVideo.Model

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return nil, err
	}

	if responseBytes = convTaskVideoToXAIRes(ctx, taskVideo); responseBytes != nil {
		// 响应中的错误信息按 errors.ShieldErrorJson 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
		return errors.ShieldErrorJson(ctx, responseBytes), nil
	}

	responseBytes, responseHeader, err = common.NewAdapterOfficial(ctx, mak, false).VideoRetrieveOfficial(ctx, requestId)
	if err != nil {
		logger.Error(ctx, err)

		service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

		isRetry, isDisabled := common.IsNeedRetry(err)

		if isDisabled {
			if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

				service.ModelAgent().DisabledKey(ctx, mak.Key, err.Error())

			}, nil); err != nil {
				logger.Error(ctx, err)
			}
		}

		if isRetry {

			if common.IsMaxRetry(mak.AgentTotal, len(retry)) {

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
								return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, requestId, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, requestId, nil, fallbackModel)
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

			return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, requestId, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return nil, err
	}

	common.WritePassthroughHeaders(ctx, mak.Passthrough, responseHeader)

	// 响应中的错误信息按 errors.ShieldErrorJson 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
	return errors.ShieldErrorJson(ctx, responseBytes), nil
}

func checkVideoCreateResponse(responseBytes []byte) error {

	var createRes smodel.XAIVideoCreateRes
	if err := json.Unmarshal(responseBytes, &createRes); err != nil {
		return errors.NewError(500, "server_error", "invalid video create response", "server_error", nil)
	}

	if createRes.RequestId != "" {
		return nil
	}

	j := gjson.New(responseBytes)
	if id := j.Get("id").String(); id != "" {
		return nil
	}

	return errors.NewError(500, "server_error", "create video task failed: missing request_id", "server_error", nil)
}

func convVideoCreateRequest(request *ghttp.Request) *smodel.XAIVideoCreateReq {

	req := new(smodel.XAIVideoCreateReq)

	if j, err := request.GetJson(); err == nil {
		if err := j.Scan(req); err != nil {
			req.Model = j.Get("model").String()
			req.Prompt = j.Get("prompt").String()
		}
	}

	return req
}

func videoSeconds(req *smodel.XAIVideoCreateReq) int {
	if req.Duration > 0 {
		return req.Duration
	}
	return 8
}

func videoSize(req *smodel.XAIVideoCreateReq) string {

	resolution, ratio := "480p", "16:9"

	if req.Resolution != "" {
		resolution = gstr.ToLower(req.Resolution)
	}
	if req.AspectRatio != "" {
		ratio = req.AspectRatio
	}

	if size := consts.VIDEO_RESOLUTION_RATIO[resolution+ratio]; size != "" {
		return size
	}

	return ""
}

func detectVideoMode(req *smodel.XAIVideoCreateReq) string {
	if req.Video != nil && (req.Video.Url != "" || req.Video.FileId != "") {
		return "has_video_input"
	}
	return "no_video_input"
}

func convTaskVideoToXAIRes(ctx context.Context, task *entity.TaskVideo) []byte {

	status := "pending"
	switch task.Status {
	case "queued":
		status = "pending"
	case "in_progress":
		status = "in_progress"
	case "completed":
		status = "completed"
	case "failed":
		status = "failed"
	case "expired":
		status = "expired"
	case "deleted":
		status = "cancelled"
	}

	res := &smodel.XAIVideoJobRes{
		RequestId:   task.VideoId,
		Id:          task.VideoId,
		Object:      "video",
		Model:       task.Model,
		Status:      status,
		Prompt:      task.Prompt,
		Duration:    task.Seconds,
		AspectRatio: xaiReqString(task.RequestData, "aspect_ratio"),
		Resolution:  xaiReqString(task.RequestData, "resolution"),
		Usage:       xaiUsageFromResponse(task.ResponseData),
	}

	if task.VideoUrl != "" {
		res.Video = &smodel.XAIVideoGenerated{
			Url:      common.ReplaceVideoUrl(task.VideoUrl),
			Duration: task.Seconds,
		}
	}

	if task.Error != nil {
		res.Error = &smodel.XAIVideoError{Code: task.Error.Code, Message: task.Error.Message}
	}

	data, err := json.Marshal(res)
	if err != nil {
		logger.Error(ctx, err)
		return nil
	}

	return data
}

func xaiReqString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return gconv.String(m[key])
}

func xaiUsageFromResponse(data map[string]any) *smodel.XAIUsage {

	if data == nil {
		return nil
	}

	raw := data["usage"]
	if raw == nil {
		return nil
	}

	usage := &smodel.XAIUsage{}
	if err := json.Unmarshal(gjson.MustEncode(raw), usage); err != nil {
		return nil
	}

	return usage
}
