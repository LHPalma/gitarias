package branch

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

const equivalenceProbeMessage = "gtr equivalence probe"

type Repo struct {
	runner git.Runner
}

func NewRepo(runner git.Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

func (repo *Repo) ResolveBase(ctx context.Context, requested string) (Base, error) {
	if requested != "" {
		if !repo.localExists(ctx, requested) {
			return Base{}, fmt.Errorf("a branch base %q não existe neste repositório", requested)
		}
		return Base{Name: requested, Source: BaseFromFlag}, nil
	}

	if originHead, err := repo.runner.Run(ctx, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil {
		detected := strings.TrimPrefix(originHead, "origin/")
		if detected != "" && repo.localExists(ctx, detected) {
			return Base{Name: detected, Source: BaseFromOriginHead}, nil
		}
	}

	for _, candidate := range []string{"main", "master"} {
		if repo.localExists(ctx, candidate) {
			return Base{Name: candidate, Source: BaseFromLocal}, nil
		}
	}

	return Base{}, errors.New("não consegui determinar a branch base: nem main nem master existem aqui. Use --base <branch>")
}

func (repo *Repo) localExists(ctx context.Context, name string) bool {
	_, err := repo.runner.Run(ctx, "rev-parse", "--verify", "--quiet", "refs/heads/"+name)
	return err == nil
}

func (repo *Repo) Merged(ctx context.Context, base Base) ([]Branch, error) {
	protected := repo.protected(ctx, base)

	all, kinds, err := repo.classify(ctx, base, protected)
	if err != nil {
		return nil, err
	}

	var merged []Branch
	for _, byAncestry := range []bool{true, false} {
		for _, name := range all {
			kind, integrated := kinds[name]
			if !integrated || protected[name] || (kind == MergedByAncestry) != byAncestry {
				continue
			}
			merged = append(merged, Branch{Name: name, Merge: kind})
		}
	}

	return merged, nil
}

func (repo *Repo) classify(ctx context.Context, base Base, protected map[string]bool) ([]string, map[string]MergeKind, error) {
	ancestors, err := repo.refs(ctx, "--merged", base.Name)
	if err != nil {
		return nil, nil, err
	}

	kinds := make(map[string]MergeKind, len(ancestors))
	for _, name := range ancestors {
		kinds[name] = MergedByAncestry
	}

	all, err := repo.refs(ctx)
	if err != nil {
		return nil, nil, err
	}

	for _, name := range all {
		if _, contained := kinds[name]; contained || protected[name] {
			continue
		}
		if kind, equivalent := repo.equivalence(ctx, base, name); equivalent {
			kinds[name] = kind
		}
	}

	return all, kinds, nil
}

func (repo *Repo) refs(ctx context.Context, filters ...string) ([]string, error) {
	args := append([]string{"for-each-ref", "refs/heads/"}, filters...)
	output, err := repo.runner.Run(ctx, append(args, "--format=%(refname:short)")...)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, line := range strings.Split(output, "\n") {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}

	return names, nil
}

func (repo *Repo) protected(ctx context.Context, base Base) map[string]bool {
	protected := map[string]bool{
		base.Name: true,
		"main":    true,
		"master":  true,
	}
	if currentBranch, err := repo.runner.Run(ctx, "branch", "--show-current"); err == nil && currentBranch != "" {
		protected[currentBranch] = true
	}

	return protected
}

func (repo *Repo) equivalence(ctx context.Context, base Base, name string) (MergeKind, bool) {
	mergeBase, err := repo.runner.Run(ctx, "merge-base", base.Name, name)
	if err != nil || mergeBase == "" {
		return MergedByAncestry, false
	}

	if repo.squashedInto(ctx, base, name, mergeBase) {
		return MergedBySquash, true
	}
	if repo.rebasedInto(ctx, base, name) {
		return MergedByRebase, true
	}

	return MergedByAncestry, false
}

func (repo *Repo) squashedInto(ctx context.Context, base Base, name string, mergeBase string) bool {
	tree, err := repo.runner.Run(ctx, "rev-parse", name+"^{tree}")
	if err != nil || tree == "" {
		return false
	}

	probe, err := repo.runner.Run(ctx, "commit-tree", tree, "-p", mergeBase, "-m", equivalenceProbeMessage)
	if err != nil || probe == "" {
		return false
	}

	present, err := repo.equivalents(ctx, base.Name, probe)

	return err == nil && len(present) == 1 && present[0]
}

func (repo *Repo) rebasedInto(ctx context.Context, base Base, name string) bool {
	present, err := repo.equivalents(ctx, base.Name, name)
	if err != nil || len(present) == 0 {
		return false
	}

	for _, found := range present {
		if !found {
			return false
		}
	}

	return true
}

func (repo *Repo) equivalents(ctx context.Context, base string, head string) ([]bool, error) {
	output, err := repo.runner.Run(ctx, "cherry", base, head)
	if err != nil {
		return nil, err
	}

	var present []bool
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "- "):
			present = append(present, true)
		case strings.HasPrefix(line, "+ "):
			present = append(present, false)
		}
	}

	return present, nil
}

func (repo *Repo) Delete(ctx context.Context, branches []Branch, forceEquivalent bool) []DeleteResult {
	results := make([]DeleteResult, 0, len(branches))

	for _, target := range branches {
		flag := "-d"
		if forceEquivalent && target.Merge != MergedByAncestry {
			flag = "-D"
		}
		_, err := repo.runner.Run(ctx, "branch", flag, target.Name)
		results = append(results, DeleteResult{Branch: target, Err: err})
	}

	return results
}

func (repo *Repo) Tree(ctx context.Context, base Base) ([]Layer, error) {
	protected := repo.protected(ctx, base)

	all, kinds, err := repo.classify(ctx, base, protected)
	if err != nil {
		return nil, err
	}

	tips, err := repo.tips(ctx)
	if err != nil {
		return nil, err
	}

	layers := make([]Layer, 0, len(all))
	for _, name := range all {
		if name == base.Name {
			continue
		}

		kind, integrated := kinds[name]
		layers = append(layers, Layer{
			Branch: Branch{Name: name, Merge: kind},
			Parent: repo.parentOf(ctx, base, name, tips),
			Merged: integrated,
		})
	}

	return layers, nil
}

func (repo *Repo) tips(ctx context.Context) (map[string][]string, error) {
	output, err := repo.runner.Run(ctx, "for-each-ref", "refs/heads/", "--format=%(objectname) %(refname:short)")
	if err != nil {
		return nil, err
	}

	tips := map[string][]string{}
	for _, line := range strings.Split(output, "\n") {
		sha, name, complete := strings.Cut(strings.TrimSpace(line), " ")
		if !complete || sha == "" || name == "" {
			continue
		}
		tips[sha] = append(tips[sha], name)
	}

	return tips, nil
}

func (repo *Repo) parentOf(ctx context.Context, base Base, name string, tips map[string][]string) string {
	output, err := repo.runner.Run(ctx, "rev-list", name, "^"+base.Name)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(output, "\n") {
		sha := strings.TrimSpace(line)
		if sha == "" {
			continue
		}
		for _, candidate := range tips[sha] {
			if candidate != name && candidate != base.Name {
				return candidate
			}
		}
	}

	return ""
}
