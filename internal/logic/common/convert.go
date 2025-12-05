package common

import (
	"context"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/util/gconv"
	smodel "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/internal/consts"
	"github.com/iimeta/fastapi/utility/logger"
	"github.com/iimeta/fastapi/utility/util"
)

// 转换额度单位
func ConvQuota(quota int, n ...int) float64 {

	if quota == 0 {
		return 0.0
	}

	if len(n) == 0 {
		n = []int{6}
	}

	return util.Round(float64(quota)/consts.QUOTA_DEFAULT_UNIT, n[0])
}

func ConvResponsesToChatCompletionsRequest(request *ghttp.Request, isChatCompletions bool) smodel.ChatCompletionRequest {

	if isChatCompletions {
		chatCompletionRequest := smodel.ChatCompletionRequest{}
		if err := gjson.Unmarshal(request.GetBody(), &chatCompletionRequest); err != nil {
			logger.Error(request.GetCtx(), err)
			return smodel.ChatCompletionRequest{}
		}
		return chatCompletionRequest
	}

	responsesReq := smodel.OpenAIResponsesReq{}
	if err := gjson.Unmarshal(request.GetBody(), &responsesReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return smodel.ChatCompletionRequest{}
	}

	chatCompletionRequest := smodel.ChatCompletionRequest{
		Model:               responsesReq.Model,
		MaxCompletionTokens: responsesReq.MaxOutputTokens,
		Temperature:         responsesReq.Temperature,
		TopP:                responsesReq.TopP,
		Stream:              responsesReq.Stream,
		User:                responsesReq.User,
		Tools:               responsesReq.Tools,
		ToolChoice:          responsesReq.ToolChoice,
		ParallelToolCalls:   responsesReq.ParallelToolCalls,
		Store:               responsesReq.Store,
		Metadata:            responsesReq.Metadata,
		ServiceTier:         responsesReq.ServiceTier,
	}

	if responsesReq.Input != nil {
		if value, ok := responsesReq.Input.([]any); ok {

			inputs := make([]smodel.OpenAIResponsesInput, 0)
			if err := gjson.Unmarshal(gjson.MustEncode(value), &inputs); err != nil {
				logger.Error(request.GetCtx(), err)
				return chatCompletionRequest
			}

			for _, input := range inputs {
				chatCompletionRequest.Messages = append(chatCompletionRequest.Messages, smodel.ChatCompletionMessage{
					Role:    input.Role,
					Content: input.Content,
				})
			}

		} else {
			chatCompletionRequest.Messages = []smodel.ChatCompletionMessage{{
				Role:    "user",
				Content: responsesReq.Input,
			}}
		}
	}

	if responsesReq.Reasoning != nil {
		chatCompletionRequest.ReasoningEffort = responsesReq.Reasoning.Effort
	}

	return chatCompletionRequest
}

func ConvResponsesToChatCompletionsResponse(ctx context.Context, res smodel.OpenAIResponsesRes) smodel.ChatCompletionResponse {

	responsesRes := smodel.OpenAIResponsesRes{
		Model:         res.Model,
		Usage:         res.Usage,
		ResponseBytes: res.ResponseBytes,
		ConnTime:      res.ConnTime,
		Duration:      res.Duration,
		TotalTime:     res.TotalTime,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &responsesRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := smodel.ChatCompletionResponse{
		Id:            responsesRes.Id,
		Object:        responsesRes.Object,
		Created:       responsesRes.CreatedAt,
		Model:         responsesRes.Model,
		ServiceTier:   responsesRes.ServiceTier,
		ResponseBytes: responsesRes.ResponseBytes,
		ConnTime:      responsesRes.ConnTime,
		Duration:      responsesRes.Duration,
		TotalTime:     responsesRes.TotalTime,
		Error:         responsesRes.Err,
	}

	for _, output := range responsesRes.Output {
		if len(output.Content) > 0 {
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
				Message: &smodel.ChatCompletionMessage{
					Role:    output.Role,
					Content: output.Content[0].Text,
				},
				FinishReason: "stop",
			})
		}
	}

	if responsesRes.Tools != nil && len(gconv.Interfaces(responsesRes.Tools)) > 0 {
		chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, smodel.ChatCompletionChoice{
			Message: &smodel.ChatCompletionMessage{
				Role:      "assistant",
				ToolCalls: responsesRes.Tools,
			},
			FinishReason: "tool_calls",
		})
	}

	if responsesRes.Usage != nil {
		chatCompletionResponse.Usage = &smodel.Usage{
			PromptTokens:     responsesRes.Usage.InputTokens,
			CompletionTokens: responsesRes.Usage.OutputTokens,
			TotalTokens:      responsesRes.Usage.TotalTokens,
			PromptTokensDetails: smodel.PromptTokensDetails{
				CachedTokens: responsesRes.Usage.InputTokensDetails.CachedTokens,
				TextTokens:   responsesRes.Usage.InputTokensDetails.TextTokens,
			},
			CompletionTokensDetails: smodel.CompletionTokensDetails{
				ReasoningTokens: responsesRes.Usage.OutputTokensDetails.ReasoningTokens,
			},
		}
	}

	return chatCompletionResponse
}

func ConvResponsesStreamToChatCompletionsResponse(ctx context.Context, res smodel.OpenAIResponsesStreamRes) smodel.ChatCompletionResponse {

	responsesStreamRes := smodel.OpenAIResponsesStreamRes{
		ResponseBytes: res.ResponseBytes,
		ConnTime:      res.ConnTime,
		Duration:      res.Duration,
		TotalTime:     res.TotalTime,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &responsesStreamRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := smodel.ChatCompletionResponse{
		Id:            responsesStreamRes.Response.Id,
		Object:        responsesStreamRes.Response.Object,
		Created:       responsesStreamRes.Response.CreatedAt,
		Model:         responsesStreamRes.Response.Model,
		ServiceTier:   responsesStreamRes.Response.ServiceTier,
		ResponseBytes: responsesStreamRes.ResponseBytes,
		ConnTime:      responsesStreamRes.ConnTime,
		Duration:      responsesStreamRes.Duration,
		TotalTime:     responsesStreamRes.TotalTime,
		Error:         responsesStreamRes.Err,
	}

	if chatCompletionResponse.Id == "" {
		chatCompletionResponse.Id = responsesStreamRes.Item.Id
	}

	if chatCompletionResponse.Id == "" {
		chatCompletionResponse.Id = responsesStreamRes.ItemId
	}

	chatCompletionChoice := smodel.ChatCompletionChoice{
		Delta: &smodel.ChatCompletionStreamChoiceDelta{
			Content: responsesStreamRes.Delta,
		},
	}

	if "response.completed" == responsesStreamRes.Type {
		chatCompletionChoice.FinishReason = "stop"
	}

	chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, chatCompletionChoice)

	if responsesStreamRes.Response.Usage != nil {
		chatCompletionResponse.Usage = &smodel.Usage{
			PromptTokens:     responsesStreamRes.Response.Usage.InputTokens,
			CompletionTokens: responsesStreamRes.Response.Usage.OutputTokens,
			TotalTokens:      responsesStreamRes.Response.Usage.TotalTokens,
			PromptTokensDetails: smodel.PromptTokensDetails{
				CachedTokens: responsesStreamRes.Response.Usage.InputTokensDetails.CachedTokens,
				TextTokens:   responsesStreamRes.Response.Usage.InputTokensDetails.TextTokens,
			},
			CompletionTokensDetails: smodel.CompletionTokensDetails{
				ReasoningTokens: responsesStreamRes.Response.Usage.OutputTokensDetails.ReasoningTokens,
			},
		}
	}

	return chatCompletionResponse
}

func ConvChatCompletionsToResponsesRequest(ctx context.Context, body []byte) smodel.OpenAIResponsesReq {

	chatCompletionRequest := smodel.ChatCompletionRequest{}
	if err := gjson.Unmarshal(body, &chatCompletionRequest); err != nil {
		logger.Error(ctx, err)
		return smodel.OpenAIResponsesReq{}
	}

	responsesReq := smodel.OpenAIResponsesReq{
		Model:             chatCompletionRequest.Model,
		Stream:            chatCompletionRequest.Stream,
		MaxOutputTokens:   chatCompletionRequest.MaxTokens,
		Metadata:          chatCompletionRequest.Metadata,
		ParallelToolCalls: chatCompletionRequest.ParallelToolCalls != nil,
		Store:             chatCompletionRequest.Store,
		Temperature:       chatCompletionRequest.Temperature,
		Tools:             chatCompletionRequest.Tools,
		ToolChoice:        chatCompletionRequest.ToolChoice,
		TopP:              chatCompletionRequest.TopP,
		User:              chatCompletionRequest.User,
		ServiceTier:       chatCompletionRequest.ServiceTier,
	}

	input := make([]smodel.OpenAIResponsesInput, 0)

	for _, message := range chatCompletionRequest.Messages {

		responsesContent := make([]smodel.OpenAIResponsesContent, 0)

		if multiContent, ok := message.Content.([]any); ok {
			for _, value := range multiContent {
				if content, ok := value.(map[string]any); ok {

					if content["type"] == "text" {
						responsesContent = append(responsesContent, smodel.OpenAIResponsesContent{
							Type: "input_text",
							Text: gconv.String(content["text"]),
						})
					} else if content["type"] == "image_url" {

						imageContent := smodel.OpenAIResponsesContent{
							Type: "input_image",
						}

						if imageUrl, ok := content["image_url"].(map[string]any); ok {
							imageContent.ImageUrl = gconv.String(imageUrl["url"])
						}

						responsesContent = append(responsesContent, imageContent)
					}
				}
			}
		} else {
			responsesContent = append(responsesContent, smodel.OpenAIResponsesContent{
				Type: "input_text",
				Text: gconv.String(message.Content),
			})
		}

		input = append(input, smodel.OpenAIResponsesInput{
			Role:    message.Role,
			Content: responsesContent,
		})
	}

	responsesReq.Input = input

	if chatCompletionRequest.ReasoningEffort != "" {
		responsesReq.Reasoning = &smodel.OpenAIResponsesReasoning{
			Effort: chatCompletionRequest.ReasoningEffort,
		}
	}

	return responsesReq
}
