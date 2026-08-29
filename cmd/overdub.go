package cmd

import (
	"fmt"

	"github.com/LHPalma/gitarias/internal/commits"
	"github.com/LHPalma/gitarias/internal/exec"
	"github.com/LHPalma/gitarias/internal/overdub"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type overdubOptions struct {
	until string
}

func newOverdubCommand(runner Runner, commands exec.Runner) *cobra.Command {
	var options overdubOptions

	command := &cobra.Command{
		Use:   "overdub <sha> -- <comando> [-- <comando-de-verificação>]",
		Short: "Conserta um commit no lugar, com confirmação, e recoloca o resto do histórico por cima",
		RunE: func(command *cobra.Command, args []string) error {
			return runOverdub(command, overdub.NewRepo(runner, commands), runner, commands, args, options)
		},
	}

	command.Flags().StringVar(&options.until, "until", "", "até onde reescrever; vazio significa HEAD")

	return command
}

func newOverdubSequenceStepCommand() *cobra.Command {
	return &cobra.Command{
		Use:    overdub.SequenceStepCommand + " <sha> <arquivo-do-todo>",
		Hidden: true,
		Args:   cobra.ExactArgs(2),
		RunE: func(command *cobra.Command, args []string) error {
			return overdub.MarkForEdit(args[0], args[1])
		},
	}
}

func runOverdub(command *cobra.Command, repo *overdub.Repo, verifyRunner Runner, commands exec.Runner, args []string, options overdubOptions) error {
	sha, fix, verify, err := splitAtOverdubDash(command, args)
	if err != nil {
		return err
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	plan, err := repo.Plan(ctx, sha, options.until)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()
	if _, err := fmt.Fprintf(output, "Isso vai reescrever %d %s a partir de %s %q.\n",
		plan.Count, ui.Plural(plan.Count, "commit", "commits"), plan.Target, plan.Subject); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(output, "HEAD atual: %s — se algo der errado, git reset --hard %s desfaz.\n\n",
		plan.Head, plan.Head); err != nil {
		return err
	}

	confirmed, err := confirm(command.InOrStdin(), output, "Confirma? [y/N] ")
	if err != nil {
		return err
	}
	if !confirmed {
		_, err := fmt.Fprintln(output, "Nada foi alterado.")
		return err
	}

	result, err := repo.Overdub(ctx, sha, options.until, fix[0], fix[1:])
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "Consertado. Novo HEAD: %s\n", result.NewHead[:7]); err != nil {
		return err
	}

	if verify == nil {
		return nil
	}

	return runOverdubVerify(command, verifyRunner, commands, result, verify)
}

func runOverdubVerify(command *cobra.Command, runner Runner, commands exec.Runner, result overdub.Result, verify []string) error {
	ctx := command.Context()
	checkRepo := commits.NewRepo(runner, commands)

	list, err := checkRepo.Interval(ctx, result.NewTarget+"^", result.NewHead)
	if err != nil {
		return err
	}

	results, err := checkRepo.Check(ctx, list, commits.NewArchiveExtractor(runner), verify[0], verify[1:])
	if err != nil {
		return err
	}

	data := checkedTable{base: result.NewTarget + "^", command: verify, results: results}

	if err := data.text(command.OutOrStdout()); err != nil {
		return err
	}

	return failure(data)
}

func splitAtOverdubDash(command *cobra.Command, args []string) (string, []string, []string, error) {
	dash := command.ArgsLenAtDash()

	switch {
	case dash < 0:
		return "", nil, nil, fmt.Errorf("informe o comando de conserto depois de --, como em: gtr overdub <sha> -- gofmt -w arquivo.go")
	case dash != 1:
		return "", nil, nil, fmt.Errorf("informe exatamente um commit antes do --, e vieram %d", dash)
	case len(args) == dash:
		return "", nil, nil, fmt.Errorf("o -- veio sem comando de conserto depois")
	}

	sha := args[0]
	rest := args[dash:]

	for index, token := range rest {
		if token != "--" {
			continue
		}
		if index == 0 {
			return "", nil, nil, fmt.Errorf("o -- de conserto veio vazio antes do -- de verificação")
		}
		if index == len(rest)-1 {
			return "", nil, nil, fmt.Errorf("o -- de verificação veio sem comando depois")
		}
		return sha, rest[:index], rest[index+1:], nil
	}

	return sha, rest, nil, nil
}
