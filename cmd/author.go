package cmd

import (
	"fmt"
	"strings"

	"github.com/LHPalma/gitarias/internal/author"
	"github.com/LHPalma/gitarias/internal/ui"
	"github.com/spf13/cobra"
)

type authorOptions struct {
	name   string
	email  string
	base   string
	until  string
	commit string
}

func newAuthorCommand(runner Runner) *cobra.Command {
	var options authorOptions

	command := &cobra.Command{
		Use:     "author",
		Aliases: []string{"blame-someone-else"},
		Short:   "Reescreve a autoria de commits, com confirmação",
		Args:    cobra.NoArgs,
		RunE: func(command *cobra.Command, args []string) error {
			return runAuthor(command, author.NewRepo(runner), options)
		},
	}

	command.Flags().StringVar(&options.name, "name", "", "o nome a atribuir (obrigatório)")
	command.Flags().StringVar(&options.email, "email", "", "o e-mail a atribuir (obrigatório)")
	command.Flags().StringVar(&options.base, "base", "",
		"reescreve <base>..HEAD em vez de só o commit mais recente; <base> fica de fora, reescreve a cadeia de SHAs dali pra frente")
	command.Flags().StringVar(&options.until, "until", "",
		"com --base, fecha o intervalo antes do HEAD; o que vem depois de <until> é preservado, só reencaixado em cima")
	command.Flags().StringVar(&options.commit, "commit", "",
		"reescreve só esse commit, preservando tudo antes e depois; incompatível com --base e --until. Não funciona no commit raiz, que não tem pai")

	return command
}

func runAuthor(command *cobra.Command, repo *author.Repo, options authorOptions) error {
	if options.name == "" || options.email == "" {
		return fmt.Errorf("--name e --email são obrigatórios")
	}
	if options.commit != "" && (options.base != "" || options.until != "") {
		return fmt.Errorf("--commit não se combina com --base nem --until")
	}
	if options.until != "" && options.base == "" {
		return fmt.Errorf("--until só faz sentido com --base")
	}

	base, until := options.base, options.until
	if options.commit != "" {
		base = options.commit + "^"
		until = options.commit
	}

	ctx := command.Context()

	if err := repo.Ensure(ctx); err != nil {
		return err
	}

	plan, err := repo.Plan(ctx, base, until)
	if err != nil {
		return err
	}

	output := command.OutOrStdout()

	switch {
	case base == "":
		fmt.Fprintf(output, "Vai reescrever o commit mais recente, hoje em nome de %s, para %s <%s>.\n",
			plan.Author, options.name, options.email)
	case options.commit != "":
		fmt.Fprintf(output, "Vai reescrever o commit %s (hoje em %s) para %s <%s>.\n",
			options.commit, strings.Join(plan.Authors, ", "), options.name, options.email)
		if plan.Tail > 0 {
			fmt.Fprintf(output, "Mais %d %s depois de %s serão preservados, só reencaixados em cima.\n",
				plan.Tail, ui.Plural(plan.Tail, "commit", "commits"), options.commit)
		}
	default:
		resolvedUntil := until
		if resolvedUntil == "" {
			resolvedUntil = "HEAD"
		}

		fmt.Fprintf(output, "Vai reescrever %d %s (%s..%s) para %s <%s>.\n",
			plan.Count, ui.Plural(plan.Count, "commit", "commits"), base, resolvedUntil, options.name, options.email)
		fmt.Fprintf(output, "Autores atuais no intervalo: %s.\n", strings.Join(plan.Authors, ", "))

		if plan.Tail > 0 {
			fmt.Fprintf(output, "Mais %d %s depois de %s serão preservados, só reencaixados em cima.\n",
				plan.Tail, ui.Plural(plan.Tail, "commit", "commits"), resolvedUntil)
		}
	}
	fmt.Fprintf(output, "Recuperável com: git reset --hard %s\n", plan.Head)

	confirmed, err := confirm(command.InOrStdin(), output, "Confirma? [y/N] ")
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Fprintln(output, "Cancelado, nada foi reescrito.")
		return nil
	}

	if err := repo.Rewrite(ctx, base, until, options.name, options.email); err != nil {
		return err
	}

	fmt.Fprintln(output, "Pronto.")

	return nil
}
