package gitrepo

import (
	"context"
	"errors"
	"maps"
	"os/exec"
	"strings"
)

type FlowConfig map[string]string

func (r *Repository) FlowConfig(ctx context.Context) (FlowConfig, error) {
	out, err := run(ctx, r.Root, "", "config", "--null", "--get-regexp", "^gitflow\\.")
	var exit *exec.ExitError
	if err != nil && !(errors.As(err, &exit) && exit.ExitCode() == 1) {
		return nil, err
	}
	config := FlowConfig{}
	for entry := range strings.SplitSeq(strings.TrimSuffix(out, "\x00"), "\x00") {
		if key, value, ok := strings.Cut(entry, "\n"); ok {
			config[key] = value
		}
	}
	return config, nil
}

func (c FlowConfig) Origin() string {
	if name := c["gitflow.origin"]; name != "" {
		return name
	}
	return "origin"
}

func (c FlowConfig) DefaultBase(kind string) string {
	if kind == "hotfix" || kind == "support" {
		return c["gitflow.branch.master"]
	}
	return c["gitflow.branch.develop"]
}

type FlowCommand struct {
	Action, Kind, Name, Base, TagMessage string
	Production, Development              string
	Config                               FlowConfig
	Expected                             Status
	Branches                             []Branch
	Remote                               Remote
}

func flowKind(kind string) bool {
	switch kind {
	case "feature", "bugfix", "release", "hotfix", "support":
		return true
	}
	return false
}

func (r *Repository) SetFlowConfig(ctx context.Context, expected FlowConfig, key, value string) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.idle(ctx); err != nil {
		return err
	}
	current, err := r.FlowConfig(ctx)
	if err != nil {
		return err
	}
	if !maps.Equal(current, expected) {
		return ErrIntegrationTargetChanged
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return errors.New("Invalid Git Flow setting")
	}
	switch key {
	case "gitflow.branch.master", "gitflow.branch.develop":
		other := "gitflow.branch.master"
		if key == other {
			other = "gitflow.branch.develop"
		}
		if value == expected[other] {
			return errors.New("Production and development branches must differ")
		}
		if err := r.branchName(ctx, value); err != nil {
			return err
		}
		if _, err := r.resolve(ctx, "refs/heads/"+value); err != nil {
			return err
		}
	case "gitflow.prefix.feature", "gitflow.prefix.bugfix", "gitflow.prefix.release", "gitflow.prefix.hotfix", "gitflow.prefix.support", "gitflow.prefix.versiontag":
		if err := r.branchName(ctx, value+"example"); err != nil {
			return err
		}
	case "gitflow.path.hooks":
	default:
		return errors.New("Unknown Git Flow setting")
	}
	_, err = run(ctx, r.Root, "", "config", "--local", key, value)
	return err
}

func (r *Repository) RunFlow(ctx context.Context, q FlowCommand) error {
	r.writeMu.Lock()
	defer r.writeMu.Unlock()
	if err := r.checkPaletteStatus(ctx, q.Expected); err != nil {
		return err
	}
	if err := r.clean(ctx); err != nil {
		return err
	}
	config, err := r.FlowConfig(ctx)
	if err != nil {
		return err
	}
	if !maps.Equal(config, q.Config) {
		return ErrIntegrationTargetChanged
	}
	if q.Action == "init" {
		if err := r.noIgnored(ctx); err != nil {
			return err
		}
		return r.initPaletteFlow(ctx, q)
	}
	if !flowKind(q.Kind) || q.Kind == "support" && (q.Action == "finish" || q.Action == "publish") {
		return errors.New("Unsupported Git Flow action")
	}
	prefix, ok := config["gitflow.prefix."+q.Kind]
	if !ok || config["gitflow.branch.master"] == "" || config["gitflow.branch.develop"] == "" {
		return errors.New("Initialize Git Flow first")
	}
	if err := r.branchName(ctx, q.Name); err != nil {
		return err
	}
	target := prefix + q.Name
	if target == config["gitflow.branch.master"] || target == config["gitflow.branch.develop"] {
		return errors.New("Select a topic branch")
	}
	if err := r.branchName(ctx, target); err != nil {
		return err
	}
	refs := []string{config["gitflow.branch.master"], config["gitflow.branch.develop"]}
	base := config["gitflow.branch."+target+".base"]
	if base != "" {
		refs = append(refs, base)
	}
	args := []string{"-c", "core.editor=true"}
	command := []string{"flow", q.Kind, q.Action}
	switch q.Action {
	case "start":
		if q.Base == "" {
			return errors.New("Choose a base branch or commit")
		}
		if q.Kind == "support" && oidRE.MatchString(q.Base) {
			if _, err := r.resolve(ctx, q.Base); err != nil {
				return err
			}
		} else {
			refs = append(refs, q.Base)
		}
		if err := r.protectIgnored(ctx, q.Base); err != nil {
			return err
		}
		command = append(command, q.Name, q.Base)
	case "finish", "rebase", "publish":
		refs = append(refs, target)
		if q.Action == "publish" {
			if q.Remote.Name != config.Origin() {
				return ErrRemoteTargetChanged
			}
			if err := r.checkRemoteTarget(ctx, q.Remote); err != nil {
				return err
			}
			if err := r.checkRemote(ctx, q.Remote.Name, false); err != nil {
				return err
			}
			if err := r.checkRemote(ctx, q.Remote.Name, true); err != nil {
				return err
			}
			if err := r.protectIgnored(ctx, target); err != nil {
				return err
			}
		} else if err := r.noIgnored(ctx); err != nil {
			return err
		}
		if q.Action == "finish" {
			// Finishing is a local operation. AVH otherwise deletes an existing
			// remote branch even when its normal push option is disabled.
			for _, setting := range []string{"keepremote=true", "push=false", "pushproduction=false", "pushdevelop=false", "pushtag=false"} {
				args = append(args, "-c", "gitflow."+q.Kind+".finish."+setting)
			}
			if q.Kind == "release" || q.Kind == "hotfix" {
				if q.TagMessage == "" {
					command = append(command, "-n")
				} else {
					command = append(command, "-m", q.TagMessage)
				}
			}
		}
		command = append(command, q.Name)
	default:
		return errors.New("Unsupported Git Flow action")
	}
	for _, name := range refs {
		var expected string
		for _, b := range q.Branches {
			if b.Name == name {
				expected = b.SHA
				break
			}
		}
		if !oidRE.MatchString(expected) {
			return ErrIntegrationTargetChanged
		}
		actual, err := r.resolve(ctx, "refs/heads/"+name)
		if err != nil || actual != expected {
			return errors.Join(ErrIntegrationTargetChanged, err)
		}
	}
	_, err = flowBinary().run(ctx, r.Root, "", append(args, command...)...)
	return err
}

// AVH only offers to create a development branch when the production branch
// is the only local branch. Feed answers to its prompts, retaining any
// partially configured settings when initialization is retried.
func (r *Repository) initPaletteFlow(ctx context.Context, q FlowCommand) error {
	current, err := r.Branches(ctx)
	if err != nil {
		return err
	}
	names := func(branches []Branch) map[string]string {
		result := make(map[string]string, len(branches))
		for _, b := range branches {
			result[b.Name] = b.SHA
		}
		return result
	}
	refs := names(current)
	if !maps.Equal(refs, names(q.Branches)) {
		return ErrIntegrationTargetChanged
	}
	if q.Production == "" && q.Development == "" {
		_, err = flowBinary().run(ctx, r.Root, "", "flow", "init", "-d")
		return err
	}
	if err := r.branchName(ctx, q.Production); err != nil {
		return err
	}
	if err := r.branchName(ctx, q.Development); err != nil {
		return err
	}
	if q.Production == q.Development || len(refs) > 0 && refs[q.Production] == "" || len(refs) > 1 && refs[q.Development] == "" {
		return errors.New("Choose distinct production and development branches from this repository")
	}
	var input strings.Builder
	for _, v := range []struct{ key, name string }{{"gitflow.branch.master", q.Production}, {"gitflow.branch.develop", q.Development}} {
		if configured := q.Config[v.key]; refs[configured] != "" {
			if configured != v.name {
				return ErrIntegrationTargetChanged
			}
			continue
		}
		input.WriteString(v.name)
		input.WriteByte('\n')
	}
	_, err = flowBinary().run(ctx, r.Root, input.String(), "-c", "gitflow.init.defaults=false", "flow", "init")
	return err
}
