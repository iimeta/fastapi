package bailian

import (
	"context"
	"encoding/json"
	"fmt"
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

// VideoCreate
func (s *sBailian) VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBailian VideoCreate time: %d", gtime.TimestampMilli()-now)
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
					Prompt:       params.Input.Prompt,
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

	responseBytes = gjson.MustEncode(map[string]any{"output": map[string]any{"task_id": videoId, "task_status": "PENDING"}})

	return responseBytes, nil
}

// VideoRetrieve
func (s *sBailian) VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sBailian VideoRetrieve time: %d", gtime.TimestampMilli()-now)
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

	// 后台任务已轮询到结果则直接返回落库的官方响应, 否则实时查询上游
	if responseBytes = convTaskVideoToBailianRes(ctx, taskVideo); responseBytes != nil {
		// 响应中的错误信息按 errors.ShieldErrorJson 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
		return errors.ShieldErrorJson(ctx, responseBytes), nil
	}

	responseBytes, responseHeader, err = common.NewAdapterOfficial(ctx, mak, false).VideoRetrieveOfficial(ctx, taskId)
	if err != nil {
		logger.Error(ctx, err)

		// 记录错误次数和禁用
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
								return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, taskId, fallbackModelAgent, fallbackModel)
							}
						}

						if mak.RealModel.FallbackConfig.Model != "" && fallbackModel == nil {
							if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
								retryInfo = &mcommon.Retry{
									IsRetry:    true,
									RetryCount: len(retry),
									ErrMsg:     err.Error(),
								}
								return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, taskId, nil, fallbackModel)
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

			return s.VideoRetrieve(g.RequestFromCtx(ctx).GetCtx(), request, taskId, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return nil, err
	}

	// 响应头透传
	common.WritePassthroughHeaders(ctx, mak.Passthrough, responseHeader)

	// 响应中的错误信息按 errors.ShieldErrorJson 逻辑处理: 屏蔽的错误统一返回未知错误, 不屏蔽的原样返回
	return errors.ShieldErrorJson(ctx, responseBytes), nil
}

func convVideoCreateRequest(request *ghttp.Request) *smodel.BailianVideoCreateReq {

	req := new(smodel.BailianVideoCreateReq)

	if j, err := request.GetJson(); err == nil {
		if err := j.Scan(req); err != nil {
			req.Model = j.Get("model").String()
		}
	}

	return req
}

// 计费秒数: 未传 duration 时按官方默认 5 秒; -1 为智能时长, 创建时无法确定, 交由后台任务在完成后按实际时长补计费
func videoSeconds(req *smodel.BailianVideoCreateReq) int {

	if req.Parameters == nil || req.Parameters.Duration == nil {
		return 5
	}

	if *req.Parameters.Duration < 0 {
		return 0
	}

	return *req.Parameters.Duration
}

// 按 resolution + ratio 映射为像素尺寸用于匹配定价, 默认 1080P; ratio 为 adaptive 或缺省时按 16:9 估算
func videoSize(req *smodel.BailianVideoCreateReq) string {

	resolution, ratio := "1080p", "16:9"

	if req.Parameters != nil {
		if req.Parameters.Resolution != "" {
			resolution = gstr.ToLower(req.Parameters.Resolution)
		}
		if req.Parameters.Ratio != "" && req.Parameters.Ratio != "adaptive" {
			ratio = req.Parameters.Ratio
		}
	}

	return consts.VIDEO_RESOLUTION_RATIO[resolution+ratio]
}

func detectVideoMode(req *smodel.BailianVideoCreateReq) string {
	for _, media := range req.Input.Media {
		if media.Type == "reference_video" {
			return "has_video_input"
		}
	}
	return "no_video_input"
}

func convTaskVideoToBailianRes(ctx context.Context, task *entity.TaskVideo) []byte {

	res := &smodel.BailianVideoTaskRes{
		RequestId: task.VideoId,
		Output: &smodel.BailianVideoOutput{
			TaskId:     task.VideoId,
			TaskStatus: bailianStatus(task.Status),
			OrigPrompt: task.Prompt,
		},
	}

	if task.CreatedAt > 0 {
		res.Output.SubmitTime = bailianTime(task.CreatedAt)
	}

	if task.Status == "in_progress" || task.Status == "completed" || task.Status == "failed" {
		if task.UpdatedAt > 0 {
			res.Output.ScheduledTime = bailianTime(task.UpdatedAt)
		}
	}

	if task.CompletedAt > 0 {
		res.Output.EndTime = bailianTime(task.CompletedAt * 1000)
	} else if task.Status == "failed" && task.UpdatedAt > 0 {
		res.Output.EndTime = bailianTime(task.UpdatedAt)
	}

	if task.VideoUrl != "" {
		res.Output.VideoUrl = common.ReplaceVideoUrl(task.VideoUrl)
	}

	if task.Error != nil {
		res.Output.Code = task.Error.Code
		res.Output.Message = task.Error.Message
	}

	res.Usage = bailianUsageFromTask(task)

	data, err := json.Marshal(res)
	if err != nil {
		logger.Error(ctx, err)
		return nil
	}

	return data
}

func bailianStatus(status string) string {
	switch status {
	case "queued":
		return "PENDING"
	case "in_progress":
		return "RUNNING"
	case "completed":
		return "SUCCEEDED"
	case "failed":
		return "FAILED"
	case "deleted", "expired":
		return "CANCELED"
	default:
		return "PENDING"
	}
}

func bailianUsageFromResponse(data map[string]any) *smodel.BailianVideoUsage {

	if data == nil {
		return nil
	}

	raw, ok := data["usage"]
	if !ok {
		return nil
	}

	usage := &smodel.BailianVideoUsage{}
	if err := json.Unmarshal(gjson.MustEncode(raw), usage); err != nil {
		return nil
	}

	return usage
}

func bailianUsageFromTask(task *entity.TaskVideo) *smodel.BailianVideoUsage {

	if usage := bailianUsageFromResponse(task.ResponseData); usage != nil {
		return usage
	}

	if task.Status != "completed" {
		return nil
	}

	return &smodel.BailianVideoUsage{
		VideoCount:          1,
		Duration:            float64(task.Seconds),
		OutputVideoDuration: float64(task.Seconds),
		Fps:                 30,
		SR:                  bailianSR(task.RequestData),
		Ratio:               bailianParamString(task.RequestData, "ratio", "16:9"),
	}
}

func bailianTime(ms int64) string {

	if ms <= 0 {
		return ""
	}

	return gtime.NewFromTimeStamp(ms).Format("Y-m-d H:i:s") + fmt.Sprintf(".%03d", ms%1000)
}

func bailianParamString(req map[string]any, key, def string) string {

	if req != nil {
		if params, ok := req["parameters"].(map[string]any); ok {
			if s := gconv.String(params[key]); s != "" {
				return s
			}
		}
	}

	return def
}

func bailianSR(req map[string]any) int {
	res := gstr.ToUpper(bailianParamString(req, "resolution", "1080P"))
	switch {
	case gstr.Contains(res, "480"):
		return 480
	case gstr.Contains(res, "720"):
		return 720
	default:
		return 1080
	}
}
