package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/LHPalma/gitarias/internal/ignore"
	"github.com/spf13/cobra"
)

type ignoreListOptions struct {
	formatOptions
	expand     bool
	expandDirs []string
}

func newIgnoreCommand(runner Runner) *cobra.Command {
	command := &cobra.Command{
		Use:   "ignore",
		Short: "Lê e edita o que o repositório ignora",
	}

	command.AddCommand(newIgnoreListCommand(runner))
	command.AddCommand(newIgnoreAddCommand(runner))

	return command
}

func newIgnoreListCommand(runner Runner) *cobra.Command {
	var options ignoreListOptions

	command := &cobra.Command{
		Use:   "list",
		Short: "Lista o que está sendo ignorado, com a regra que ignorou",
		Args:  cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runIgnoreList(command, ignore.NewRepo(runner), options)
		},
	}

	command.Flags().BoolVar(&options.expand, "expand", false, "lista arquivo a arquivo em vez de colapsar o diretório ignorado")
	command.Flags().StringArrayVar(&options.expandDirs, "expand-dir", nil,
		"expande só o(s) diretório(s) informado(s) em vez de colapsar; repetível, incompatível com --expand")
	options.register(command)

	if err := command.RegisterFlagCompletionFunc("expand-dir", completeExpandDir(runner)); err != nil {
		panic(err)
	}

	return command
}

func runIgnoreList(command *cobra.Command, repo *ignore.Repo, options ignoreListOptions) error {
	chosen, err := options.resolve(command)
	if err != nil {
		return err
	}

	if options.expand && len(options.expandDirs) > 0 {
		return fmt.Errorf("--expand já expande tudo; --expand-dir junto seria descartado em silêncio")
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	entries, err := repo.List(ctx, options.expand, options.expandDirs)
	if err != nil {
		return err
	}

	data := ignoredTable{entries: entries, expand: options.expand}

	return emit(command.OutOrStdout(), options.output, "ignorados", chosen, data)
}

type ignoreAddOptions struct {
	local  bool
	global bool
	force  bool
}

func newIgnoreAddCommand(runner Runner) *cobra.Command {
	var options ignoreAddOptions

	command := &cobra.Command{
		Use:   "add <padrão>",
		Short: "Adiciona um padrão ao que é ignorado",
		Args:  cobra.ExactArgs(1),
		RunE: func(command *cobra.Command, args []string) error {
			return runIgnoreAdd(command, ignore.NewRepo(runner), args[0], options)
		},
	}

	command.Flags().BoolVar(&options.local, "local", false,
		"grava em .git/info/exclude em vez de <raiz>/.gitignore — vale só neste clone")
	command.Flags().BoolVar(&options.global, "global", false,
		"grava em core.excludesFile em vez de <raiz>/.gitignore — vale na máquina inteira")
	command.Flags().BoolVar(&options.force, "force", false,
		"só com --global: configura e cria o core.excludesFile quando ele ainda não existe, em vez de recusar")

	return command
}

func runIgnoreAdd(command *cobra.Command, repo *ignore.Repo, pattern string, options ignoreAddOptions) error {
	if pattern == "" {
		return fmt.Errorf("padrão vazio")
	}
	if options.local && options.global {
		return fmt.Errorf("--local e --global são exclusivos")
	}
	if options.force && !options.global {
		return fmt.Errorf("--force só faz sentido com --global")
	}

	destination := ignore.Repository
	switch {
	case options.local:
		destination = ignore.Local
	case options.global:
		destination = ignore.Global
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	result, err := repo.Add(ctx, pattern, destination, options.force)
	if err != nil {
		return err
	}

	return reportAdd(command.OutOrStdout(), result)
}

func reportAdd(output io.Writer, result ignore.AddResult) error {
	if result.CoveredBy != nil {
		_, err := fmt.Fprintf(output, "Já está coberto por %s (%s:%d); nada foi gravado.\n",
			result.CoveredBy.Pattern, result.CoveredBy.Source, result.CoveredBy.Line)
		return err
	}

	if result.Duplicate {
		_, err := fmt.Fprintf(output, "%s já está em %s; nada foi gravado.\n", result.Pattern, result.Path)
		return err
	}

	if _, err := fmt.Fprintf(output, "Adicionado %s a %s.\n", result.Pattern, result.Path); err != nil {
		return err
	}

	if result.Unmatched {
		if _, err := fmt.Fprintln(output, "Aviso: a regra não pegou nenhum caminho existente agora — pode ser regra preventiva."); err != nil {
			return err
		}
	}

	if len(result.AlreadyTracked) > 0 {
		_, err := fmt.Fprintf(output,
			"Aviso: já rastreados, a regra não vai tirá-los do controle de versão: %s. Use git rm --cached se quiser.\n",
			strings.Join(result.AlreadyTracked, ", "))
		return err
	}

	return nil
}
