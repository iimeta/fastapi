package common

import (
	"context"
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/ghttp"
	sdkm "github.com/iimeta/fastapi-sdk/model"
	"github.com/iimeta/fastapi/utility/logger"
)

func ConvToChatCompletionRequest(request *ghttp.Request) sdkm.ChatCompletionRequest {

	openaiResponsesReq := sdkm.OpenAIResponsesReq{}
	if err := gjson.Unmarshal(request.GetBody(), &openaiResponsesReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return sdkm.ChatCompletionRequest{}
	}

	chatCompletionRequest := sdkm.ChatCompletionRequest{
		Model:               openaiResponsesReq.Model,
		MaxCompletionTokens: openaiResponsesReq.MaxOutputTokens,
		Temperature:         openaiResponsesReq.Temperature,
		TopP:                openaiResponsesReq.TopP,
		Stream:              openaiResponsesReq.Stream,
		User:                openaiResponsesReq.User,
		Tools:               openaiResponsesReq.Tools,
		ToolChoice:          openaiResponsesReq.ToolChoice,
		ParallelToolCalls:   openaiResponsesReq.ParallelToolCalls,
		Store:               openaiResponsesReq.Store,
		Metadata:            openaiResponsesReq.Metadata,
	}

	if openaiResponsesReq.Input != nil {
		if value, ok := openaiResponsesReq.Input.([]interface{}); ok {

			inputs := make([]sdkm.OpenAIResponsesInput, 0)
			if err := gjson.Unmarshal(gjson.MustEncode(value), &inputs); err != nil {
				logger.Error(request.GetCtx(), err)
				return chatCompletionRequest
			}

			for _, input := range inputs {
				chatCompletionRequest.Messages = append(chatCompletionRequest.Messages, sdkm.ChatCompletionMessage{
					Role:    input.Role,
					Content: input.Content,
				})
			}

		} else {
			chatCompletionRequest.Messages = []sdkm.ChatCompletionMessage{{
				Role:    "user",
				Content: openaiResponsesReq.Input,
			}}
		}
	}

	if openaiResponsesReq.Reasoning != nil {
		chatCompletionRequest.ReasoningEffort = openaiResponsesReq.Reasoning.Effort
	}

	return chatCompletionRequest
}

func ConvToChatCompletionResponse(ctx context.Context, res sdkm.OpenAIResponsesRes, stream bool) sdkm.ChatCompletionResponse {

	openaiResponsesRes := sdkm.OpenAIResponsesRes{
		Model:         res.Model,
		Usage:         res.Usage,
		ResponseBytes: res.ResponseBytes,
		ConnTime:      res.ConnTime,
		Duration:      res.Duration,
		TotalTime:     res.TotalTime,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &openaiResponsesRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	chatCompletionResponse := sdkm.ChatCompletionResponse{
		ID:            openaiResponsesRes.Id,
		Object:        openaiResponsesRes.Object,
		Created:       openaiResponsesRes.CreatedAt,
		Model:         openaiResponsesRes.Model,
		ResponseBytes: openaiResponsesRes.ResponseBytes,
		ConnTime:      openaiResponsesRes.ConnTime,
		Duration:      openaiResponsesRes.Duration,
		TotalTime:     openaiResponsesRes.TotalTime,
		Error:         openaiResponsesRes.Err,
	}

	if stream {

		chatCompletionChoice := sdkm.ChatCompletionChoice{
			Delta: &sdkm.ChatCompletionStreamChoiceDelta{
				Content: openaiResponsesRes.Delta,
			},
		}

		if "response.completed" == openaiResponsesRes.Type {
			chatCompletionChoice.FinishReason = "stop"
		}

		chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, chatCompletionChoice)

		if openaiResponsesRes.Response.Usage != nil {
			chatCompletionResponse.Usage = &sdkm.Usage{
				PromptTokens:     openaiResponsesRes.Response.Usage.InputTokens,
				CompletionTokens: openaiResponsesRes.Response.Usage.OutputTokens,
				TotalTokens:      openaiResponsesRes.Response.Usage.TotalTokens,
			}
		}

	} else {

		for _, output := range openaiResponsesRes.Output {
			chatCompletionResponse.Choices = append(chatCompletionResponse.Choices, sdkm.ChatCompletionChoice{
				Message: &sdkm.ChatCompletionMessage{
					Role:    output.Role,
					Content: output.Content[0].Text,
				},
				FinishReason: "stop",
			})
		}

		if openaiResponsesRes.Usage != nil {
			chatCompletionResponse.Usage = &sdkm.Usage{
				PromptTokens:     openaiResponsesRes.Usage.InputTokens,
				CompletionTokens: openaiResponsesRes.Usage.OutputTokens,
				TotalTokens:      openaiResponsesRes.Usage.TotalTokens,
			}
		}
	}

	return chatCompletionResponse
}

func ConvToResponsesRequest(request *ghttp.Request) sdkm.OpenAIResponsesReq {

	openaiResponsesReq := sdkm.OpenAIResponsesReq{}
	if err := gjson.Unmarshal(request.GetBody(), &openaiResponsesReq); err != nil {
		logger.Error(request.GetCtx(), err)
		return sdkm.OpenAIResponsesReq{}
	}

	return openaiResponsesReq
}

func ConvToResponsesResponse(ctx context.Context, res sdkm.OpenAIResponsesRes, stream bool) sdkm.OpenAIResponsesRes {

	openaiResponsesRes := sdkm.OpenAIResponsesRes{
		Model:         res.Model,
		Usage:         res.Usage,
		ResponseBytes: res.ResponseBytes,
		Err:           res.Err,
	}

	if res.ResponseBytes != nil {
		if err := gjson.Unmarshal(res.ResponseBytes, &openaiResponsesRes); err != nil {
			logger.Error(ctx, err)
		}
	}

	return openaiResponsesRes
}
