package common

import (
	"github.com/gogf/gf/v2/text/gstr"
	"github.com/iimeta/fastapi/v2/internal/config"
)

func isVideoUrlReplaceOpen() bool {
	return config.Cfg != nil && config.Cfg.VideoUrl != nil && config.Cfg.VideoUrl.Open && len(config.Cfg.VideoUrl.Urls) > 0
}

// 按配置顺序对视频 URL 做前缀替换, 命中后继续套后续规则
func ReplaceVideoUrl(videoUrl string) string {

	if videoUrl == "" || !isVideoUrlReplaceOpen() {
		return videoUrl
	}

	for _, item := range config.Cfg.VideoUrl.Urls {
		if item.ReplaceUrl == "" {
			continue
		}
		if gstr.HasPrefix(videoUrl, item.ReplaceUrl) {
			videoUrl = item.TargetUrl + videoUrl[len(item.ReplaceUrl):]
		}
	}

	return videoUrl
}
