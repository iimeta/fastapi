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
	"github.com/iimeta/fastapi/v2/internal/logic/common"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

func saveGoogleInteractionImageStorage(ctx context.Context, responseBytes []byte, imageIndexOffset int) (newBytes []byte, filePaths []string, expiresAt int64, imageData []smodel.ImageResponseData) {

	if config.Cfg.ImageStorage == nil || !config.Cfg.ImageStorage.Open || len(responseBytes) == 0 {
		return common.ReplaceImageUrlsInJSON(responseBytes), nil, 0, nil
	}

	var root map[string]any
	if err := gjson.Unmarshal(responseBytes, &root); err != nil {
		logger.Error(ctx, err)
		return common.ReplaceImageUrlsInJSON(responseBytes), nil, 0, nil
	}

	steps, _ := root["steps"].([]any)
	if len(steps) == 0 {
		return common.ReplaceImageUrlsInJSON(responseBytes), nil, 0, nil
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
	isRaw := isImageStorageRaw(ctx)

	for _, step := range steps {

		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}

		contents, _ := stepMap["content"].([]any)
		for _, item := range contents {

			contentMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			contentType, _ := contentMap["type"].(string)
			mimeType, _ := contentMap["mime_type"].(string)
			if mimeType == "" {
				mimeType, _ = contentMap["mimeType"].(string)
			}

			if contentType != "image" && !gstr.HasPrefix(mimeType, "image/") {
				continue
			}

			dataStr, _ := contentMap["data"].(string)
			if dataStr == "" || isImageUrl(dataStr) {
				continue
			}

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

			if !config.Cfg.ImageStorage.IsReturnBase64 && !isRaw {
				contentMap["data"] = imageUrl
			}

			hasStored = true
			idx++
		}
	}

	if !hasStored {
		return common.ReplaceImageUrlsInJSON(responseBytes), nil, 0, nil
	}

	if config.Cfg.ImageStorage.StorageExpiresAt > 0 {
		expiresAt = gtime.NewFromTimeStamp(gtime.TimestampMilli() / 1000).Add(config.Cfg.ImageStorage.StorageExpiresAt * time.Minute).Unix()
	}

	common.ReplaceImageDataUrls(imageData)

	return common.ReplaceImageUrlsInJSON(gjson.MustEncode(root)), filePaths, expiresAt, imageData
}

func extractGoogleInteractionImageDataForLog(responseBytes []byte) []smodel.ImageResponseData {

	if len(responseBytes) == 0 {
		return nil
	}

	var root map[string]any
	if err := gjson.Unmarshal(responseBytes, &root); err != nil {
		return nil
	}

	steps, _ := root["steps"].([]any)
	if len(steps) == 0 {
		return nil
	}

	var result []smodel.ImageResponseData

	for _, step := range steps {

		stepMap, ok := step.(map[string]any)
		if !ok {
			continue
		}

		contents, _ := stepMap["content"].([]any)
		for _, item := range contents {

			contentMap, ok := item.(map[string]any)
			if !ok {
				continue
			}

			contentType, _ := contentMap["type"].(string)
			mimeType, _ := contentMap["mime_type"].(string)
			if mimeType == "" {
				mimeType, _ = contentMap["mimeType"].(string)
			}

			if contentType != "image" && !gstr.HasPrefix(mimeType, "image/") {
				continue
			}

			dataStr, _ := contentMap["data"].(string)
			uriStr, _ := contentMap["uri"].(string)

			itemData := smodel.ImageResponseData{}
			if isImageUrl(uriStr) {
				itemData.Url = uriStr
			}
			if dataStr != "" {
				if isImageUrl(dataStr) {
					itemData.Url = dataStr
				} else {
					itemData.B64Json = dataStr
				}
			}

			if itemData.Url != "" || itemData.B64Json != "" {
				result = append(result, itemData)
			}
		}
	}

	return result
}
