package common

type UserPrivacy struct {
	LogRequestContent  bool     `bson:"log_request_content"  json:"log_request_content"`  // 记录请求内容
	LogResponseContent bool     `bson:"log_response_content" json:"log_response_content"` // 记录响应内容
	LogResourceUrl     bool     `bson:"log_resource_url"     json:"log_resource_url"`     // 记录资源链接
	LogClientIp        bool     `bson:"log_client_ip"        json:"log_client_ip"`        // 记录客户端IP
	LogRequestFields   []string `bson:"log_request_fields"   json:"log_request_fields"`   // 请求内容字段
	LogResponseFields  []string `bson:"log_response_fields"  json:"log_response_fields"`  // 响应内容字段
	LogResourceFields  []string `bson:"log_resource_fields"  json:"log_resource_fields"`  // 资源链接字段
	LogNetworkFields   []string `bson:"log_network_fields"   json:"log_network_fields"`   // 网络信息字段
}

type PrivacyLogFieldOption struct {
	Key         string   `bson:"key"         json:"key"`                   // 字段标识
	Label       string   `bson:"label"       json:"label"`                 // 展示名称
	Category    string   `bson:"category"    json:"category"`              // 分类[request,response,resource,network]
	Description string   `bson:"description" json:"description,omitempty"` // 描述
	LogTypes    []string `bson:"log_types"   json:"log_types,omitempty"`   // 日志类型
	Enabled     bool     `bson:"enabled"     json:"enabled"`               // 是否启用
	Sort        int      `bson:"sort"        json:"sort,omitempty"`        // 排序
}

func DefaultUserPrivacy() *UserPrivacy {
	return &UserPrivacy{
		LogRequestFields:  []string{},
		LogResponseFields: []string{},
		LogResourceFields: []string{},
		LogNetworkFields:  []string{},
	}
}

func DefaultPrivacyLogFields() []PrivacyLogFieldOption {
	return []PrivacyLogFieldOption{
		{Key: "messages", Label: "完整消息上下文", Category: "request", Description: "保存文本对话的完整 messages", LogTypes: []string{"log_text"}, Enabled: true, Sort: 10},
		{Key: "prompt", Label: "提示词", Category: "request", Description: "保存文本、绘图、Midjourney 等提示词", LogTypes: []string{"log_text", "log_image", "log_midjourney"}, Enabled: true, Sort: 20},
		{Key: "request_data", Label: "请求参数", Category: "request", Description: "保存视频、文件、批处理、通用接口请求参数", LogTypes: []string{"log_video", "log_file", "log_batch", "log_general"}, Enabled: true, Sort: 30},
		{Key: "input", Label: "音频输入文本", Category: "request", Description: "保存音频接口输入文本", LogTypes: []string{"log_audio"}, Enabled: true, Sort: 40},
		{Key: "completion", Label: "模型输出", Category: "response", Description: "保存文本或通用接口模型输出", LogTypes: []string{"log_text", "log_general"}, Enabled: true, Sort: 10},
		{Key: "response_data", Label: "响应数据", Category: "response", Description: "保存视频、文件、批处理、通用接口响应数据", LogTypes: []string{"log_video", "log_file", "log_batch", "log_general"}, Enabled: true, Sort: 20},
		{Key: "text", Label: "音频输出文本", Category: "response", Description: "保存音频接口输出或转写文本", LogTypes: []string{"log_audio"}, Enabled: true, Sort: 30},
		{Key: "revised_prompt", Label: "改写提示词", Category: "response", Description: "保存绘图接口返回的改写提示词", LogTypes: []string{"log_image"}, Enabled: true, Sort: 40},
		{Key: "upstream_response", Label: "上游完整响应", Category: "response", Description: "保存 Midjourney 等上游完整响应", LogTypes: []string{"log_midjourney"}, Enabled: true, Sort: 50},
		{Key: "image_url", Label: "图片地址", Category: "resource", Description: "保存生成图片或图片资源地址", Enabled: true, Sort: 10},
		{Key: "file_url", Label: "文件地址", Category: "resource", Description: "保存文件资源地址", Enabled: true, Sort: 20},
		{Key: "video_url", Label: "视频地址", Category: "resource", Description: "保存视频资源地址", Enabled: true, Sort: 30},
		{Key: "download_url", Label: "下载地址", Category: "resource", Description: "保存下载地址", Enabled: true, Sort: 40},
		{Key: "content_url", Label: "内容地址", Category: "resource", Description: "保存内容访问地址", Enabled: true, Sort: 50},
		{Key: "b64_json", Label: "Base64 图片数据", Category: "resource", Description: "保存 Base64 图片数据", Enabled: true, Sort: 60},
		{Key: "data", Label: "内联资源数据", Category: "resource", Description: "保存内联资源数据", Enabled: true, Sort: 70},
		{Key: "client_ip", Label: "客户端 IP", Category: "network", Description: "保存发起请求的客户端 IP", Enabled: true, Sort: 10},
	}
}

func EnabledPrivacyLogFields(fields []PrivacyLogFieldOption) []PrivacyLogFieldOption {
	if len(fields) == 0 {
		fields = DefaultPrivacyLogFields()
	}
	items := make([]PrivacyLogFieldOption, 0)
	for _, field := range fields {
		if field.Enabled && field.Key != "" && field.Category != "" {
			items = append(items, field)
		}
	}
	return items
}

func NormalizeUserPrivacy(privacy *UserPrivacy, fields []PrivacyLogFieldOption) *UserPrivacy {
	if privacy == nil {
		return DefaultUserPrivacy()
	}
	return &UserPrivacy{
		LogRequestContent:  privacy.LogRequestContent,
		LogResponseContent: privacy.LogResponseContent,
		LogResourceUrl:     privacy.LogResourceUrl,
		LogClientIp:        privacy.LogClientIp,
		LogRequestFields:   normalizePrivacyFields(privacy.LogRequestFields, fields, "request", privacy.LogRequestContent),
		LogResponseFields:  normalizePrivacyFields(privacy.LogResponseFields, fields, "response", privacy.LogResponseContent),
		LogResourceFields:  normalizePrivacyFields(privacy.LogResourceFields, fields, "resource", privacy.LogResourceUrl),
		LogNetworkFields:   normalizePrivacyFields(privacy.LogNetworkFields, fields, "network", privacy.LogClientIp),
	}
}

func normalizePrivacyFields(values []string, fields []PrivacyLogFieldOption, category string, enabled bool) []string {
	if !enabled {
		return []string{}
	}
	allowed := map[string]bool{}
	all := make([]string, 0)
	for _, field := range EnabledPrivacyLogFields(fields) {
		if field.Category == category {
			allowed[field.Key] = true
			all = append(all, field.Key)
		}
	}
	if len(values) == 0 {
		return all
	}
	result := make([]string, 0)
	seen := map[string]bool{}
	for _, value := range values {
		if allowed[value] && !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	return result
}
