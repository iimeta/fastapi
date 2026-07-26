package google

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/gogf/gf/v2/encoding/gjson"
	"github.com/gogf/gf/v2/net/gtrace"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/gogf/gf/v2/text/gstr"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

// 转储 Google 原生响应中的 inlineData 图像:
// 开启 ImageStorage 时将 base64 落盘并生成访问 URL;
// IsReturnBase64=false 时把 inlineData.data 替换为 URL, true 时保留 base64 给客户端;
// 返回值中的 imageData 始终带 Url, 供日志按 url/filepath 形式记录.
// imageIndexOffset 用于流式场景下跨 chunk 的文件名序号.
func saveGoogleImageStorage(ctx context.Context, responseBytes []byte, imageIndexOffset int) (newBytes []byte, filePaths []string, expiresAt int64, imageData []smodel.ImageResponseData) {

	if config.Cfg.ImageStorage == nil || !config.Cfg.ImageStorage.Open || len(responseBytes) == 0 {
		return responseBytes, nil, 0, nil
	}

	var root map[string]any
	if err := gjson.Unmarshal(responseBytes, &root); err != nil {
		logger.Error(ctx, err)
		return responseBytes, nil, 0, nil
	}

	candidates, _ := root["candidates"].([]any)
	if len(candidates) == 0 {
		return responseBytes, nil, 0, nil
	}

	storageDir := config.Cfg.ImageStorage.StorageDir
	if storageDir == "" {
		storageDir = "./resource/public/image/"
	} else if !gstr.HasSuffix(storageDir, "/") {
		storageDir = storageDir + "/"
	}

	traceId := gtrace.GetTraceID(ctx)
	idx := imageIndexOffset
	hasStored := false

	for _, candidate := range candidates {

		candMap, ok := candidate.(map[string]any)
		if !ok {
			continue
		}

		content, _ := candMap["content"].(map[string]any)
		if content == nil {
			continue
		}

		parts, _ := content["parts"].([]any)
		for _, part := range parts {

			partMap, ok := part.(map[string]any)
			if !ok {
				continue
			}

			inlineData, _ := partMap["inlineData"].(map[string]any)
			if inlineData == nil {
				continue
			}

			dataStr, _ := inlineData["data"].(string)
			if dataStr == "" {
				continue
			}

			// 已是 URL 的不再处理(避免流式重复转储)
			if gstr.HasPrefix(dataStr, "http://") || gstr.HasPrefix(dataStr, "https://") || gstr.HasPrefix(dataStr, "/") {
				continue
			}

			mimeType, _ := inlineData["mimeType"].(string)
			if mimeType == "" {
				mimeType, _ = inlineData["mime_type"].(string)
			}
			// 仅处理图像类型; mimeType 缺失时按图像处理(生图响应通常如此)
			if mimeType != "" && !gstr.HasPrefix(mimeType, "image/") {
				continue
			}

			imageBytes, err := base64.StdEncoding.DecodeString(dataStr)
			if err != nil {
				logger.Error(ctx, err)
				continue
			}

			ext := mimeToExt(mimeType)
			fileName := fmt.Sprintf("%s%d.%s", traceId, idx, ext)

			if err := gfile.PutBytes(storageDir+fileName, imageBytes); err != nil {
				logger.Error(ctx, err)
				continue
			}

			imageUrl := buildImageStorageUrl(storageDir, fileName)

			filePaths = append(filePaths, storageDir+fileName)
			imageData = append(imageData, smodel.ImageResponseData{Url: imageUrl})

			// 关闭 IsReturnBase64 时用 URL 替换 base64 回传给客户端
			if !config.Cfg.ImageStorage.IsReturnBase64 {
				inlineData["data"] = imageUrl
			}

			hasStored = true
			idx++
		}
	}

	if !hasStored {
		return responseBytes, nil, 0, nil
	}

	if config.Cfg.ImageStorage.StorageExpiresAt > 0 {
		expiresAt = gtime.NewFromTimeStamp(gtime.TimestampMilli() / 1000).Add(config.Cfg.ImageStorage.StorageExpiresAt * time.Minute).Unix()
	}

	return gjson.MustEncode(root), filePaths, expiresAt, imageData
}

// 根据存储目录规则拼接对外可访问的图像 URL, 逻辑与 image.saveImageStorage 保持一致
func buildImageStorageUrl(storageDir, fileName string) string {

	var imageUrl string
	if gstr.HasPrefix(storageDir, "./resource/public/") {
		imageUrl = "/public/" + gstr.TrimLeftStr(storageDir, "./resource/public/") + fileName
	} else if config.Cfg.ImageStorage.StorageBaseUrl == "" {
		imageUrl = "/open/image/" + fileName
	} else {
		imageUrl = fileName
	}

	if config.Cfg.ImageStorage.StorageBaseUrl != "" {
		if gstr.HasSuffix(config.Cfg.ImageStorage.StorageBaseUrl, "/") {
			imageUrl = gstr.TrimLeftStr(imageUrl, "/")
		} else if !gstr.HasPrefix(imageUrl, "/") {
			imageUrl = "/" + imageUrl
		}
		imageUrl = config.Cfg.ImageStorage.StorageBaseUrl + imageUrl
	}

	return imageUrl
}

// mimeType 转文件扩展名
func mimeToExt(mimeType string) string {
	switch gstr.ToLower(mimeType) {
	case "image/jpeg", "image/jpg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/bmp":
		return "bmp"
	case "image/png", "":
		return "png"
	default:
		// image/xxx -> xxx
		if i := gstr.PosR(mimeType, "/"); i != -1 && i+1 < len(mimeType) {
			return mimeType[i+1:]
		}
		return "png"
	}
}
