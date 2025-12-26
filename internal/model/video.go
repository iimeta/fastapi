package model

type VideoReq struct {
	Action      string         // 接口
	RequestData map[string]any // 请求数据
}

type VideoRes struct {
	VideoId      string         // 视频ID
	ResponseData map[string]any // 响应数据
	Error        error          // 错误信息
	TotalTime    int64          // 总时间
	InternalTime int64          // 内耗时间
	EnterTime    int64          // 进入时间
}
