package common

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/util/gconv"
	mcommon "github.com/iimeta/fastapi/v2/internal/model/common"
)

// 从响应 JSON 中提取调用方看到的消息ID.
// 不同接口字段不同: id / request_id / message.id / response.id / task_id.
func ExtractMsgId(data []byte) string {

	if len(data) == 0 {
		return ""
	}

	var payload map[string]any
	if err := gjson.Unmarshal(data, &payload); err != nil {
		return ""
	}

	return ExtractMsgIdFromMap(payload)
}

// 从响应 map 中提取调用方看到的消息ID.
func ExtractMsgIdFromMap(payload map[string]any) string {

	if len(payload) == 0 {
		return ""
	}

	if id := gconv.String(payload["id"]); id != "" {
		return id
	}

	if id := gconv.String(payload["request_id"]); id != "" {
		return id
	}

	if id := gconv.String(payload["task_id"]); id != "" {
		return id
	}

	if message, ok := payload["message"].(map[string]any); ok {
		if id := gconv.String(message["id"]); id != "" {
			return id
		}
	}

	if response, ok := payload["response"].(map[string]any); ok {
		if id := gconv.String(response["id"]); id != "" {
			return id
		}
	}

	return ""
}

// ResolveMsgId 优先使用已解析的响应ID(调用方实际收到的), 否则从原始响应中提取.
func ResolveMsgId(id string, responseBytes []byte) string {

	if id != "" {
		return id
	}

	return ExtractMsgId(responseBytes)
}

func resolveTextMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if id := ExtractMsgId(after.ChatCompletionRes.ResponseBytes); id != "" {
		return id
	}

	if after.ChatCompletionRes.Id != "" {
		return after.ChatCompletionRes.Id
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveImageMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if after.ImageId != "" {
		return after.ImageId
	}

	if id := ExtractMsgId(after.ImageResponse.ResponseBytes); id != "" {
		return id
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveAudioMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveVideoMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if after.VideoId != "" {
		return after.VideoId
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveFileMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if after.FileId != "" {
		return after.FileId
	}

	if after.FileRes.Id != "" {
		return after.FileRes.Id
	}

	if id := ExtractMsgId(after.FileRes.ResponseBytes); id != "" {
		return id
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveBatchMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if after.BatchId != "" {
		return after.BatchId
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}

func resolveGeneralMsgId(after *mcommon.AfterHandler) string {

	if after.MsgId != "" {
		return after.MsgId
	}

	if id := ExtractMsgId(after.ChatCompletionRes.ResponseBytes); id != "" {
		return id
	}

	if after.ChatCompletionRes.Id != "" {
		return after.ChatCompletionRes.Id
	}

	return ExtractMsgIdFromMap(after.ResponseData)
}
