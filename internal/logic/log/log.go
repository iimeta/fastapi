package log

import (
	"context"
	"slices"
	"time"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/gogf/gf/v2/util/gconv"
	sconsts "github.com/iimeta/fastapi-sdk/consts"
	serrors "github.com/iimeta/fastapi-sdk/errors"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/logic/common"
	"github.com/iimeta/fastapi/internal/model"
	mcommon "github.com/iimeta/fastapi/internal/model/common"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sLog struct{}

func init() {
	service.RegisterLog(New())
}

func New() service.ILog {
	return &sLog{}
}

// 文本日志
func (s *sLog) Text(ctx context.Context, textLog model.LogText, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Text time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if textLog.CompletionsRes.Error != nil && checkError(textLog.CompletionsRes.Error) {
		return
	}

	text := do.LogText{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: textLog.IsSmartMatch,
		Stream:       textLog.CompletionsReq.Stream,
		Spend:        textLog.Spend,
		ConnTime:     textLog.CompletionsRes.ConnTime,
		Duration:     textLog.CompletionsRes.Duration,
		TotalTime:    textLog.CompletionsRes.TotalTime,
		InternalTime: textLog.CompletionsRes.InternalTime,
		ReqTime:      textLog.CompletionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(textLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.TextRecords, "prompt") {

		if textLog.CompletionsReq != nil {

			if len(textLog.CompletionsReq.Messages) > 0 {

				prompt := textLog.CompletionsReq.Messages[len(textLog.CompletionsReq.Messages)-1].Content

				if textLog.ReqModel.Type == 102 {

					if slices.Contains(config.Cfg.Log.TextRecords, "audio") {
						text.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]any); ok {

							multiContents := make([]any, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]any); ok {

									if content["type"] == "input_audio" {

										if inputAudio, ok := content["input_audio"].(map[string]any); ok {

											inputAudio = gmap.NewStrAnyMapFrom(inputAudio).MapCopy()
											inputAudio["data"] = "[BASE64音频数据]"

											content = gmap.NewStrAnyMapFrom(content).MapCopy()
											content["input_audio"] = inputAudio
										}
									}

									value = content
								}

								multiContents = append(multiContents, value)
							}

							text.Prompt = gconv.String(multiContents)

						} else {
							text.Prompt = gconv.String(prompt)
						}
					}

				} else {

					if slices.Contains(config.Cfg.Log.TextRecords, "image") {
						text.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]any); ok {

							multiContents := make([]any, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]any); ok {

									if content["type"] == "image_url" {

										if imageUrl, ok := content["image_url"].(map[string]any); ok {

											if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {

												imageUrl = gmap.NewStrAnyMapFrom(imageUrl).MapCopy()
												imageUrl["url"] = "[BASE64图像数据]"

												content = gmap.NewStrAnyMapFrom(content).MapCopy()
												content["image_url"] = imageUrl
											}
										}
									}

									if content["type"] == "image" {
										if source, ok := content["source"].(smodel.Source); ok {
											source.Data = "[BASE64图像数据]"
											content = gmap.NewStrAnyMapFrom(content).MapCopy()
											content["source"] = source
										}
									}

									value = content
								}

								multiContents = append(multiContents, value)
							}

							text.Prompt = gconv.String(multiContents)

						} else if multiContent, ok := prompt.([]smodel.OpenAIResponsesContent); ok {

							multiContents := make([]smodel.OpenAIResponsesContent, 0)

							for _, value := range multiContent {
								if value.Type == "input_image" && !gstr.HasPrefix(value.ImageUrl, "http") {
									value.ImageUrl = "[BASE64图像数据]"
								}
								multiContents = append(multiContents, value)
							}

							text.Prompt = gconv.String(multiContents)

						} else {
							text.Prompt = gconv.String(prompt)
						}
					}
				}
			}

		} else if textLog.EmbeddingReq != nil {
			text.Prompt = gconv.String(textLog.EmbeddingReq.Input)
		} else if textLog.ModerationReq != nil {
			text.Prompt = gconv.String(textLog.ModerationReq.Input)
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.TextRecords, "completion") && textLog.CompletionsRes != nil {
		text.Completion = textLog.CompletionsRes.Completion
	}

	if textLog.ReqModel != nil {
		text.ProviderId = textLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, textLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			text.ProviderName = provider.Name
		}
		text.ModelId = textLog.ReqModel.Id
		text.ModelName = textLog.ReqModel.Name
		text.Model = textLog.ReqModel.Model
		text.ModelType = textLog.ReqModel.Type
	}

	if textLog.RealModel != nil {
		text.IsEnablePresetConfig = textLog.RealModel.IsEnablePresetConfig
		text.PresetConfig = textLog.RealModel.PresetConfig
		text.IsEnableForward = textLog.RealModel.IsEnableForward
		text.ForwardConfig = textLog.RealModel.ForwardConfig
		text.IsEnableModelAgent = textLog.RealModel.IsEnableModelAgent
		text.RealModelId = textLog.RealModel.Id
		text.RealModelName = textLog.RealModel.Name
		text.RealModel = textLog.RealModel.Model
	}

	if textLog.ModelAgent != nil {
		text.IsEnableModelAgent = true
		text.ModelAgentId = textLog.ModelAgent.Id
		text.ModelAgent = &do.ModelAgent{
			ProviderId: textLog.ModelAgent.ProviderId,
			Name:       textLog.ModelAgent.Name,
			BaseUrl:    textLog.ModelAgent.BaseUrl,
			Path:       textLog.ModelAgent.Path,
			Weight:     textLog.ModelAgent.Weight,
			Remark:     textLog.ModelAgent.Remark,
		}
	}

	if textLog.FallbackModelAgent != nil {
		text.IsEnableFallback = true
		text.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     textLog.FallbackModelAgent.Id,
			ModelAgentName: textLog.FallbackModelAgent.Name,
		}
	}

	if textLog.FallbackModel != nil {
		text.IsEnableFallback = true
		if text.FallbackConfig == nil {
			text.FallbackConfig = new(mcommon.FallbackConfig)
		}
		text.FallbackConfig.Model = textLog.FallbackModel.Model
		text.FallbackConfig.ModelName = textLog.FallbackModel.Name
	}

	if textLog.Key != nil {
		text.Key = textLog.Key.Key
	}

	if textLog.CompletionsRes.Error != nil {

		text.ErrMsg = textLog.CompletionsRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(textLog.CompletionsRes.Error, &openaiApiError) {
			text.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(textLog.CompletionsRes.Error) {
			text.Status = 2
		} else {
			text.Status = -1
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.TextRecords, "messages") && textLog.CompletionsReq != nil {
		for _, message := range textLog.CompletionsReq.Messages {

			content := message.Content

			if !slices.Contains(config.Cfg.Log.TextRecords, "image") {

				if multiContent, ok := content.([]any); ok {

					multiContents := make([]any, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]any); ok {

							if content["type"] == "image_url" {

								if imageUrl, ok := content["image_url"].(map[string]any); ok {

									if !gstr.HasPrefix(gconv.String(imageUrl["url"]), "http") {

										imageUrl = gmap.NewStrAnyMapFrom(imageUrl).MapCopy()
										imageUrl["url"] = "[BASE64图像数据]"

										content = gmap.NewStrAnyMapFrom(content).MapCopy()
										content["image_url"] = imageUrl
									}
								}
							}

							if content["type"] == "image" {
								if source, ok := content["source"].(smodel.Source); ok {
									source.Data = "[BASE64图像数据]"
									content = gmap.NewStrAnyMapFrom(content).MapCopy()
									content["source"] = source
								}
							}

							value = content
						}

						multiContents = append(multiContents, value)
					}

					content = multiContents

				} else if multiContent, ok := content.([]smodel.OpenAIResponsesContent); ok {

					multiContents := make([]smodel.OpenAIResponsesContent, 0)

					for _, value := range multiContent {
						if value.Type == "input_image" && !gstr.HasPrefix(value.ImageUrl, "http") {
							value.ImageUrl = "[BASE64图像数据]"
						}
						multiContents = append(multiContents, value)
					}

					content = multiContents
				}
			}

			if !slices.Contains(config.Cfg.Log.TextRecords, "audio") {

				if multiContent, ok := content.([]any); ok {

					multiContents := make([]any, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]any); ok {

							if content["type"] == "input_audio" {

								if inputAudio, ok := content["input_audio"].(map[string]any); ok {

									inputAudio = gmap.NewStrAnyMapFrom(inputAudio).MapCopy()
									inputAudio["data"] = "[BASE64音频数据]"

									content = gmap.NewStrAnyMapFrom(content).MapCopy()
									content["input_audio"] = inputAudio
								}
							}

							value = content
						}

						multiContents = append(multiContents, value)
					}

					content = multiContents
				}
			}

			text.Messages = append(text.Messages, mcommon.Message{
				Role:         message.Role,
				Content:      gconv.String(content),
				Refusal:      message.Refusal,
				Name:         message.Name,
				FunctionCall: message.FunctionCall,
				ToolCalls:    message.ToolCalls,
				ToolCallId:   message.ToolCallId,
				Audio:        message.Audio,
			})
		}
	}

	if textLog.RetryInfo != nil {

		text.IsRetry = textLog.RetryInfo.IsRetry
		text.Retry = &mcommon.Retry{
			IsRetry:    textLog.RetryInfo.IsRetry,
			RetryCount: textLog.RetryInfo.RetryCount,
			ErrMsg:     textLog.RetryInfo.ErrMsg,
		}

		if text.IsRetry {
			text.Status = 3
			text.ErrMsg = textLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogText.Insert(ctx, text); err != nil {
		logger.Errorf(ctx, "sLog Text error: %v", err)

		if err.Error() == "an inserted document is too large" {
			if textLog.CompletionsReq != nil {
				textLog.CompletionsReq.Messages = []smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: err.Error(),
				}}
			} else if textLog.EmbeddingReq != nil {
				textLog.EmbeddingReq.Input = err.Error()
			} else if textLog.ModerationReq != nil {
				textLog.ModerationReq.Input = err.Error()
			}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Text retry: %d", len(retry))

		s.Text(ctx, textLog, retry...)
	}
}

// 绘图日志
func (s *sLog) Image(ctx context.Context, imageLog model.LogImage, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Image time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if imageLog.ImageRes.Error != nil && checkError(imageLog.ImageRes.Error) {
		return
	}

	image := do.LogImage{
		TraceId:        gtrace.GetTraceID(ctx),
		UserId:         service.Session().GetUserId(ctx),
		AppId:          service.Session().GetAppId(ctx),
		Prompt:         imageLog.ImageReq.Prompt,
		Size:           imageLog.ImageReq.Size,
		N:              imageLog.ImageReq.N,
		Quality:        imageLog.ImageReq.Quality,
		Style:          imageLog.ImageReq.Style,
		ResponseFormat: imageLog.ImageReq.ResponseFormat,
		Spend:          imageLog.Spend,
		TotalTime:      imageLog.ImageRes.TotalTime,
		InternalTime:   imageLog.ImageRes.InternalTime,
		ReqTime:        imageLog.ImageRes.EnterTime,
		ReqDate:        gtime.NewFromTimeStamp(imageLog.ImageRes.EnterTime).Format("Y-m-d"),
		ClientIp:       g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:       g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:        util.GetLocalIp(),
		Status:         1,
		Host:           g.RequestFromCtx(ctx).GetHost(),
		Rid:            service.Session().GetRid(ctx),
	}

	for _, data := range imageLog.ImageRes.Data {
		image.ImageData = append(image.ImageData, mcommon.ImageData{
			Url: data.Url,
			//B64Json:       data.B64Json, // todo 太大了, 不存
			RevisedPrompt: data.RevisedPrompt,
		})
	}

	if imageLog.ReqModel != nil {
		image.ProviderId = imageLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, imageLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			image.ProviderName = provider.Name
		}
		image.ModelId = imageLog.ReqModel.Id
		image.ModelName = imageLog.ReqModel.Name
		image.Model = imageLog.ReqModel.Model
		image.ModelType = imageLog.ReqModel.Type
	}

	if imageLog.RealModel != nil {
		image.IsEnablePresetConfig = imageLog.RealModel.IsEnablePresetConfig
		image.PresetConfig = imageLog.RealModel.PresetConfig
		image.IsEnableForward = imageLog.RealModel.IsEnableForward
		image.ForwardConfig = imageLog.RealModel.ForwardConfig
		image.IsEnableModelAgent = imageLog.RealModel.IsEnableModelAgent
		image.RealModelId = imageLog.RealModel.Id
		image.RealModelName = imageLog.RealModel.Name
		image.RealModel = imageLog.RealModel.Model
	}

	if image.IsEnableModelAgent && imageLog.ModelAgent != nil {
		image.ModelAgentId = imageLog.ModelAgent.Id
		image.ModelAgent = &do.ModelAgent{
			ProviderId: imageLog.ModelAgent.ProviderId,
			Name:       imageLog.ModelAgent.Name,
			BaseUrl:    imageLog.ModelAgent.BaseUrl,
			Path:       imageLog.ModelAgent.Path,
			Weight:     imageLog.ModelAgent.Weight,
			Remark:     imageLog.ModelAgent.Remark,
		}
	}

	if imageLog.FallbackModelAgent != nil {
		image.IsEnableFallback = true
		image.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     imageLog.FallbackModelAgent.Id,
			ModelAgentName: imageLog.FallbackModelAgent.Name,
		}
	}

	if imageLog.FallbackModel != nil {
		image.IsEnableFallback = true
		if image.FallbackConfig == nil {
			image.FallbackConfig = new(mcommon.FallbackConfig)
		}
		image.FallbackConfig.Model = imageLog.FallbackModel.Model
		image.FallbackConfig.ModelName = imageLog.FallbackModel.Name
	}

	if imageLog.Key != nil {
		image.Key = imageLog.Key.Key
	}

	if imageLog.ImageRes.Error != nil {

		image.ErrMsg = imageLog.ImageRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(imageLog.ImageRes.Error, &openaiApiError) {
			image.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(imageLog.ImageRes.Error) {
			image.Status = 2
		} else {
			image.Status = -1
		}
	}

	if imageLog.RetryInfo != nil {

		image.IsRetry = imageLog.RetryInfo.IsRetry
		image.Retry = &mcommon.Retry{
			IsRetry:    imageLog.RetryInfo.IsRetry,
			RetryCount: imageLog.RetryInfo.RetryCount,
			ErrMsg:     imageLog.RetryInfo.ErrMsg,
		}

		if image.IsRetry {
			image.Status = 3
			image.ErrMsg = imageLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogImage.Insert(ctx, image); err != nil {
		logger.Errorf(ctx, "sLog Image error: %v", err)

		if err.Error() == "an inserted document is too large" {
			imageLog.ImageReq.Prompt = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Image retry: %d", len(retry))

		s.Image(ctx, imageLog, retry...)
	}
}

// 音频日志
func (s *sLog) Audio(ctx context.Context, audioLog model.LogAudio, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Audio time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if audioLog.AudioRes.Error != nil && checkError(audioLog.AudioRes.Error) {
		return
	}

	audio := do.LogAudio{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Input:        audioLog.AudioReq.Input,
		Text:         audioLog.AudioRes.Text,
		Spend:        audioLog.Spend,
		TotalTime:    audioLog.AudioRes.TotalTime,
		InternalTime: audioLog.AudioRes.InternalTime,
		ReqTime:      audioLog.AudioRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(audioLog.AudioRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if audioLog.ReqModel != nil {
		audio.ProviderId = audioLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, audioLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			audio.ProviderName = provider.Name
		}
		audio.ModelId = audioLog.ReqModel.Id
		audio.ModelName = audioLog.ReqModel.Name
		audio.Model = audioLog.ReqModel.Model
		audio.ModelType = audioLog.ReqModel.Type
	}

	if audioLog.RealModel != nil {
		audio.IsEnablePresetConfig = audioLog.RealModel.IsEnablePresetConfig
		audio.PresetConfig = audioLog.RealModel.PresetConfig
		audio.IsEnableForward = audioLog.RealModel.IsEnableForward
		audio.ForwardConfig = audioLog.RealModel.ForwardConfig
		audio.IsEnableModelAgent = audioLog.RealModel.IsEnableModelAgent
		audio.RealModelId = audioLog.RealModel.Id
		audio.RealModelName = audioLog.RealModel.Name
		audio.RealModel = audioLog.RealModel.Model
	}

	if audio.IsEnableModelAgent && audioLog.ModelAgent != nil {
		audio.ModelAgentId = audioLog.ModelAgent.Id
		audio.ModelAgent = &do.ModelAgent{
			ProviderId: audioLog.ModelAgent.ProviderId,
			Name:       audioLog.ModelAgent.Name,
			BaseUrl:    audioLog.ModelAgent.BaseUrl,
			Path:       audioLog.ModelAgent.Path,
			Weight:     audioLog.ModelAgent.Weight,
			Remark:     audioLog.ModelAgent.Remark,
			Status:     audioLog.ModelAgent.Status,
		}
	}

	if audioLog.FallbackModelAgent != nil {
		audio.IsEnableFallback = true
		audio.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     audioLog.FallbackModelAgent.Id,
			ModelAgentName: audioLog.FallbackModelAgent.Name,
		}
	}

	if audioLog.FallbackModel != nil {
		audio.IsEnableFallback = true
		if audio.FallbackConfig == nil {
			audio.FallbackConfig = new(mcommon.FallbackConfig)
		}
		audio.FallbackConfig.Model = audioLog.FallbackModel.Model
		audio.FallbackConfig.ModelName = audioLog.FallbackModel.Name
	}

	if audioLog.Key != nil {
		audio.Key = audioLog.Key.Key
	}

	if audioLog.AudioRes.Error != nil {

		audio.ErrMsg = audioLog.AudioRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(audioLog.AudioRes.Error, &openaiApiError) {
			audio.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(audioLog.AudioRes.Error) {
			audio.Status = 2
		} else {
			audio.Status = -1
		}
	}

	if audioLog.RetryInfo != nil {

		audio.IsRetry = audioLog.RetryInfo.IsRetry
		audio.Retry = &mcommon.Retry{
			IsRetry:    audioLog.RetryInfo.IsRetry,
			RetryCount: audioLog.RetryInfo.RetryCount,
			ErrMsg:     audioLog.RetryInfo.ErrMsg,
		}

		if audio.IsRetry {
			audio.Status = 3
			audio.ErrMsg = audioLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogAudio.Insert(ctx, audio); err != nil {
		logger.Errorf(ctx, "sLog Audio error: %v", err)

		if err.Error() == "an inserted document is too large" {
			audioLog.AudioReq.Input = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Audio retry: %d", len(retry))

		s.Audio(ctx, audioLog, retry...)
	}
}

// 视频日志
func (s *sLog) Video(ctx context.Context, videoLog model.LogVideo, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Video time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if videoLog.VideoRes.Error != nil && checkError(videoLog.VideoRes.Error) {
		return
	}

	video := do.LogVideo{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Action:       videoLog.VideoReq.Action,
		VideoId:      videoLog.VideoRes.VideoId,
		RequestData:  videoLog.VideoReq.RequestData,
		ResponseData: videoLog.VideoRes.ResponseData,
		Spend:        videoLog.Spend,
		TotalTime:    videoLog.VideoRes.TotalTime,
		InternalTime: videoLog.VideoRes.InternalTime,
		ReqTime:      videoLog.VideoRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(videoLog.VideoRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if videoLog.ReqModel != nil {
		video.ProviderId = videoLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, videoLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			video.ProviderName = provider.Name
		}
		video.ModelId = videoLog.ReqModel.Id
		video.ModelName = videoLog.ReqModel.Name
		video.Model = videoLog.ReqModel.Model
		video.ModelType = videoLog.ReqModel.Type
	}

	if videoLog.RealModel != nil {
		video.IsEnablePresetConfig = videoLog.RealModel.IsEnablePresetConfig
		video.PresetConfig = videoLog.RealModel.PresetConfig
		video.IsEnableForward = videoLog.RealModel.IsEnableForward
		video.ForwardConfig = videoLog.RealModel.ForwardConfig
		video.IsEnableModelAgent = videoLog.RealModel.IsEnableModelAgent
		video.RealModelId = videoLog.RealModel.Id
		video.RealModelName = videoLog.RealModel.Name
		video.RealModel = videoLog.RealModel.Model
	}

	if video.IsEnableModelAgent && videoLog.ModelAgent != nil {
		video.ModelAgentId = videoLog.ModelAgent.Id
		video.ModelAgent = &do.ModelAgent{
			ProviderId: videoLog.ModelAgent.ProviderId,
			Name:       videoLog.ModelAgent.Name,
			BaseUrl:    videoLog.ModelAgent.BaseUrl,
			Path:       videoLog.ModelAgent.Path,
			Weight:     videoLog.ModelAgent.Weight,
			Remark:     videoLog.ModelAgent.Remark,
			Status:     videoLog.ModelAgent.Status,
		}
	}

	if videoLog.FallbackModelAgent != nil {
		video.IsEnableFallback = true
		video.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     videoLog.FallbackModelAgent.Id,
			ModelAgentName: videoLog.FallbackModelAgent.Name,
		}
	}

	if videoLog.FallbackModel != nil {
		video.IsEnableFallback = true
		if video.FallbackConfig == nil {
			video.FallbackConfig = new(mcommon.FallbackConfig)
		}
		video.FallbackConfig.Model = videoLog.FallbackModel.Model
		video.FallbackConfig.ModelName = videoLog.FallbackModel.Name
	}

	if videoLog.Key != nil {
		video.Key = videoLog.Key.Key
	}

	if videoLog.VideoRes.Error != nil {

		video.ErrMsg = videoLog.VideoRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(videoLog.VideoRes.Error, &openaiApiError) {
			video.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(videoLog.VideoRes.Error) {
			video.Status = 2
		} else {
			video.Status = -1
		}
	}

	if videoLog.RetryInfo != nil {

		video.IsRetry = videoLog.RetryInfo.IsRetry
		video.Retry = &mcommon.Retry{
			IsRetry:    videoLog.RetryInfo.IsRetry,
			RetryCount: videoLog.RetryInfo.RetryCount,
			ErrMsg:     videoLog.RetryInfo.ErrMsg,
		}

		if video.IsRetry {
			video.Status = 3
			video.ErrMsg = videoLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogVideo.Insert(ctx, video); err != nil {
		logger.Errorf(ctx, "sLog Video error: %v", err)

		if err.Error() == "an inserted document is too large" {
			videoLog.VideoReq.RequestData = map[string]any{"error": err.Error()}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Video retry: %d", len(retry))

		s.Video(ctx, videoLog, retry...)
	}
}

// 文件日志
func (s *sLog) File(ctx context.Context, fileLog model.LogFile, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog File time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if fileLog.FileRes.Error != nil && checkError(fileLog.FileRes.Error) {
		return
	}

	file := do.LogFile{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Action:       fileLog.FileReq.Action,
		FileId:       fileLog.FileRes.FileId,
		RequestData:  fileLog.FileReq.RequestData,
		ResponseData: fileLog.FileRes.ResponseData,
		Spend:        fileLog.Spend,
		TotalTime:    fileLog.FileRes.TotalTime,
		InternalTime: fileLog.FileRes.InternalTime,
		ReqTime:      fileLog.FileRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(fileLog.FileRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if fileLog.ReqModel != nil {
		file.ProviderId = fileLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, fileLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			file.ProviderName = provider.Name
		}
		file.ModelId = fileLog.ReqModel.Id
		file.ModelName = fileLog.ReqModel.Name
		file.Model = fileLog.ReqModel.Model
		file.ModelType = fileLog.ReqModel.Type
	}

	if fileLog.RealModel != nil {
		file.IsEnablePresetConfig = fileLog.RealModel.IsEnablePresetConfig
		file.PresetConfig = fileLog.RealModel.PresetConfig
		file.IsEnableForward = fileLog.RealModel.IsEnableForward
		file.ForwardConfig = fileLog.RealModel.ForwardConfig
		file.IsEnableModelAgent = fileLog.RealModel.IsEnableModelAgent
		file.RealModelId = fileLog.RealModel.Id
		file.RealModelName = fileLog.RealModel.Name
		file.RealModel = fileLog.RealModel.Model
	}

	if file.IsEnableModelAgent && fileLog.ModelAgent != nil {
		file.ModelAgentId = fileLog.ModelAgent.Id
		file.ModelAgent = &do.ModelAgent{
			ProviderId: fileLog.ModelAgent.ProviderId,
			Name:       fileLog.ModelAgent.Name,
			BaseUrl:    fileLog.ModelAgent.BaseUrl,
			Path:       fileLog.ModelAgent.Path,
			Weight:     fileLog.ModelAgent.Weight,
			Remark:     fileLog.ModelAgent.Remark,
			Status:     fileLog.ModelAgent.Status,
		}
	}

	if fileLog.FallbackModelAgent != nil {
		file.IsEnableFallback = true
		file.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     fileLog.FallbackModelAgent.Id,
			ModelAgentName: fileLog.FallbackModelAgent.Name,
		}
	}

	if fileLog.FallbackModel != nil {
		file.IsEnableFallback = true
		if file.FallbackConfig == nil {
			file.FallbackConfig = new(mcommon.FallbackConfig)
		}
		file.FallbackConfig.Model = fileLog.FallbackModel.Model
		file.FallbackConfig.ModelName = fileLog.FallbackModel.Name
	}

	if fileLog.Key != nil {
		file.Key = fileLog.Key.Key
	}

	if fileLog.FileRes.Error != nil {

		file.ErrMsg = fileLog.FileRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(fileLog.FileRes.Error, &openaiApiError) {
			file.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(fileLog.FileRes.Error) {
			file.Status = 2
		} else {
			file.Status = -1
		}
	}

	if fileLog.RetryInfo != nil {

		file.IsRetry = fileLog.RetryInfo.IsRetry
		file.Retry = &mcommon.Retry{
			IsRetry:    fileLog.RetryInfo.IsRetry,
			RetryCount: fileLog.RetryInfo.RetryCount,
			ErrMsg:     fileLog.RetryInfo.ErrMsg,
		}

		if file.IsRetry {
			file.Status = 3
			file.ErrMsg = fileLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogFile.Insert(ctx, file); err != nil {
		logger.Errorf(ctx, "sLog File error: %v", err)

		if err.Error() == "an inserted document is too large" {
			fileLog.FileReq.RequestData = map[string]any{"error": err.Error()}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog File retry: %d", len(retry))

		s.File(ctx, fileLog, retry...)
	}
}

// 批处理日志
func (s *sLog) Batch(ctx context.Context, batchLog model.LogBatch, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Batch time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if batchLog.BatchRes.Error != nil && checkError(batchLog.BatchRes.Error) {
		return
	}

	batch := do.LogBatch{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Action:       batchLog.BatchReq.Action,
		BatchId:      batchLog.BatchRes.BatchId,
		RequestData:  batchLog.BatchReq.RequestData,
		ResponseData: batchLog.BatchRes.ResponseData,
		Spend:        batchLog.Spend,
		TotalTime:    batchLog.BatchRes.TotalTime,
		InternalTime: batchLog.BatchRes.InternalTime,
		ReqTime:      batchLog.BatchRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(batchLog.BatchRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if batchLog.ReqModel != nil {
		batch.ProviderId = batchLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, batchLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			batch.ProviderName = provider.Name
		}
		batch.ModelId = batchLog.ReqModel.Id
		batch.ModelName = batchLog.ReqModel.Name
		batch.Model = batchLog.ReqModel.Model
		batch.ModelType = batchLog.ReqModel.Type
	}

	if batchLog.RealModel != nil {
		batch.IsEnablePresetConfig = batchLog.RealModel.IsEnablePresetConfig
		batch.PresetConfig = batchLog.RealModel.PresetConfig
		batch.IsEnableForward = batchLog.RealModel.IsEnableForward
		batch.ForwardConfig = batchLog.RealModel.ForwardConfig
		batch.IsEnableModelAgent = batchLog.RealModel.IsEnableModelAgent
		batch.RealModelId = batchLog.RealModel.Id
		batch.RealModelName = batchLog.RealModel.Name
		batch.RealModel = batchLog.RealModel.Model
	}

	if batch.IsEnableModelAgent && batchLog.ModelAgent != nil {
		batch.ModelAgentId = batchLog.ModelAgent.Id
		batch.ModelAgent = &do.ModelAgent{
			ProviderId: batchLog.ModelAgent.ProviderId,
			Name:       batchLog.ModelAgent.Name,
			BaseUrl:    batchLog.ModelAgent.BaseUrl,
			Path:       batchLog.ModelAgent.Path,
			Weight:     batchLog.ModelAgent.Weight,
			Remark:     batchLog.ModelAgent.Remark,
			Status:     batchLog.ModelAgent.Status,
		}
	}

	if batchLog.FallbackModelAgent != nil {
		batch.IsEnableFallback = true
		batch.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     batchLog.FallbackModelAgent.Id,
			ModelAgentName: batchLog.FallbackModelAgent.Name,
		}
	}

	if batchLog.FallbackModel != nil {
		batch.IsEnableFallback = true
		if batch.FallbackConfig == nil {
			batch.FallbackConfig = new(mcommon.FallbackConfig)
		}
		batch.FallbackConfig.Model = batchLog.FallbackModel.Model
		batch.FallbackConfig.ModelName = batchLog.FallbackModel.Name
	}

	if batchLog.Key != nil {
		batch.Key = batchLog.Key.Key
	}

	if batchLog.BatchRes.Error != nil {

		batch.ErrMsg = batchLog.BatchRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(batchLog.BatchRes.Error, &openaiApiError) {
			batch.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(batchLog.BatchRes.Error) {
			batch.Status = 2
		} else {
			batch.Status = -1
		}
	}

	if batchLog.RetryInfo != nil {

		batch.IsRetry = batchLog.RetryInfo.IsRetry
		batch.Retry = &mcommon.Retry{
			IsRetry:    batchLog.RetryInfo.IsRetry,
			RetryCount: batchLog.RetryInfo.RetryCount,
			ErrMsg:     batchLog.RetryInfo.ErrMsg,
		}

		if batch.IsRetry {
			batch.Status = 3
			batch.ErrMsg = batchLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogBatch.Insert(ctx, batch); err != nil {
		logger.Errorf(ctx, "sLog Batch error: %v", err)

		if err.Error() == "an inserted document is too large" {
			batchLog.BatchReq.RequestData = map[string]any{"error": err.Error()}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Batch retry: %d", len(retry))

		s.Batch(ctx, batchLog, retry...)
	}
}

// Midjourney日志
func (s *sLog) Midjourney(ctx context.Context, midjourneyLog model.LogMidjourney, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog Midjourney time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if midjourneyLog.Response.Error != nil && checkError(midjourneyLog.Response.Error) {
		return
	}

	midjourney := do.LogMidjourney{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		ReqUrl:       midjourneyLog.Response.ReqUrl,
		TaskId:       midjourneyLog.Response.TaskId,
		Action:       midjourneyLog.Response.Action,
		Prompt:       midjourneyLog.Response.Prompt,
		PromptEn:     midjourneyLog.Response.PromptEn,
		ImageUrl:     midjourneyLog.Response.ImageUrl,
		Progress:     midjourneyLog.Response.Progress,
		Spend:        midjourneyLog.Spend,
		ConnTime:     midjourneyLog.Response.ConnTime,
		Duration:     midjourneyLog.Response.Duration,
		TotalTime:    midjourneyLog.Response.TotalTime,
		InternalTime: midjourneyLog.Response.InternalTime,
		ReqTime:      midjourneyLog.Response.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(midjourneyLog.Response.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if midjourneyLog.ReqModel != nil {
		midjourney.ProviderId = midjourneyLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, midjourneyLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			midjourney.ProviderName = provider.Name
		}
		midjourney.ModelId = midjourneyLog.ReqModel.Id
		midjourney.ModelName = midjourneyLog.ReqModel.Name
		midjourney.Model = midjourneyLog.ReqModel.Model
		midjourney.ModelType = midjourneyLog.ReqModel.Type
	}

	if midjourneyLog.RealModel != nil {
		midjourney.IsEnablePresetConfig = midjourneyLog.RealModel.IsEnablePresetConfig
		midjourney.PresetConfig = midjourneyLog.RealModel.PresetConfig
		midjourney.IsEnableForward = midjourneyLog.RealModel.IsEnableForward
		midjourney.ForwardConfig = midjourneyLog.RealModel.ForwardConfig
		midjourney.IsEnableModelAgent = midjourneyLog.RealModel.IsEnableModelAgent
		midjourney.RealModelId = midjourneyLog.RealModel.Id
		midjourney.RealModelName = midjourneyLog.RealModel.Name
		midjourney.RealModel = midjourneyLog.RealModel.Model
	}

	if midjourney.IsEnableModelAgent && midjourneyLog.ModelAgent != nil {
		midjourney.ModelAgentId = midjourneyLog.ModelAgent.Id
		midjourney.ModelAgent = &do.ModelAgent{
			ProviderId: midjourneyLog.ModelAgent.ProviderId,
			Name:       midjourneyLog.ModelAgent.Name,
			BaseUrl:    midjourneyLog.ModelAgent.BaseUrl,
			Path:       midjourneyLog.ModelAgent.Path,
			Weight:     midjourneyLog.ModelAgent.Weight,
			Remark:     midjourneyLog.ModelAgent.Remark,
		}
	}

	if midjourneyLog.FallbackModelAgent != nil {
		midjourney.IsEnableFallback = true
		midjourney.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     midjourneyLog.FallbackModelAgent.Id,
			ModelAgentName: midjourneyLog.FallbackModelAgent.Name,
		}
	}

	if midjourneyLog.FallbackModel != nil {
		midjourney.IsEnableFallback = true
		if midjourney.FallbackConfig == nil {
			midjourney.FallbackConfig = new(mcommon.FallbackConfig)
		}
		midjourney.FallbackConfig.Model = midjourneyLog.FallbackModel.Model
		midjourney.FallbackConfig.ModelName = midjourneyLog.FallbackModel.Name
	}

	if midjourneyLog.Key != nil {
		midjourney.Key = midjourneyLog.Key.Key
	}

	if midjourneyLog.Response.Response != nil {
		if err := gjson.Unmarshal(midjourneyLog.Response.Response, &midjourney.Response); err != nil {
			logger.Error(ctx, err)
		}
	}

	if midjourneyLog.Response.Error != nil {
		midjourney.ErrMsg = midjourneyLog.Response.Error.Error()
		if common.IsAborted(midjourneyLog.Response.Error) {
			midjourney.Status = 2
		} else {
			midjourney.Status = -1
		}
	}

	if midjourneyLog.RetryInfo != nil {

		midjourney.IsRetry = midjourneyLog.RetryInfo.IsRetry
		midjourney.Retry = &mcommon.Retry{
			IsRetry:    midjourneyLog.RetryInfo.IsRetry,
			RetryCount: midjourneyLog.RetryInfo.RetryCount,
			ErrMsg:     midjourneyLog.RetryInfo.ErrMsg,
		}

		if midjourney.IsRetry {
			midjourney.Status = 3
			midjourney.ErrMsg = midjourneyLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogMidjourney.Insert(ctx, midjourney); err != nil {
		logger.Errorf(ctx, "sLog Midjourney error: %v", err)

		if err.Error() == "an inserted document is too large" {
			midjourneyLog.Response.Prompt = err.Error()
			midjourneyLog.Response.PromptEn = err.Error()
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog Midjourney retry: %d", len(retry))

		s.Midjourney(ctx, midjourneyLog, retry...)
	}
}

// 通用日志
func (s *sLog) General(ctx context.Context, generalLog model.LogGeneral, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLog General time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if generalLog.GeneralRes.Error != nil && checkError(generalLog.GeneralRes.Error) {
		return
	}

	general := do.LogGeneral{
		TraceId:      gtrace.GetTraceID(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		RequestData:  generalLog.GeneralReq.RequestData,
		ResponseData: generalLog.GeneralRes.ResponseData,
		Stream:       generalLog.GeneralReq.Stream,
		Completion:   generalLog.GeneralRes.Completion,
		Spend:        generalLog.Spend,
		ConnTime:     generalLog.GeneralRes.ConnTime,
		Duration:     generalLog.GeneralRes.Duration,
		TotalTime:    generalLog.GeneralRes.TotalTime,
		InternalTime: generalLog.GeneralRes.InternalTime,
		ReqTime:      generalLog.GeneralRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(generalLog.GeneralRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if generalLog.ReqModel != nil {
		general.ProviderId = generalLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, generalLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			general.ProviderName = provider.Name
		}
		general.ModelId = generalLog.ReqModel.Id
		general.ModelName = generalLog.ReqModel.Name
		general.Model = generalLog.ReqModel.Model
		general.ModelType = generalLog.ReqModel.Type
	}

	if generalLog.RealModel != nil {
		general.IsEnablePresetConfig = generalLog.RealModel.IsEnablePresetConfig
		general.PresetConfig = generalLog.RealModel.PresetConfig
		general.IsEnableForward = generalLog.RealModel.IsEnableForward
		general.ForwardConfig = generalLog.RealModel.ForwardConfig
		general.IsEnableModelAgent = generalLog.RealModel.IsEnableModelAgent
		general.RealModelId = generalLog.RealModel.Id
		general.RealModelName = generalLog.RealModel.Name
		general.RealModel = generalLog.RealModel.Model
	}

	if general.IsEnableModelAgent && generalLog.ModelAgent != nil {
		general.ModelAgentId = generalLog.ModelAgent.Id
		general.ModelAgent = &do.ModelAgent{
			ProviderId: generalLog.ModelAgent.ProviderId,
			Name:       generalLog.ModelAgent.Name,
			BaseUrl:    generalLog.ModelAgent.BaseUrl,
			Path:       generalLog.ModelAgent.Path,
			Weight:     generalLog.ModelAgent.Weight,
			Remark:     generalLog.ModelAgent.Remark,
			Status:     generalLog.ModelAgent.Status,
		}
	}

	if generalLog.FallbackModelAgent != nil {
		general.IsEnableFallback = true
		general.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     generalLog.FallbackModelAgent.Id,
			ModelAgentName: generalLog.FallbackModelAgent.Name,
		}
	}

	if generalLog.FallbackModel != nil {
		general.IsEnableFallback = true
		if general.FallbackConfig == nil {
			general.FallbackConfig = new(mcommon.FallbackConfig)
		}
		general.FallbackConfig.Model = generalLog.FallbackModel.Model
		general.FallbackConfig.ModelName = generalLog.FallbackModel.Name
	}

	if generalLog.Key != nil {
		general.Key = generalLog.Key.Key
	}

	if generalLog.GeneralRes.Error != nil {

		general.ErrMsg = generalLog.GeneralRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(generalLog.GeneralRes.Error, &openaiApiError) {
			general.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(generalLog.GeneralRes.Error) {
			general.Status = 2
		} else {
			general.Status = -1
		}
	}

	if generalLog.RetryInfo != nil {

		general.IsRetry = generalLog.RetryInfo.IsRetry
		general.Retry = &mcommon.Retry{
			IsRetry:    generalLog.RetryInfo.IsRetry,
			RetryCount: generalLog.RetryInfo.RetryCount,
			ErrMsg:     generalLog.RetryInfo.ErrMsg,
		}

		if general.IsRetry {
			general.Status = 3
			general.ErrMsg = generalLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.LogGeneral.Insert(ctx, general); err != nil {
		logger.Errorf(ctx, "sLog General error: %v", err)

		if err.Error() == "an inserted document is too large" {
			generalLog.GeneralReq.RequestData = map[string]any{"error": err.Error()}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLog General retry: %d", len(retry))

		s.General(ctx, generalLog, retry...)
	}
}

func checkError(err error) bool {
	return errors.Is(err, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(err, errors.ERR_MODEL_DISABLED) ||
		errors.Is(err, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(err, errors.ERR_GROUP_DISABLED) ||
		errors.Is(err, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(err, errors.ERR_GROUP_INSUFFICIENT_QUOTA)
}
