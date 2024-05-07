package image

import (
	"context"
	"fmt"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/grpool"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi-sdk"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi-sdk/sdkerr"
	"github.com/iimeta/fastapi/internal/config"
	"github.com/iimeta/fastapi/internal/dao"
	"github.com/iimeta/fastapi/internal/errors"
	"github.com/iimeta/fastapi/internal/model"
	"github.com/iimeta/fastapi/internal/model/do"
	"github.com/iimeta/fastapi/internal/service"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

type sImage struct{}

func init() {
	service.RegisterImage(New())
}

func New() service.IImage {
	return &sImage{}
}

// Generations
func (s *sImage) Generations(ctx context.Context, params sdkm.ImageRequest, retry ...int) (response sdkm.ImageResponse, err error) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage Generations time: %d", gtime.TimestampMilli()-now)
	}()

	var m *model.Model
	var key *model.Key
	var modelAgent *model.ModelAgent
	var baseUrl string
	var path string
	var keyTotal int
	var isRetry bool

	defer func() {

		// 不记录重试
		if isRetry {
			return
		}

		enterTime := g.RequestFromCtx(ctx).EnterTime.TimestampMilli()
		internalTime := gtime.TimestampMilli() - enterTime - response.TotalTime
		usage := &sdkm.Usage{
			TotalTokens: m.FixedQuota,
		}

		if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

			if err == nil {
				if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {
					if err := service.Common().RecordUsage(ctx, m, usage); err != nil {
						logger.Error(ctx, err)
					}
				}, nil); err != nil {
					logger.Error(ctx, err)
				}
			}

			if err := grpool.AddWithRecover(ctx, func(ctx context.Context) {

				m.ModelAgent = modelAgent

				imageRes := &model.ImageRes{
					Created:      response.Created,
					Data:         response.Data,
					Usage:        usage,
					TotalTime:    response.TotalTime,
					Error:        err,
					InternalTime: internalTime,
					EnterTime:    enterTime,
				}

				s.SaveChat(ctx, m, key, &params, imageRes)

			}, nil); err != nil {
				logger.Error(ctx, err)
			}

		}, nil); err != nil {
			logger.Error(ctx, err)
		}
	}()

	if m, err = service.Model().GetModelBySecretKey(ctx, params.Model, service.Session().GetSecretKey(ctx)); err != nil {
		logger.Error(ctx, err)
		return response, err
	}

	if m.IsEnableModelAgent {

		if _, modelAgent, err = service.ModelAgent().PickModelAgent(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}

		if modelAgent != nil {

			baseUrl = modelAgent.BaseUrl
			path = modelAgent.Path

			if keyTotal, key, err = service.ModelAgent().PickModelAgentKey(ctx, modelAgent); err != nil {
				service.ModelAgent().RecordErrorModelAgent(ctx, m, modelAgent)
				logger.Error(ctx, err)
				return response, err
			}
		}

	} else {
		if keyTotal, key, err = service.Key().PickModelKey(ctx, m); err != nil {
			logger.Error(ctx, err)
			return response, err
		}
	}

	request := params
	request.Model = m.Model

	client := sdk.NewClient(ctx, m.Corp, m.Model, key.Key, baseUrl, path, config.Cfg.Http.ProxyUrl)
	if response, err = client.Image(ctx, request); err != nil {
		logger.Error(ctx, err)

		if len(retry) > 0 {
			if config.Cfg.Api.Retry > 0 && len(retry) == config.Cfg.Api.Retry {
				return response, err
			} else if config.Cfg.Api.Retry < 0 && len(retry) == keyTotal {
				return response, err
			} else if config.Cfg.Api.Retry == 0 {
				return response, err
			}
		}

		e := &sdkerr.APIError{}
		if errors.As(err, &e) {

			isRetry = true
			service.Common().RecordError(ctx, m, key, modelAgent)

			switch e.HTTPStatusCode {
			case 400:

				if gstr.Contains(err.Error(), "Please reduce the length of the messages") {
					return response, err
				}

				response, err = s.Generations(ctx, params, append(retry, 1)...)

			case 429:

				if gstr.Contains(err.Error(), "You exceeded your current quota") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
						}

					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}

				response, err = s.Generations(ctx, params, append(retry, 1)...)

			default:

				if gstr.Contains(err.Error(), "Incorrect API key provided") {
					if err := grpool.AddWithRecover(gctx.NeverDone(ctx), func(ctx context.Context) {

						if m.IsEnableModelAgent {
							service.ModelAgent().DisabledModelAgentKey(ctx, key)
						} else {
							service.Key().DisabledModelKey(ctx, key)
						}

					}, nil); err != nil {
						logger.Error(ctx, err)
					}
				}

				response, err = s.Generations(ctx, params, append(retry, 1)...)
			}

			return response, err
		}

		return response, err
	}

	return response, nil
}

// 保存文生图聊天数据
func (s *sImage) SaveChat(ctx context.Context, model *model.Model, key *model.Key, imageReq *sdkm.ImageRequest, imageRes *model.ImageRes) {

	now := gtime.TimestampMilli()
	defer func() {
		logger.Debugf(ctx, "sImage SaveChat time: %d", gtime.TimestampMilli()-now)
	}()

	completion := ""
	for i, data := range imageRes.Data {

		if len(completion) > 0 {
			completion += "\n\n"
		}

		completion += fmt.Sprintf("%d. %s", i+1, data.URL)
	}

	chat := do.Chat{
		TraceId:      gctx.CtxId(ctx),
		UserId:       service.Session().GetUserId(ctx),
		AppId:        service.Session().GetAppId(ctx),
		Prompt:       imageReq.Prompt,
		Completion:   completion,
		ConnTime:     imageRes.ConnTime,
		Duration:     imageRes.Duration,
		TotalTime:    imageRes.TotalTime,
		InternalTime: imageRes.InternalTime,
		ReqTime:      imageRes.EnterTime,
		ReqDate:      gtime.NewFromTimeStamp(imageRes.EnterTime).Format("Y-m-d"),
		ClientIp:     g.RequestFromCtx(ctx).GetClientIp(),
		RemoteIp:     g.RequestFromCtx(ctx).GetRemoteIp(),
		LocalIp:      util.GetLocalIp(),
		Status:       1,
	}

	if model != nil {

		chat.Corp = model.Corp
		chat.ModelId = model.Id
		chat.Name = model.Name
		chat.Model = model.Model
		chat.Type = model.Type
		chat.BillingMethod = model.BillingMethod
		chat.PromptRatio = model.PromptRatio
		chat.CompletionRatio = model.CompletionRatio
		chat.FixedQuota = model.FixedQuota
		chat.IsEnableModelAgent = model.IsEnableModelAgent
		if chat.IsEnableModelAgent && model.ModelAgent != nil {
			chat.ModelAgentId = model.ModelAgent.Id
			chat.ModelAgent = &do.ModelAgent{
				Name:    model.ModelAgent.Name,
				BaseUrl: model.ModelAgent.BaseUrl,
				Path:    model.ModelAgent.Path,
				Weight:  model.ModelAgent.Weight,
				Remark:  model.ModelAgent.Remark,
				Status:  model.ModelAgent.Status,
			}
		}

		if imageRes.Error == nil {
			chat.TotalTokens = imageRes.Usage.TotalTokens
		}
	}

	if key != nil {
		chat.Key = key.Key
	}

	if imageRes.Error != nil {
		chat.ErrMsg = imageRes.Error.Error()
		chat.Status = -1
	}

	if _, err := dao.Chat.Insert(ctx, chat); err != nil {
		logger.Error(ctx, err)
	}
}
