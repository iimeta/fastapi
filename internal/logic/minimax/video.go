package minimax

import (
	"context"
	"encoding/json"

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

type sMiniMax struct{}

func init() {
	service.RegisterMiniMax(New())
}

func New() service.IMiniMax {
	return &sMiniMax{}
}

// VideoCreate
func (s *sMiniMax) VideoCreate(ctx context.Context, request *ghttp.Request, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMiniMax VideoCreate time: %d", gtime.TimestampMilli()-now)
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
					Prompt:       videoPrompt(params),
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

	responseBytes = gjson.MustEncode(map[string]any{"task_id": videoId})

	return responseBytes, nil
}

// VideoRetrieve
func (s *sMiniMax) VideoRetrieve(ctx context.Context, request *ghttp.Request, taskId string, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (responseBytes []byte, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sMiniMax VideoRetrieve time: %d", gtime.TimestampMilli()-now)
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

	if taskVideo.Error != nil {
		taskVideo.Error.Code, taskVideo.Error.Message = errors.ShieldError(ctx, taskVideo.Error.Code, taskVideo.Error.Message)
	}

	responseBytes = convTaskVideoToMiniMaxRes(ctx, taskVideo)
	if responseBytes == nil {
		err = errors.NewError(500, "server_error", "invalid video retrieve response", "server_error", nil)
		logger.Error(ctx, err)
		return nil, err
	}

	return responseBytes, nil
}

func convVideoCreateRequest(request *ghttp.Request) *smodel.MiniMaxVideoCreateReq {

	req := new(smodel.MiniMaxVideoCreateReq)

	if j, err := request.GetJson(); err == nil {
		if err := j.Scan(req); err != nil {
			req.Model = j.Get("model").String()
		}
	}

	return req
}

func videoPrompt(req *smodel.MiniMaxVideoCreateReq) string {
	for _, item := range req.Content {
		if item.Type == "text" && item.Text != "" {
			return item.Text
		}
	}
	return ""
}

func videoSeconds(req *smodel.MiniMaxVideoCreateReq) int {
	if req.Duration > 0 {
		return req.Duration
	}
	return 5
}

// 按 resolution + ratio 映射为像素尺寸用于匹配定价, 默认 768P 16:9; ratio 为 adaptive 或缺省时按 16:9 估算
func videoSize(req *smodel.MiniMaxVideoCreateReq) string {

	resolution, ratio := "768p", "16:9"

	if req.Resolution != "" {
		resolution = gstr.ToLower(req.Resolution)
	}
	if req.Ratio != "" && req.Ratio != "adaptive" {
		ratio = req.Ratio
	}

	return consts.VIDEO_RESOLUTION_RATIO[resolution+ratio]
}

func detectVideoMode(req *smodel.MiniMaxVideoCreateReq) string {
	for _, item := range req.Content {
		if item.Type == "video_url" || item.Role == "reference_video" {
			return "has_video_input"
		}
	}
	return "no_video_input"
}

func convTaskVideoToMiniMaxRes(ctx context.Context, task *entity.TaskVideo) []byte {

	status := "queued"
	switch task.Status {
	case "in_progress":
		status = "running"
	case "completed":
		status = "succeeded"
	case "failed":
		status = "failed"
	case "expired", "deleted":
		status = "cancelled"
	}

	item := &smodel.MiniMaxVideoTask{
		Id:         task.VideoId,
		Model:      task.Model,
		Status:     status,
		CreatedAt:  task.CreatedAt / 1000,
		UpdatedAt:  task.UpdatedAt / 1000,
		Duration:   task.Seconds,
		Resolution: miniMaxMapString(task.RequestData, "resolution", "768P"),
		Ratio:      miniMaxMapString(task.RequestData, "ratio", "16:9"),
		TaskType:   miniMaxMapString(task.RequestData, "task_type", "generation"),
		Modality:   "video",
		Progress:   miniMaxProgress(task, status),
		Usage:      miniMaxUsageFromTask(task, status),
	}

	if task.VideoUrl != "" {
		item.Content = &smodel.MiniMaxVideoContent{
			Url: common.ReplaceVideoUrl(task.VideoUrl),
		}
	}

	if task.Error != nil {
		item.Error = &smodel.MiniMaxVideoError{Code: task.Error.Code, Message: task.Error.Message}
	}

	res := smodel.MiniMaxVideoQueryRes{Task: item}
	data, err := json.Marshal(res)
	if err != nil {
		logger.Error(ctx, err)
		return nil
	}

	return data
}

func miniMaxProgress(task *entity.TaskVideo, status string) *float64 {

	switch status {
	case "succeeded", "failed", "cancelled":
		p := 1.0
		return &p
	case "running":
		p := float64(task.Progress) / 100
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		return &p
	default:
		return nil
	}
}

func miniMaxUsageFromTask(task *entity.TaskVideo, status string) *smodel.MiniMaxVideoUsage {

	usage := miniMaxUsageFromResponse(task.ResponseData)
	if usage == nil {
		usage = &smodel.MiniMaxVideoUsage{
			InputImageCount: miniMaxInputImageCount(task.RequestData),
		}
	}

	if status == "succeeded" && task.Seconds > 0 {
		if usage.OutputSeconds == 0 {
			usage.OutputSeconds = task.Seconds
		}
		if usage.TotalSeconds == 0 {
			usage.TotalSeconds = task.Seconds
		}
	}

	return usage
}

func miniMaxUsageFromResponse(data map[string]any) *smodel.MiniMaxVideoUsage {

	if data == nil {
		return nil
	}

	raw := data["usage"]
	if task, ok := data["task"].(map[string]any); ok {
		if u, exists := task["usage"]; exists {
			raw = u
		}
	}

	if raw == nil {
		return nil
	}

	usage := &smodel.MiniMaxVideoUsage{}
	if err := json.Unmarshal(gjson.MustEncode(raw), usage); err != nil {
		return nil
	}

	return usage
}

func miniMaxInputImageCount(req map[string]any) int {

	if req == nil {
		return 0
	}

	content, ok := req["content"].([]any)
	if !ok {
		return 0
	}

	n := 0
	for _, item := range content {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if gconv.String(m["type"]) == "image_url" {
			n++
		}
	}

	return n
}

func miniMaxMapString(m map[string]any, key, def string) string {

	if m != nil {
		if v, ok := m[key]; ok {
			if s := gconv.String(v); s != "" {
				return s
			}
		}
	}

	return def
}
