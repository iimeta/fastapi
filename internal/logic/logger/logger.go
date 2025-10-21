package logger

import (
	"context"
	"slices"
	"time"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
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

type sLogger struct{}

func init() {
	service.RegisterLogger(New())
}

func New() service.ILogger {
	return &sLogger{}
}

func (s *sLogger) Chat(ctx context.Context, chatLog model.ChatLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sLogger Chat time: %d", gtime.TimestampMilli()-now)
	}()

	// 不记录此错误日志
	if chatLog.CompletionsRes.Error != nil && (errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_MODEL_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_NOT_FOUND) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_DISABLED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_EXPIRED) ||
		errors.Is(chatLog.CompletionsRes.Error, errors.ERR_GROUP_INSUFFICIENT_QUOTA)) {
		return
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		IsSmartMatch: chatLog.IsSmartMatch,
		Stream:       chatLog.CompletionsReq.Stream,
		Spend:        chatLog.Spend,
		ConnTime:     chatLog.CompletionsRes.ConnTime,
		Duration:     chatLog.CompletionsRes.Duration,
		TotalTime:    chatLog.CompletionsRes.TotalTime,
		InternalTime: chatLog.CompletionsRes.InternalTime,
		ReqTime:      chatLog.CompletionsRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(chatLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
		Host:         g.RequestFromCtx(ctx).GetHost(),
		Rid:          service.Session().GetRid(ctx),
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {

		if chatLog.CompletionsReq != nil {

			if len(chatLog.CompletionsReq.Messages) > 0 {

				prompt := chatLog.CompletionsReq.Messages[len(chatLog.CompletionsReq.Messages)-1].Content

				if chatLog.ReqModel.Type == 102 {

					if slices.Contains(config.Cfg.Log.ChatRecords, "audio") {
						chat.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]interface{}); ok {

							multiContents := make([]interface{}, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]interface{}); ok {

									if content["type"] == "input_audio" {

										if inputAudio, ok := content["input_audio"].(map[string]interface{}); ok {

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

							chat.Prompt = gconv.String(multiContents)

						} else {
							chat.Prompt = gconv.String(prompt)
						}
					}

				} else {

					if slices.Contains(config.Cfg.Log.ChatRecords, "image") {
						chat.Prompt = gconv.String(prompt)
					} else {
						if multiContent, ok := prompt.([]interface{}); ok {

							multiContents := make([]interface{}, 0)

							for _, value := range multiContent {

								if content, ok := value.(map[string]interface{}); ok {

									if content["type"] == "image_url" {

										if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

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

							chat.Prompt = gconv.String(multiContents)

						} else if multiContent, ok := prompt.([]smodel.OpenAIResponsesContent); ok {

							multiContents := make([]smodel.OpenAIResponsesContent, 0)

							for _, value := range multiContent {
								if value.Type == "input_image" && !gstr.HasPrefix(value.ImageUrl, "http") {
									value.ImageUrl = "[BASE64图像数据]"
								}
								multiContents = append(multiContents, value)
							}

							chat.Prompt = gconv.String(multiContents)

						} else {
							chat.Prompt = gconv.String(prompt)
						}
					}
				}
			}

		} else if chatLog.EmbeddingReq != nil {
			chat.Prompt = gconv.String(chatLog.EmbeddingReq.Input)
		} else if chatLog.ModerationReq != nil {
			chat.Prompt = gconv.String(chatLog.ModerationReq.Input)
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") && chatLog.CompletionsRes != nil {
		chat.Completion = chatLog.CompletionsRes.Completion
	}

	if chatLog.ReqModel != nil {
		chat.ProviderId = chatLog.ReqModel.ProviderId
		if provider, err := service.Provider().GetCache(ctx, chatLog.ReqModel.ProviderId); err != nil {
			logger.Error(ctx, err)
		} else {
			chat.ProviderName = provider.Name
		}
		chat.ModelId = chatLog.ReqModel.Id
		chat.ModelName = chatLog.ReqModel.Name
		chat.Model = chatLog.ReqModel.Model
		chat.ModelType = chatLog.ReqModel.Type
	}

	if chatLog.RealModel != nil {
		chat.IsEnablePresetConfig = chatLog.RealModel.IsEnablePresetConfig
		chat.PresetConfig = chatLog.RealModel.PresetConfig
		chat.IsEnableForward = chatLog.RealModel.IsEnableForward
		chat.ForwardConfig = chatLog.RealModel.ForwardConfig
		chat.IsEnableModelAgent = chatLog.RealModel.IsEnableModelAgent
		chat.RealModelId = chatLog.RealModel.Id
		chat.RealModelName = chatLog.RealModel.Name
		chat.RealModel = chatLog.RealModel.Model
	}

	if chatLog.ModelAgent != nil {
		chat.IsEnableModelAgent = true
		chat.ModelAgentId = chatLog.ModelAgent.Id
		chat.ModelAgent = &do.ModelAgent{
			ProviderId: chatLog.ModelAgent.ProviderId,
			Name:       chatLog.ModelAgent.Name,
			BaseUrl:    chatLog.ModelAgent.BaseUrl,
			Path:       chatLog.ModelAgent.Path,
			Weight:     chatLog.ModelAgent.Weight,
			Remark:     chatLog.ModelAgent.Remark,
		}
	}

	if chatLog.FallbackModelAgent != nil {
		chat.IsEnableFallback = true
		chat.FallbackConfig = &mcommon.FallbackConfig{
			ModelAgent:     chatLog.FallbackModelAgent.Id,
			ModelAgentName: chatLog.FallbackModelAgent.Name,
		}
	}

	if chatLog.FallbackModel != nil {
		chat.IsEnableFallback = true
		if chat.FallbackConfig == nil {
			chat.FallbackConfig = new(mcommon.FallbackConfig)
		}
		chat.FallbackConfig.Model = chatLog.FallbackModel.Model
		chat.FallbackConfig.ModelName = chatLog.FallbackModel.Name
	}

	if chatLog.Key != nil {
		chat.Key = chatLog.Key.Key
	}

	if chatLog.CompletionsRes.Error != nil {

		chat.ErrMsg = chatLog.CompletionsRes.Error.Error()
		openaiApiError := &serrors.ApiError{}
		if errors.As(chatLog.CompletionsRes.Error, &openaiApiError) {
			chat.ErrMsg = openaiApiError.Message
		}

		if common.IsAborted(chatLog.CompletionsRes.Error) {
			chat.Status = 2
		} else {
			chat.Status = -1
		}
	}

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "messages") && chatLog.CompletionsReq != nil {
		for _, message := range chatLog.CompletionsReq.Messages {

			content := message.Content

			if !slices.Contains(config.Cfg.Log.ChatRecords, "image") {

				if multiContent, ok := content.([]interface{}); ok {

					multiContents := make([]interface{}, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "image_url" {

								if imageUrl, ok := content["image_url"].(map[string]interface{}); ok {

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

			if !slices.Contains(config.Cfg.Log.ChatRecords, "audio") {

				if multiContent, ok := content.([]interface{}); ok {

					multiContents := make([]interface{}, 0)

					for _, value := range multiContent {

						if content, ok := value.(map[string]interface{}); ok {

							if content["type"] == "input_audio" {

								if inputAudio, ok := content["input_audio"].(map[string]interface{}); ok {

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

			chat.Messages = append(chat.Messages, mcommon.Message{
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

	if chatLog.RetryInfo != nil {

		chat.IsRetry = chatLog.RetryInfo.IsRetry
		chat.Retry = &mcommon.Retry{
			IsRetry:    chatLog.RetryInfo.IsRetry,
			RetryCount: chatLog.RetryInfo.RetryCount,
			ErrMsg:     chatLog.RetryInfo.ErrMsg,
		}

		if chat.IsRetry {
			chat.Status = 3
			chat.ErrMsg = chatLog.RetryInfo.ErrMsg
		}
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Errorf(ctx, "sLogger Chat error: %v", err)

		if err.Error() == "an inserted document is too large" {
			if chatLog.CompletionsReq != nil {
				chatLog.CompletionsReq.Messages = []smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: err.Error(),
				}}
			} else if chatLog.EmbeddingReq != nil {
				chatLog.EmbeddingReq.Input = err.Error()
			} else if chatLog.ModerationReq != nil {
				chatLog.ModerationReq.Input = err.Error()
			}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sLogger Chat retry: %d", len(retry))

		s.Chat(ctx, chatLog, retry...)
	}
}
