package model

type AudioReq struct {
	Input    string `json:"input,omitempty"`     // 输入文本
	FilePath string `json:"file_path,omitempty"` // 文件路径
}

type AudioRes struct {
	Text         string  `json:"text"`         // 输出文本
	Characters   int     `json:"characters"`   // 字符数
	Minute       float64 `json:"minute"`       // 分钟数
	TotalTokens  int     `json:"total_tokens"` // 总令牌数
	Error        error   `json:"err"`
	ConnTime     int64   `json:"-"`
	Duration     int64   `json:"-"`
	TotalTime    int64   `json:"-"`
	InternalTime int64   `json:"-"`
	EnterTime    int64   `json:"-"`
}
