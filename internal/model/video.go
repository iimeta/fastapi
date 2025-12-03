package model

type VideoReq struct {
	Request map[string]any // 请求
}

type VideoRes struct {
	Response     map[string]any // 响应
	Error        error          // 错误信息
	TotalTime    int64          // 总时间
	InternalTime int64          // 内耗时间
	EnterTime    int64          // 进入时间
}
