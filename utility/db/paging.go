package db

import (
	"math"
)

type Paging struct {
	Page      int64 // 当前页
	PageSize  int64 // 每页条数
	Total     int64 // 总条数
	PageCount int64 // 总页数
	StartNums int64 // 起始条数
	EndNums   int64 // 结束条数
}

// 获取分页信息
func (p *Paging) GetPages() {

	if p.Page < 1 {
		p.Page = 1
	}

	if p.PageSize < 1 {
		p.PageSize = 10
	}

	p.StartNums = p.PageSize * (p.Page - 1)
	if p.StartNums > p.Total {
		p.StartNums = 0
	}

	p.EndNums = p.StartNums + p.PageSize
	if p.EndNums > p.Total {
		p.EndNums = p.Total
	}

	p.PageCount = int64(math.Ceil(float64(p.Total) / float64(p.PageSize)))
}
