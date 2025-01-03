package main

import (
	_ "github.com/iimeta/fastapi/internal/logic"

	_ "github.com/iimeta/fastapi/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
	"github.com/iimeta/fastapi/internal/cmd"
)

func main() {

	// 设置进程全局时区
	if err := gtime.SetTimeZone("Asia/Shanghai"); err != nil {
		panic(err)
	}

	cmd.Main.Run(gctx.GetInitCtx())
}
