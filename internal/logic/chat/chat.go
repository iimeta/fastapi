package chat

import (
	"context"
	"fmt"
	"io"
	"math"
	"slices"
	"time"

	"github.com/gogf/gf/v2/container/gmap"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
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

type sChat struct{}

func init() {
	service.RegisterChat(New())
}

func New() service.IChat {
	return &sChat{}
}

// Completions
func (s *sChat) Completions(ctx context.Context, params smodel.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (response smodel.ChatCompletionResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat Completions time: %d", gtime.TimestampMilli()-now)
	}()

	if len(params.Functions) == 0 {
		params.Messages = common.HandleMessages(params.Messages)
		if len(params.Messages) == 0 {
			return response, errors.ERR_INVALID_PARAMETER
		}
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		retryInfo   *mcommon.Retry
		totalTokens int
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime

		if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

			// 替换成调用的模型
			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			completion := ""
			if len(response.Choices) > 0 && response.Choices[0].Message != nil {
				for _, choice := range response.Choices {
					completion += gconv.String(choice.Message.Content)
				}
			}

			usageSpend := common.ChatUsageSpend(ctx, params, completion, response.Usage, mak)
			totalTokens = usageSpend.TotalTokens
			response.Usage = usageSpend.Usage

			// 分组折扣
			if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
				totalTokens = int(math.Ceil(float64(totalTokens) * mak.Group.Discount))
			}

			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
				if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key, mak.Group); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}); err != nil {
				logger.Error(ctx, err)
			}
		}

		if mak.ReqModel != nil && mak.RealModel != nil {
			if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

				completionsRes := &model.CompletionsRes{
					Error:        err,
					ConnTime:     response.ConnTime,
					Duration:     response.Duration,
					TotalTime:    response.TotalTime,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				if retryInfo == nil && response.Usage != nil {

					if response.Usage.PromptTokensDetails.CachedTokens != 0 {
						response.Usage.CacheCreationInputTokens = response.Usage.PromptTokensDetails.CachedTokens
					}

					if response.Usage.CompletionTokensDetails.CachedTokens != 0 {
						response.Usage.CacheReadInputTokens = response.Usage.CompletionTokensDetails.CachedTokens
					}

					completionsRes.Usage = *response.Usage
					completionsRes.Usage.TotalTokens = totalTokens
				}

				if retryInfo == nil && len(response.Choices) > 0 && response.Choices[0].Message != nil {
					if mak.RealModel.Type == 102 && response.Choices[0].Message.Audio != nil {
						completionsRes.Completion = response.Choices[0].Message.Audio.Transcript
					} else {
						if len(response.Choices) > 1 {
							for i, choice := range response.Choices {

								if choice.Message.Content != nil {
									completionsRes.Completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, gconv.String(choice.Message.Content))
								}

								if choice.Message.ToolCalls != nil {
									completionsRes.Completion += fmt.Sprintf("index: %d\ntool_calls: %s\n\n", i, gconv.String(choice.Message.ToolCalls))
								}
							}
						} else {

							if response.Choices[0].Message.ReasoningContent != nil {
								completionsRes.Completion = gconv.String(response.Choices[0].Message.ReasoningContent)
							}

							completionsRes.Completion += gconv.String(response.Choices[0].Message.Content)

							if response.Choices[0].Message.ToolCalls != nil {
								completionsRes.Completion += fmt.Sprintf("\ntool_calls: %s", gconv.String(response.Choices[0].Message.ToolCalls))
							}
						}
					}
				}

				s.SaveLog(ctx, model.ChatLog{
					Group:              mak.Group,
					ReqModel:           mak.ReqModel,
					RealModel:          mak.RealModel,
					ModelAgent:         mak.ModelAgent,
					FallbackModelAgent: fallbackModelAgent,
					FallbackModel:      fallbackModel,
					Key:                mak.Key,
					CompletionsReq:     &params,
					CompletionsRes:     completionsRes,
					RetryInfo:          retryInfo,
				})

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

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if request.Messages[0].Role == sconsts.ROLE_SYSTEM {
				request.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages[1:]...)
			} else {
				request.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if request.MaxTokens != 0 {
			if mak.RealModel.PresetConfig.MinTokens != 0 && request.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MinTokens
			} else if mak.RealModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
			}
		}
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sChat Completions request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	response, err = common.NewAdapter(ctx, mak, false).ChatCompletions(ctx, request)
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
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
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

			return s.Completions(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return response, err
	}

	return response, nil
}

// CompletionsStream
func (s *sChat) CompletionsStream(ctx context.Context, params smodel.ChatCompletionRequest, fallbackModelAgent *model.ModelAgent, fallbackModel *model.Model, retry ...int) (err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat CompletionsStream time: %d", gtime.TimestampMilli()-now)
	}()

	if len(params.Functions) == 0 {
		params.Messages = common.HandleMessages(params.Messages)
		if len(params.Messages) == 0 {
			return errors.ERR_INVALID_PARAMETER
		}
	}

	var (
		mak = &common.MAK{
			Model:              params.Model,
			Messages:           params.Messages,
			FallbackModelAgent: fallbackModelAgent,
			FallbackModel:      fallbackModel,
		}
		completion  string
		connTime    int64
		duration    int64
		totalTime   int64
		totalTokens int
		usage       *smodel.Usage
		retryInfo   *mcommon.Retry
	)

	defer func() {

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - totalTime

		if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
			if retryInfo == nil && (err == nil || common.IsAborted(err)) && mak.ReqModel != nil {

				usageSpend := common.ChatUsageSpend(ctx, params, completion, usage, mak)
				totalTokens = usageSpend.TotalTokens
				usage = usageSpend.Usage

				// 分组折扣
				if mak.Group != nil && slices.Contains(mak.Group.Models, mak.ReqModel.Id) {
					totalTokens = int(math.Ceil(float64(totalTokens) * mak.Group.Discount))
				}

				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, totalTokens, mak.Key.Key, mak.Group); err != nil {
						logger.Error(ctx, err)
						panic(err)
					}
				}); err != nil {
					logger.Error(ctx, err)
				}
			}

			if mak.ReqModel != nil && mak.RealModel != nil {
				if err := grpool.Add(gctx.NeverDone(ctx), func(ctx context.Context) {

					completionsRes := &model.CompletionsRes{
						Completion:   completion,
						Error:        err,
						ConnTime:     connTime,
						Duration:     duration,
						TotalTime:    totalTime,
						InternalTime: internalTime,
						EnterTime:    enterTime,
					}

					if usage != nil {

						if usage.PromptTokensDetails.CachedTokens != 0 {
							usage.CacheCreationInputTokens = usage.PromptTokensDetails.CachedTokens
						}

						if usage.CompletionTokensDetails.CachedTokens != 0 {
							usage.CacheReadInputTokens = usage.CompletionTokensDetails.CachedTokens
						}

						completionsRes.Usage = *usage
						completionsRes.Usage.TotalTokens = totalTokens
					}

					s.SaveLog(ctx, model.ChatLog{
						Group:              mak.Group,
						ReqModel:           mak.ReqModel,
						RealModel:          mak.RealModel,
						ModelAgent:         mak.ModelAgent,
						FallbackModelAgent: fallbackModelAgent,
						FallbackModel:      fallbackModel,
						Key:                mak.Key,
						CompletionsReq:     &params,
						CompletionsRes:     completionsRes,
						RetryInfo:          retryInfo,
					})

				}); err != nil {
					logger.Error(ctx, err)
					panic(err)
				}
			}

		}); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if err = mak.InitMAK(ctx); err != nil {
		logger.Error(ctx, err)
		return err
	}

	request := params

	if !gstr.Contains(mak.RealModel.Model, "*") {
		request.Model = mak.RealModel.Model
	}

	// 预设配置
	if mak.RealModel.IsEnablePresetConfig {

		// 替换预设提示词
		if mak.RealModel.PresetConfig.IsSupportSystemRole && mak.RealModel.PresetConfig.SystemRolePrompt != "" {
			if request.Messages[0].Role == sconsts.ROLE_SYSTEM {
				request.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages[1:]...)
			} else {
				request.Messages = append([]smodel.ChatCompletionMessage{{
					Role:    sconsts.ROLE_SYSTEM,
					Content: mak.RealModel.PresetConfig.SystemRolePrompt,
				}}, request.Messages...)
			}
		}

		// 检查MaxTokens取值范围
		if request.MaxTokens != 0 {
			if mak.RealModel.PresetConfig.MinTokens != 0 && request.MaxTokens < mak.RealModel.PresetConfig.MinTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MinTokens
			} else if mak.RealModel.PresetConfig.MaxTokens != 0 && request.MaxTokens > mak.RealModel.PresetConfig.MaxTokens {
				request.MaxTokens = mak.RealModel.PresetConfig.MaxTokens
			}
		}
	}

	if mak.ModelAgent != nil && mak.ModelAgent.IsEnableModelReplace {
		for i, replaceModel := range mak.ModelAgent.ReplaceModels {
			if replaceModel == request.Model {
				logger.Infof(ctx, "sChat CompletionsStream request.Model: %s replaced %s", request.Model, mak.ModelAgent.TargetModels[i])
				request.Model = mak.ModelAgent.TargetModels[i]
				mak.RealModel.Model = request.Model
				break
			}
		}
	}

	response, err := common.NewAdapter(ctx, mak, true).ChatCompletionsStream(ctx, request)
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
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel)
						}
					}

					if mak.RealModel.FallbackConfig.Model != "" {
						if fallbackModel, _ = service.Model().GetFallbackModel(ctx, mak.RealModel); fallbackModel != nil {
							retryInfo = &mcommon.Retry{
								IsRetry:    true,
								RetryCount: len(retry),
								ErrMsg:     err.Error(),
							}
							return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, nil, fallbackModel)
						}
					}
				}

				return err
			}

			retryInfo = &mcommon.Retry{
				IsRetry:    true,
				RetryCount: len(retry),
				ErrMsg:     err.Error(),
			}

			return s.CompletionsStream(g.RequestFromCtx(ctx).GetCtx(), params, fallbackModelAgent, fallbackModel, append(retry, 1)...)
		}

		return err
	}

	defer close(response)

	for {

		response := <-response

		connTime = response.ConnTime
		duration = response.Duration
		totalTime = response.TotalTime

		if response.Error != nil {

			if errors.Is(response.Error, io.EOF) {

				if err = util.SSEServer(ctx, "[DONE]"); err != nil {
					logger.Error(ctx, err)
					return err
				}

				return nil
			}

			err = response.Error

			// 记录错误次数和禁用
			service.Common().RecordError(ctx, mak.RealModel, mak.Key, mak.ModelAgent)

			if _, isDisabled := common.IsNeedRetry(err); isDisabled {
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

			return err
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil {
			if mak.RealModel.Type == 102 && response.Choices[0].Delta.Audio != nil {
				completion += response.Choices[0].Delta.Audio.Transcript
			} else {
				if len(response.Choices) > 1 {
					for i, choice := range response.Choices {
						completion += fmt.Sprintf("index: %d\ncontent: %s\n\n", i, choice.Delta.Content)
					}
				} else {
					if response.Choices[0].Delta.ReasoningContent != nil {
						completion += gconv.String(response.Choices[0].Delta.ReasoningContent)
					}
					completion += response.Choices[0].Delta.Content
				}
			}
		}

		if len(response.Choices) > 0 && response.Choices[0].Delta != nil && response.Choices[0].Delta.ToolCalls != nil {
			completion += gconv.String(response.Choices[0].Delta.ToolCalls)
		}

		if response.Usage != nil {
			if usage == nil {
				usage = response.Usage
			} else {
				if response.Usage.PromptTokens != 0 {
					usage.PromptTokens = response.Usage.PromptTokens
				}
				if response.Usage.CompletionTokens != 0 {
					usage.CompletionTokens = response.Usage.CompletionTokens
				}
				if response.Usage.TotalTokens != 0 {
					usage.TotalTokens = response.Usage.TotalTokens
				} else {
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				}
				if response.Usage.CacheCreationInputTokens != 0 {
					usage.CacheCreationInputTokens = response.Usage.CacheCreationInputTokens
				}
				if response.Usage.CacheReadInputTokens != 0 {
					usage.CacheReadInputTokens = response.Usage.CacheReadInputTokens
				}
			}
		}

		// 官方格式
		if mak.ReqModel.ResponseDataFormat == 2 && response.ResponseBytes != nil {

			if mak.ReqModel.IsEnableForward {

				data := make(map[string]interface{})
				if err = gjson.Unmarshal(response.ResponseBytes, &data); err != nil {
					logger.Error(ctx, err)
					return err
				}

				if _, ok := data["model"]; ok {
					data["model"] = mak.ReqModel.Model
				}

				response.ResponseBytes = gjson.MustEncode(data)
			}

			if err = util.SSEServer(ctx, string(response.ResponseBytes)); err != nil {
				logger.Error(ctx, err)
				return err
			}

		} else {

			if mak.ReqModel.IsEnableForward {
				response.Model = mak.ReqModel.Model
			}

			if err = util.SSEServer(ctx, gjson.MustEncodeString(response)); err != nil {
				logger.Error(ctx, err)
				return err
			}
		}
	}
}

// 保存日志
func (s *sChat) SaveLog(ctx context.Context, chatLog model.ChatLog, retry ...int) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sChat SaveLog time: %d", gtime.TimestampMilli()-now)
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
		TraceId:          gctx.CtxId(ctx),
		UserId:           service.Session().GetUserId(ctx),
		AppId:            service.Session().GetAppId(ctx),
		IsSmartMatch:     chatLog.IsSmartMatch,
		Stream:           chatLog.CompletionsReq.Stream,
		PromptTokens:     chatLog.CompletionsRes.Usage.PromptTokens,
		CompletionTokens: chatLog.CompletionsRes.Usage.CompletionTokens,
		TotalTokens:      chatLog.CompletionsRes.Usage.TotalTokens,
		SearchTokens:     chatLog.CompletionsRes.Usage.SearchTokens,
		CacheWriteTokens: chatLog.CompletionsRes.Usage.CacheCreationInputTokens,
		CacheHitTokens:   chatLog.CompletionsRes.Usage.CacheReadInputTokens,
		ConnTime:         chatLog.CompletionsRes.ConnTime,
		Duration:         chatLog.CompletionsRes.Duration,
		TotalTime:        chatLog.CompletionsRes.TotalTime,
		InternalTime:     chatLog.CompletionsRes.InternalTime,
		ReqTime:          chatLog.CompletionsRes.EnterTime,
		ReqDate:          gtime.NewFromTimeStamp(chatLog.CompletionsRes.EnterTime).Format("Y-m-d"),
		ClientIp:         g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:         g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:          util.GetLocalIp(),
		Status:           1,
		Host:             g.RequestFromCtx(ctx).GetHost(),
		Rid:              service.Session().GetRid(ctx),
	}

	if chatLog.Group != nil {
		chat.GroupId = chatLog.Group.Id
		chat.GroupName = chatLog.Group.Name
		chat.Discount = chatLog.Group.Discount
	}

	if config.Cfg.Log.Open && len(chatLog.CompletionsReq.Messages) > 0 && slices.Contains(config.Cfg.Log.ChatRecords, "prompt") {

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

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "completion") {
		chat.Completion = chatLog.CompletionsRes.Completion
	}

	if chatLog.ReqModel != nil {
		chat.ProviderId = chatLog.ReqModel.ProviderId
		chat.ModelId = chatLog.ReqModel.Id
		chat.ModelName = chatLog.ReqModel.Name
		chat.Model = chatLog.ReqModel.Model
		chat.ModelType = chatLog.ReqModel.Type
		chat.TextQuota = chatLog.ReqModel.TextQuota
		chat.MultimodalQuota = chatLog.ReqModel.MultimodalQuota
		chat.MultimodalAudioQuota = chatLog.ReqModel.MultimodalAudioQuota

		if chatLog.ReqModel.Type == 102 {
			chat.TextQuota.BillingMethod = chatLog.ReqModel.MultimodalAudioQuota.AudioQuota.BillingMethod
			chat.TextQuota.PromptRatio = chatLog.ReqModel.MultimodalAudioQuota.AudioQuota.PromptRatio
			chat.TextQuota.CompletionRatio = chatLog.ReqModel.MultimodalAudioQuota.AudioQuota.CompletionRatio
			chat.TextQuota.FixedQuota = chatLog.ReqModel.MultimodalAudioQuota.AudioQuota.FixedQuota
		}
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

	if config.Cfg.Log.Open && slices.Contains(config.Cfg.Log.ChatRecords, "messages") {
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
		logger.Errorf(ctx, "sChat SaveLog error: %v", err)

		if err.Error() == "an inserted document is too large" {
			chatLog.CompletionsReq.Messages = []smodel.ChatCompletionMessage{{
				Role:    sconsts.ROLE_SYSTEM,
				Content: err.Error(),
			}}
		}

		if len(retry) == 10 {
			panic(err)
		}

		retry = append(retry, 1)

		time.Sleep(time.Duration(len(retry)*5) * time.Second)

		logger.Errorf(ctx, "sChat SaveLog retry: %d", len(retry))

		s.SaveLog(ctx, chatLog, retry...)
	}
}
