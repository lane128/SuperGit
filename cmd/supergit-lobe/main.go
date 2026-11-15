package main

import (
	"fmt"
	"os"

	"supergit/internal/lobedesk"
)

func main() {
	path := ""
	if len(os.Args) > 2 {
		fmt.Fprintln(os.Stderr, "用法: supergit-lobe [仓库路径]")
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			fmt.Println("SuperGit Lobe UI 预览 — MyGo 原生 Git 客户端\n用法: supergit-lobe [仓库路径]")
			return
		}
		path = os.Args[1]
	}
	if err := lobedesk.Run(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
