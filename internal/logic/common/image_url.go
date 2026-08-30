package common

import (
	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/text/gstr"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/config"
)

func isImageUrlReplaceOpen() bool {
	return config.Cfg != nil && config.Cfg.ImageUrl != nil && config.Cfg.ImageUrl.Open && len(config.Cfg.ImageUrl.Urls) > 0
}

// 按配置顺序对图像 URL 做前缀替换, 命中后继续套后续规则
func ReplaceImageUrl(imageUrl string) string {

	if imageUrl == "" || !isImageUrlReplaceOpen() {
		return imageUrl
	}

	for _, item := range config.Cfg.ImageUrl.Urls {
		if item.ReplaceUrl == "" {
			continue
		}
		if gstr.HasPrefix(imageUrl, item.ReplaceUrl) {
			imageUrl = item.TargetUrl + imageUrl[len(item.ReplaceUrl):]
		}
	}

	return imageUrl
}

func ReplaceImageDataUrls(data []smodel.ImageResponseData) {
	for i := range data {
		data[i].Url = ReplaceImageUrl(data[i].Url)
	}
}

func ReplaceImageResponseUrls(response *smodel.ImageResponse) {
	if response == nil {
		return
	}
	ReplaceImageDataUrls(response.Data)
	if len(response.ResponseBytes) > 0 {
		response.ResponseBytes = ReplaceImageUrlsInJSON(response.ResponseBytes)
	}
}

func ReplaceImageUrlsInJSON(data []byte) []byte {

	if len(data) == 0 || !isImageUrlReplaceOpen() {
		return data
	}

	var v any
	if err := gjson.Unmarshal(data, &v); err != nil {
		return data
	}

	if !replaceImageUrlsInValue(v) {
		return data
	}

	return gjson.MustEncode(v)
}

func replaceImageUrlsInValue(v any) bool {

	changed := false

	switch x := v.(type) {
	case map[string]any:
		for k, val := range x {
			if s, ok := val.(string); ok {
				if ns := ReplaceImageUrl(s); ns != s {
					x[k] = ns
					changed = true
				}
			} else if replaceImageUrlsInValue(val) {
				changed = true
			}
		}
	case []any:
		for i, val := range x {
			if s, ok := val.(string); ok {
				if ns := ReplaceImageUrl(s); ns != s {
					x[i] = ns
					changed = true
				}
			} else if replaceImageUrlsInValue(val) {
				changed = true
			}
		}
	}

	return changed
}
