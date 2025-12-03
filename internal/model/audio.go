package model

type AudioReq struct {
	Input    string // 输入文本
	FilePath string // 文件路径
}

type AudioRes struct {
	Text         string  // 输出文本
	Characters   int     // 字符数
	Minute       float64 // 分钟数
	Error        error   // 错误信息
	TotalTime    int64   // 总时间
	InternalTime int64   // 内耗时间
	EnterTime    int64   // 进入时间
}
