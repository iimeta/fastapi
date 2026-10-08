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

// 根据是否开启转储决定视频对外地址
// 开启转储: 视频落在本地, 拼接 StorageBaseUrl 前缀
// 未开启转储: 视频由上游托管, video_url 存的即上游完整地址, 原样返回, 再按规则替换
func ResolveVideoUrl(videoUrl string) string {

	if videoUrl == "" {
		return ""
	}

	if config.Cfg.VideoTask != nil && config.Cfg.VideoTask.IsEnableStorage {
		if config.Cfg.VideoTask.StorageBaseUrl != "" {
			if gstr.HasSuffix(config.Cfg.VideoTask.StorageBaseUrl, "/") {
				videoUrl = gstr.TrimLeftStr(videoUrl, "/")
			} else if !gstr.HasPrefix(videoUrl, "/") {
				videoUrl = "/" + videoUrl
			}
			videoUrl = config.Cfg.VideoTask.StorageBaseUrl + videoUrl
		}
	}

	return ReplaceVideoUrl(videoUrl)
}
