package log

import (
	"context"

	"github.com/iimeta/fastapi/v2/internal/model/common"
	"github.com/iimeta/fastapi/v2/internal/model/do"
	"github.com/iimeta/fastapi/v2/internal/service"
)

func privacy(ctx context.Context) *common.UserPrivacy {
	return service.User().GetPrivacy(ctx, service.Session().GetUserId(ctx))
}

func hasField(fields []string, field string) bool {

	for _, value := range fields {
		if value == field {
			return true
		}
	}

	return false
}

func allowRequestField(privacy *common.UserPrivacy, field string) bool {
	return privacy != nil && privacy.LogRequestContent && hasField(privacy.LogRequestFields, field)
}

func allowResponseField(privacy *common.UserPrivacy, field string) bool {
	return privacy != nil && privacy.LogResponseContent && hasField(privacy.LogResponseFields, field)
}

func allowResourceField(privacy *common.UserPrivacy, field string) bool {
	return privacy != nil && privacy.LogResourceUrl && hasField(privacy.LogResourceFields, field)
}

func allowNetworkField(privacy *common.UserPrivacy, field string) bool {
	return privacy != nil && privacy.LogClientIp && hasField(privacy.LogNetworkFields, field)
}

func applyTextPrivacy(text *do.LogText, privacy *common.UserPrivacy) {

	text.Privacy = privacy

	if !allowRequestField(privacy, "messages") {
		text.Messages = nil
	}

	if !allowRequestField(privacy, "prompt") {
		text.Prompt = ""
	}

	if !allowResponseField(privacy, "completion") {
		text.Completion = ""
	}

	if !allowNetworkField(privacy, "client_ip") {
		text.ClientIp = ""
	}
}

func applyImagePrivacy(image *do.LogImage, privacy *common.UserPrivacy) {

	image.Privacy = privacy

	if !allowRequestField(privacy, "prompt") {
		image.Prompt = ""
	}

	for i := range image.ImageData {
		if !allowResponseField(privacy, "revised_prompt") {
			image.ImageData[i].RevisedPrompt = ""
		}
		if !allowResourceField(privacy, "image_url") {
			image.ImageData[i].Url = ""
		}
	}

	if !allowNetworkField(privacy, "client_ip") {
		image.ClientIp = ""
	}
}

func applyAudioPrivacy(audio *do.LogAudio, privacy *common.UserPrivacy) {

	audio.Privacy = privacy

	if !allowRequestField(privacy, "input") {
		audio.Input = ""
	}

	if !allowResponseField(privacy, "text") {
		audio.Text = ""
	}

	if !allowNetworkField(privacy, "client_ip") {
		audio.ClientIp = ""
	}
}

func applyVideoPrivacy(video *do.LogVideo, privacy *common.UserPrivacy) {

	video.Privacy = privacy

	if !allowRequestField(privacy, "request_data") {
		video.RequestData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(video.RequestData, privacy)
	}

	if !allowResponseField(privacy, "response_data") {
		video.ResponseData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(video.ResponseData, privacy)
	}

	if !allowNetworkField(privacy, "client_ip") {
		video.ClientIp = ""
	}
}

func applyFilePrivacy(file *do.LogFile, privacy *common.UserPrivacy) {

	file.Privacy = privacy

	if !allowRequestField(privacy, "request_data") {
		file.RequestData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(file.RequestData, privacy)
	}

	if !allowResponseField(privacy, "response_data") {
		file.ResponseData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(file.ResponseData, privacy)
	}

	if !allowNetworkField(privacy, "client_ip") {
		file.ClientIp = ""
	}
}

func applyBatchPrivacy(batch *do.LogBatch, privacy *common.UserPrivacy) {

	batch.Privacy = privacy

	if !allowRequestField(privacy, "request_data") {
		batch.RequestData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(batch.RequestData, privacy)
	}

	if !allowResponseField(privacy, "response_data") {
		batch.ResponseData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(batch.ResponseData, privacy)
	}

	if !allowNetworkField(privacy, "client_ip") {
		batch.ClientIp = ""
	}
}

func applyGeneralPrivacy(general *do.LogGeneral, privacy *common.UserPrivacy) {

	general.Privacy = privacy

	if !allowRequestField(privacy, "request_data") {
		general.RequestData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(general.RequestData, privacy)
	}

	if !allowResponseField(privacy, "response_data") {
		general.ResponseData = nil
	} else if !privacy.LogResourceUrl {
		cleanResourceData(general.ResponseData, privacy)
	}

	if !allowResponseField(privacy, "completion") {
		general.Completion = ""
	}

	if !allowNetworkField(privacy, "client_ip") {
		general.ClientIp = ""
	}
}

func cleanResourceData(value any, privacy *common.UserPrivacy) {
	switch data := value.(type) {
	case map[string]any:
		for key, item := range data {
			if isResourceKey(key) && !allowResourceKey(privacy, key) {
				delete(data, key)
				continue
			}
			cleanResourceData(item, privacy)
		}
	case []any:
		for _, item := range data {
			cleanResourceData(item, privacy)
		}
	}
}

func isResourceKey(key string) bool {

	switch key {
	case "url", "image_url", "file_url", "video_url", "download_url", "content_url", "b64_json", "data":
		return true
	}

	return false
}

func allowResourceKey(privacy *common.UserPrivacy, key string) bool {

	if key == "url" {
		return allowResourceField(privacy, "image_url") ||
			allowResourceField(privacy, "file_url") ||
			allowResourceField(privacy, "video_url") ||
			allowResourceField(privacy, "download_url") ||
			allowResourceField(privacy, "content_url")
	}

	return allowResourceField(privacy, key)
}
