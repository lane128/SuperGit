package lobedesk

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"supergit/internal/gitrepo"
)

var flowFamilies = []struct{ id, en, zh string }{
	{"feature", "Feature", "功能"}, {"bugfix", "Bugfix", "缺陷修复"}, {"release", "Release", "发布"}, {"hotfix", "Hotfix", "热修复"}, {"support", "Support", "支持"},
}

func flowPaletteActions() []paletteAction {
	makeAction := func(id, en, zh, action, kind string) paletteAction {
		return paletteAction{ID: id, English: en, Chinese: zh, Unavailable: func(a *app) string {
			if !gitrepo.HasGitFlow() {
				return a.paletteText("Install Git Flow (AVH) and reopen SuperGit", "请安装 Git Flow（AVH）后重新打开 SuperGit")
			}
			if reason := extensionIdle(a); reason != "" {
				return reason
			}
			if action != "configure" && a.operationConflicts() {
				return a.t("quickCheckoutConflict")
			}
			if action != "init" && action != "configure" && a.status.Head == "" {
				return a.t("commandPaletteNeedHead")
			}
			if action == "publish" && len(a.remotes) == 0 {
				return a.t("noRemote")
			}
			return ""
		}, Run: func(a *app) { a.openPaletteFlow(a.paletteText(en, zh), action, kind) }}
	}
	actions := []paletteAction{
		makeAction("init_gitflow", "Git Flow: Initialize", "Git Flow：初始化", "init", ""),
		makeAction("edit_gitflow_config", "Git Flow: Configure", "Git Flow：配置", "configure", ""),
	}
	for _, verb := range []struct{ id, en, zh string }{{"start", "Start", "开始"}, {"finish", "Finish", "完成"}, {"publish", "Publish", "推送"}, {"rebase", "Rebase", "变基"}} {
		for _, kind := range flowFamilies {
			if kind.id == "support" && (verb.id == "finish" || verb.id == "publish") {
				continue
			}
			actions = append(actions, makeAction("gitflow_"+verb.id+"_"+kind.id, "Git Flow: "+verb.en+" "+kind.en+" Branch", "Git Flow："+verb.zh+kind.zh+"分支", verb.id, kind.id))
		}
	}
	return actions
}

func (a *app) openPaletteFlow(title, action, kind string) {
	readExtension(a, title, func(ctx context.Context, r *gitrepo.Repository) (gitrepo.FlowConfig, error) { return r.FlowConfig(ctx) }, func(config gitrepo.FlowConfig) {
		q := gitrepo.FlowCommand{Action: action, Kind: kind, Config: config, Expected: a.status, Branches: slices.Clone(a.branches)}
		if action == "init" {
			a.choosePaletteFlowInit(title, q)
			return
		}
		if config["gitflow.branch.master"] == "" || config["gitflow.branch.develop"] == "" {
			a.queue("info", title, a.paletteText("Run Git Flow: Initialize first.", "请先运行 Git Flow：初始化。"))
			return
		}
		if action == "configure" {
			a.editPaletteFlowConfig(title, config)
			return
		}
		prefix, ok := config["gitflow.prefix."+kind]
		if !ok {
			a.queue("warning", title, a.paletteText("Configure the branch prefix first.", "请先配置分支前缀。"))
			return
		}
		if action == "start" {
			a.startPaletteFlow(title, prefix, q)
			return
		}
		a.choosePaletteFlowBranch(title, prefix, q)
	})
}

func (a *app) startPaletteFlow(title, prefix string, q gitrepo.FlowCommand) {
	a.inputPalette(title, a.paletteText("Branch name after ", "分支名，前缀为 ")+prefix, "", func(name string) {
		q.Name = name
		rows := []paletteChoice{}
		for _, b := range q.Branches {
			row := paletteChoice{id: b.Name, label: b.Name, sha: b.SHA, detail: shortSHA(b.SHA)}
			if b.Name == q.Config.DefaultBase(q.Kind) {
				rows = append([]paletteChoice{row}, rows...)
			} else {
				rows = append(rows, row)
			}
		}
		if q.Kind == "support" {
			for _, t := range a.tags {
				if t.TargetType == "commit" {
					rows = append(rows, paletteChoice{id: "refs/tags/" + t.Name, label: t.Name, sha: t.TargetSHA, detail: shortSHA(t.TargetSHA), prefix: "T"})
				}
			}
			for _, c := range a.commits {
				rows = append(rows, paletteChoice{id: c.SHA, label: shortSHA(c.SHA), sha: c.SHA, detail: c.Subject, prefix: "C"})
			}
		}
		a.choosePalette(title, a.paletteText("Base branch or commit", "起始分支或提交"), rows, func(row paletteChoice) {
			q.Base = row.id
			if row.prefix == "T" || row.prefix == "C" {
				q.Base = row.sha
			}
			a.runPaletteFlow(title, q)
		})
	})
}

func (a *app) choosePaletteFlowBranch(title, prefix string, q gitrepo.FlowCommand) {
	var rows []paletteChoice
	for _, b := range q.Branches {
		name, match := strings.CutPrefix(b.Name, prefix)
		if match && name != "" && b.Name != q.Config["gitflow.branch.master"] && b.Name != q.Config["gitflow.branch.develop"] {
			rows = append(rows, paletteChoice{id: name, label: b.Name, sha: b.SHA, detail: shortSHA(b.SHA)})
		}
	}
	if len(rows) == 0 {
		a.queue("info", title, a.paletteText("No matching local branches", "没有符合条件的本地分支"))
		return
	}
	a.choosePalette(title, a.paletteText("Choose a branch", "选择分支"), rows, func(row paletteChoice) {
		q.Name = row.id
		message := fmt.Sprintf("%s (%s)", row.label, shortSHA(row.sha))
		switch q.Action {
		case "publish":
			i := slices.IndexFunc(a.remotes, func(r gitrepo.Remote) bool { return r.Name == q.Config.Origin() })
			if i < 0 {
				a.queue("warning", title, a.paletteText("Missing Git Flow remote", "找不到 Git Flow 远端")+" "+q.Config.Origin())
				return
			}
			q.Remote = a.remotes[i]
			message += " → " + q.Remote.Name
		case "finish":
			message += "\n" + a.paletteText("Merge this branch and remove the local topic branch according to your Git Flow settings.", "合并此分支，并按 Git Flow 设置清理本地主题分支。")
		case "rebase":
			base := q.Config["gitflow.branch."+row.label+".base"]
			if base == "" {
				base = q.Config["gitflow.branch.develop"]
			}
			message += " → " + base
		}
		confirm := func() {
			a.askConfirm(title, message, title, q.Action == "finish", func() { a.runPaletteFlow(title, q) })
		}
		if q.Action == "finish" && (q.Kind == "hotfix" || q.Kind == "release") {
			a.choosePalette(title, row.label, []paletteChoice{{id: "tag", label: a.paletteText("Create tag…", "创建标签…")}, {id: "skip", label: a.paletteText("Finish without a tag", "完成并跳过标签")}}, func(choice paletteChoice) {
				if choice.id == "skip" {
					confirm()
					return
				}
				a.inputPalette(title, a.paletteText("Tag message", "标签说明"), q.Name, func(text string) { q.TagMessage = text; confirm() })
			})
		} else {
			confirm()
		}
	})
}

func (a *app) runPaletteFlow(title string, q gitrepo.FlowCommand) {
	a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error { return r.RunFlow(ctx, q) })
}

func (a *app) editPaletteFlowConfig(title string, config gitrepo.FlowConfig) {
	var rows []paletteChoice
	add := func(key, en, zh string) {
		rows = append(rows, paletteChoice{id: key, label: a.paletteText(en, zh), detail: config[key]})
	}
	add("gitflow.branch.master", "Production branch", "生产分支")
	add("gitflow.branch.develop", "Development branch", "开发分支")
	for _, kind := range flowFamilies {
		add("gitflow.prefix."+kind.id, kind.en+" branch prefix", kind.zh+"分支前缀")
	}
	add("gitflow.prefix.versiontag", "Version tag prefix", "版本标签前缀")
	add("gitflow.path.hooks", "Git hooks directory", "Git 钩子目录")
	a.choosePalette(title, a.paletteText("Choose a setting", "选择配置项"), rows, func(row paletteChoice) {
		save := func(value string) {
			a.runPaletteExtension(title, func(ctx context.Context, r *gitrepo.Repository) error {
				return r.SetFlowConfig(ctx, config, row.id, value)
			})
		}
		if strings.HasPrefix(row.id, "gitflow.branch.") {
			var branches []paletteChoice
			for _, b := range a.branches {
				branches = append(branches, paletteChoice{id: b.Name, label: b.Name, detail: shortSHA(b.SHA)})
			}
			a.choosePalette(title, row.label, branches, func(branch paletteChoice) { save(branch.id) })
			return
		}
		edit := func() { a.inputPalette(title, row.label, config[row.id], save) }
		if config[row.id] == "" || !strings.HasPrefix(row.id, "gitflow.prefix.") {
			edit()
			return
		}
		a.choosePalette(title, row.label, []paletteChoice{{id: "edit", label: a.paletteText("Edit prefix…", "编辑前缀…"), detail: config[row.id]}, {id: "clear", label: a.paletteText("Use an empty prefix", "使用空前缀")}}, func(choice paletteChoice) {
			if choice.id == "clear" {
				save("")
			} else {
				edit()
			}
		})
	})
}

func (a *app) choosePaletteFlowInit(title string, q gitrepo.FlowCommand) {
	existing := func(name string) bool {
		return slices.ContainsFunc(q.Branches, func(b gitrepo.Branch) bool { return b.Name == name })
	}
	confirm := func(development string) {
		q.Development = development
		message := fmt.Sprintf(a.paletteText("Initialize Git Flow using %s for production and %s for development?", "使用 %s 作为生产分支、%s 作为开发分支初始化 Git Flow？"), q.Production, development)
		if q.Expected.Head == "" {
			message += "\n" + a.paletteText("Git Flow will create an initial commit.", "Git Flow 将创建初始提交。")
		}
		a.askConfirm(title, message, title, false, func() { a.runPaletteFlow(title, q) })
	}
	develop := func(production string) {
		q.Production = production
		if configured := q.Config["gitflow.branch.develop"]; existing(configured) {
			confirm(configured)
			return
		}
		label := a.paletteText("Development branch", "开发分支")
		if len(q.Branches) <= 1 {
			initial := "develop"
			if production == initial {
				initial = "integration"
			}
			a.inputPalette(title, label, initial, confirm)
			return
		}
		var choices []paletteChoice
		for _, b := range q.Branches {
			if b.Name != production {
				choices = append(choices, paletteChoice{id: b.Name, label: b.Name, detail: shortSHA(b.SHA)})
			}
		}
		a.choosePalette(title, label, choices, func(row paletteChoice) { confirm(row.id) })
	}
	if configured := q.Config["gitflow.branch.master"]; existing(configured) {
		develop(configured)
		return
	}
	label := a.paletteText("Production branch", "生产分支")
	if len(q.Branches) == 0 {
		initial := q.Expected.Branch
		if initial == "" {
			initial = "main"
		}
		a.inputPalette(title, label, initial, develop)
		return
	}
	var choices []paletteChoice
	for _, b := range q.Branches {
		choices = append(choices, paletteChoice{id: b.Name, label: b.Name, detail: shortSHA(b.SHA)})
	}
	a.choosePalette(title, label, choices, func(row paletteChoice) { develop(row.id) })
}
