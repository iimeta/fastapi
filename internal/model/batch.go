package model

type BatchReq struct {
	Action      string         // 接口
	RequestData map[string]any // 请求数据
}

type BatchRes struct {
	BatchId      string            // 批处理ID
	MsgId        string            // 消息ID
	ResponseData map[string]any    // 响应数据
	Error        error             // 错误信息
	TotalTime    int64             // 总时间
	InternalTime int64             // 内耗时间
	EnterTime    int64             // 进入时间
	RequestIds   map[string]string // 上游请求ID
}
