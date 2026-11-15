package main

import (
	"fmt"
	"os"
	"supergit/internal/desktop"
)

func main() {
	path := ""
	if len(os.Args) > 2 {
		fmt.Fprintln(os.Stderr, "用法: supergit [仓库路径]")
		os.Exit(2)
	}
	if len(os.Args) == 2 {
		if os.Args[1] == "--help" || os.Args[1] == "-h" {
			fmt.Println("SuperGit — MyGo 原生 Git 客户端\n用法: supergit [仓库路径]")
			return
		}
		path = os.Args[1]
	}
	if err := desktop.Run(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
