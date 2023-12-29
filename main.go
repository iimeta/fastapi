package main

import (
	_ "github.com/iimeta/fastapi/internal/logic"

	"github.com/gogf/gf/v2/os/gtime"
	_ "github.com/iimeta/fastapi/internal/core"

	_ "github.com/iimeta/fastapi/internal/packed"

	"github.com/gogf/gf/v2/os/gctx"
	"github.com/iimeta/fastapi/internal/cmd"
)

func main() {

	// 设置进程全局时区
	err := gtime.SetTimeZone("Asia/Shanghai")
	if err != nil {
		panic(err)
	}

	cmd.Main.Run(gctx.GetInitCtx())
}
