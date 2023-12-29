package model

// 使用量接口响应参数
type UsageRes struct {
	UsageCount  int `json:"usage_count"`
	UsedTokens  int `json:"used_tokens"`
	TotalTokens int `json:"total_tokens"`
}
