package common

import (
	"bytes"
	"context"
	"encoding/base64"
	stdimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/text/gstr"
	smodel "github.com/iimeta/fastapi-sdk/v2/model"
	sutil "github.com/iimeta/fastapi-sdk/v2/util"
	"github.com/iimeta/fastapi/v2/internal/config"
	"github.com/iimeta/fastapi/v2/internal/consts"
	"github.com/iimeta/fastapi/v2/internal/errors"
	"github.com/iimeta/fastapi/v2/internal/service"
	"github.com/iimeta/fastapi/v2/utility/logger"
)

const imageSizeRetryKey = "image_size_retry"

// 分组开启图像尺寸强校验时才校验; 无分组则不校验.
func ShouldCheckImageSize(mak *MAK) bool {
	return mak != nil && mak.Group != nil && mak.Group.IsEnableImageSizeCheck
}

// 从用户传入的 size(如 1024x1024) 解析期望宽高; 无法解析时 ok=false, 调用方应跳过校验.
func ExpectedImageSize(size string) (width, height int, ok bool) {
	width, height = parseSizeWH(size)
	if width > 0 && height > 0 {
		return width, height, true
	}
	return 0, 0, false
}

// 从 Google imageSize(1K/2K/4K) + aspectRatio 解析期望宽高, 映射与 billing.imageGeneration 一致.
// 分辨率和比例都未传时 ok=false, 调用方应跳过校验; 只传其一则补默认值(1K / 1:1).
func ExpectedGoogleImageSize(imageSize, aspectRatio string) (width, height int, ok bool) {

	imageSize = gstr.Trim(imageSize)
	aspectRatio = gstr.Trim(aspectRatio)

	if imageSize == "" && aspectRatio == "" {
		return 0, 0, false
	}

	if aspectRatio == "" {
		aspectRatio = "1:1"
	}

	if imageSize == "" {
		imageSize = "1K"
	}

	if !gstr.HasSuffix(imageSize, "K") {
		return 0, 0, false
	}

	size := consts.RESOLUTION_ASPECT_RATIO[imageSize+aspectRatio]
	width, height = parseSizeWH(size)
	if width > 0 && height > 0 {
		return width, height, true
	}

	return 0, 0, false
}

// 校验上游返回的图片宽高是否与期望一致(允许宽高对调).
// data 中每张图的 Url(http/https 或 data URI) 与 B64Json(原始 base64 或 data URI) 均支持.
func CheckGeneratedImageSize(ctx context.Context, expectedW, expectedH int, data []smodel.ImageResponseData) error {

	if expectedW <= 0 || expectedH <= 0 {
		return nil
	}

	if len(data) == 0 {
		return errors.NewError(500, "image_size_mismatch", "Generated image size mismatch: no image in response.", "fastapi_error", nil)
	}

	for i, item := range data {

		imageBytes, err := imageBytesFromData(ctx, item)
		if err != nil {
			logger.Errorf(ctx, "CheckGeneratedImageSize image[%d] read error: %v", i, err)
			return errors.NewErrorf(500, "image_size_mismatch", "Generated image size mismatch: failed to read image %d: %s.", "fastapi_error", nil, i, err.Error())
		}

		actualW, actualH, err := decodeImageDimension(imageBytes)
		if err != nil {
			logger.Errorf(ctx, "CheckGeneratedImageSize image[%d] decode error: %v", i, err)
			return errors.NewErrorf(500, "image_size_mismatch", "Generated image size mismatch: failed to decode image %d: %s.", "fastapi_error", nil, i, err.Error())
		}

		if imageSizeMatches(expectedW, expectedH, actualW, actualH) {
			continue
		}

		logger.Infof(ctx, "CheckGeneratedImageSize image[%d] mismatch: expected %dx%d, got %dx%d", i, expectedW, expectedH, actualW, actualH)
		return errors.NewErrorf(500, "image_size_mismatch", "Generated image size mismatch: expected %dx%d, got %dx%d.", "fastapi_error", nil, expectedW, expectedH, actualW, actualH)
	}

	return nil
}

// 将当前模型代理加入本次请求排除列表, 并判断尺寸不符是否还能再重试.
// 独立计数, 不跟普通错误重试混用, 不走 fallback, 不记 key 错误, 不触发自动禁用.
// 分组 ImageSizeCheckRetry>0 用该次数; 为 0 时完整跟随系统 Base.ErrRetry 语义.
// retryCount 为已经发生过的尺寸不符重试次数(首次不符为 0).
func ExcludeAgentForImageSizeRetry(ctx context.Context, mak *MAK) (shouldRetry bool, retryCount int) {

	retryCount = imageSizeRetryCount(ctx)

	if mak != nil && mak.ModelAgent != nil {
		service.Session().RecordErrorModelAgent(ctx, mak.ModelAgent.Id)
	}

	if retryCount >= imageSizeCheckMaxRetry(mak) {
		return false, retryCount
	}

	setImageSizeRetryCount(ctx, retryCount+1)

	return true, retryCount
}

func imageSizeCheckMaxRetry(mak *MAK) int {

	if mak != nil && mak.Group != nil && mak.Group.ImageSizeCheckRetry > 0 {
		return mak.Group.ImageSizeCheckRetry
	}

	errRetry := 0
	if config.Cfg != nil && config.Cfg.Base != nil {
		errRetry = config.Cfg.Base.ErrRetry
	}

	if errRetry > 0 {
		return errRetry
	}
	if errRetry < 0 && mak != nil {
		return mak.AgentTotal
	}

	return 0
}

func imageSizeMatches(expectedW, expectedH, actualW, actualH int) bool {
	return (actualW == expectedW && actualH == expectedH) || (actualW == expectedH && actualH == expectedW)
}

func imageSizeRetryCount(ctx context.Context) int {
	if r := g.RequestFromCtx(ctx); r != nil {
		return r.GetCtxVar(imageSizeRetryKey, 0).Int()
	}
	return 0
}

func setImageSizeRetryCount(ctx context.Context, n int) {
	if r := g.RequestFromCtx(ctx); r != nil {
		r.SetCtxVar(imageSizeRetryKey, n)
	}
}

func imageBytesFromData(ctx context.Context, item smodel.ImageResponseData) ([]byte, error) {

	if payload := gstr.Trim(item.B64Json); payload != "" {
		if isHTTPURL(payload) {
			return downloadImageBytes(ctx, payload)
		}
		return decodeImagePayload(payload)
	}

	if payload := gstr.Trim(item.Url); payload != "" {
		if gstr.HasPrefix(payload, "data:") {
			return decodeImagePayload(payload)
		}
		if isHTTPURL(payload) {
			return downloadImageBytes(ctx, payload)
		}
		return decodeImagePayload(payload)
	}

	return nil, errors.New("image url and b64_json are empty")
}

func decodeImagePayload(payload string) ([]byte, error) {

	if gstr.HasPrefix(payload, "data:") {
		return decodeDataURIPayload(payload)
	}

	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err == nil {
		return decoded, nil
	}

	compact := gstr.ReplaceByMap(payload, map[string]string{
		"\n": "",
		"\r": "",
		" ":  "",
	})
	return base64.StdEncoding.DecodeString(compact)
}

func decodeDataURIPayload(dataURI string) ([]byte, error) {

	if !gstr.HasPrefix(dataURI, "data:") {
		return nil, errors.New("invalid data uri: missing data: prefix")
	}

	idx := gstr.Pos(dataURI, ",")
	if idx < 0 {
		return nil, errors.New("invalid data uri: missing comma separator")
	}

	meta := dataURI[len("data:"):idx]
	payload := dataURI[idx+1:]

	if gstr.HasSuffix(gstr.ToLower(meta), "base64") {
		return base64.StdEncoding.DecodeString(payload)
	}

	decoded, err := url.QueryUnescape(payload)
	if err != nil {
		return nil, err
	}

	return []byte(decoded), nil
}

func isHTTPURL(s string) bool {
	return gstr.HasPrefix(s, "http://") || gstr.HasPrefix(s, "https://")
}

func downloadImageBytes(ctx context.Context, imageUrl string) ([]byte, error) {

	timeout := 60 * time.Second
	if config.Cfg != nil && config.Cfg.ImageStorage != nil && config.Cfg.ImageStorage.DownloadTimeout > 0 {
		timeout = config.Cfg.ImageStorage.DownloadTimeout * time.Second
	} else if config.Cfg != nil && config.Cfg.Base != nil && config.Cfg.Base.ShortTimeout > 0 {
		timeout = config.Cfg.Base.ShortTimeout * time.Second
	}

	var proxyURL string
	if config.Cfg != nil && config.Cfg.Http != nil {
		proxyURL = config.Cfg.Http.ProxyUrl
	}

	body, _, err := sutil.HttpGet(ctx, imageUrl, nil, nil, nil, timeout, proxyURL, nil)
	if err != nil {
		return nil, err
	}
	if len(body) == 0 {
		return nil, errors.New("downloaded image content is empty")
	}

	return body, nil
}

func decodeImageDimension(imageBytes []byte) (width, height int, err error) {

	cfg, _, err := stdimage.DecodeConfig(bytes.NewReader(imageBytes))
	if err == nil {
		return cfg.Width, cfg.Height, nil
	}

	if w, h, ok := decodeWebPConfig(imageBytes); ok {
		return w, h, nil
	}

	return 0, 0, err
}

func decodeWebPConfig(b []byte) (width, height int, ok bool) {

	if len(b) < 30 {
		return 0, 0, false
	}

	if string(b[0:4]) != "RIFF" || string(b[8:12]) != "WEBP" {
		return 0, 0, false
	}

	switch string(b[12:16]) {
	case "VP8X":
		width = 1 + int(b[24]) + int(b[25])<<8 + int(b[26])<<16
		height = 1 + int(b[27]) + int(b[28])<<8 + int(b[29])<<16
		return width, height, width > 0 && height > 0
	case "VP8 ":
		width = int(b[26]) | int(b[27]&0x3f)<<8
		height = int(b[28]) | int(b[29]&0x3f)<<8
		return width, height, width > 0 && height > 0
	case "VP8L":
		if b[20] != 0x2f {
			return 0, 0, false
		}
		bits := uint32(b[21]) | uint32(b[22])<<8 | uint32(b[23])<<16 | uint32(b[24])<<24
		width = int(bits&0x3fff) + 1
		height = int((bits>>14)&0x3fff) + 1
		return width, height, width > 0 && height > 0
	default:
		return 0, 0, false
	}
}
