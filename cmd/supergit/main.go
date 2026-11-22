package main

import (
	"fmt"
	"os"

	"supergit/internal/updater"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "--apply-update" {
		if err := updater.RunApplyHelper(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "supergit apply-update: %v\n", err)
			os.Exit(1)
		}
		return
	}
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
	if err := runApp(path); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
