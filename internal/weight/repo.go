package weight

import (
	"context"
	"sort"
	"strconv"
	"strings"

	"github.com/LHPalma/gitarias/internal/git"
)

// batchFormat pede ao cat-file o tamanho em disco, e não o lógico. É o
// tamanho em disco que um clone baixa: texto que muda pouco vira poucos bytes
// no pack, enquanto um zip não comprime nem delta com nada.
const batchFormat = "--batch-check=%(objecttype) %(objectname) %(objectsize:disk) %(rest)"

type Repo struct {
	runner Runner
}

func NewRepo(runner Runner) *Repo {
	return &Repo{runner: runner}
}

func (repo *Repo) Ensure(ctx context.Context) error {
	return git.EnsureRepo(ctx, repo.runner)
}

// Heaviest devolve os caminhos que mais pesam, do maior para o menor, somando
// todas as versões de cada um. Um limite de zero ou menos traz todos.
func (repo *Repo) Heaviest(ctx context.Context, limit int) (Report, error) {
	total, err := repo.total(ctx)
	if err != nil {
		return Report{}, err
	}

	sizes, err := repo.sizes(ctx)
	if err != nil {
		return Report{}, err
	}

	present := repo.tracked(ctx)

	paths := make([]Path, 0, len(sizes))
	for name, weighed := range sizes {
		weighed.Name = name
		weighed.InTree = present[name]
		paths = append(paths, weighed)
	}

	sort.Slice(paths, func(first int, second int) bool {
		if paths[first].Bytes != paths[second].Bytes {
			return paths[first].Bytes > paths[second].Bytes
		}

		return paths[first].Name < paths[second].Name
	})

	if limit > 0 && len(paths) > limit {
		paths = paths[:limit]
	}

	return Report{Total: total, Paths: paths}, nil
}

// sizes soma, por caminho, o tamanho em disco de cada blob do histórico. A
// lista de objetos alimenta o cat-file pela entrada padrão, que é como os dois
// comandos foram feitos para conversar.
func (repo *Repo) sizes(ctx context.Context) (map[string]Path, error) {
	objects, err := repo.runner.Run(ctx, "rev-list", "--objects", "--all")
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(objects) == "" {
		return map[string]Path{}, nil
	}

	checked, err := repo.runner.RunWithInput(ctx, objects+"\n", "cat-file", batchFormat)
	if err != nil {
		return nil, err
	}

	sizes := map[string]Path{}
	for _, line := range strings.Split(checked, "\n") {
		name, size, ok := blob(line)
		if !ok {
			continue
		}

		weighed := sizes[name]
		weighed.Bytes += size
		weighed.Versions++
		sizes[name] = weighed
	}

	return sizes, nil
}

// blob lê uma linha do cat-file e devolve o caminho e o tamanho, ignorando o
// que não é blob com caminho — commit e tree não têm caminho, e a raiz da
// árvore vem com o campo vazio.
func blob(line string) (string, int64, bool) {
	fields := strings.SplitN(line, " ", 4)
	if len(fields) != 4 || fields[0] != "blob" {
		return "", 0, false
	}

	name := fields[3]
	if name == "" {
		return "", 0, false
	}

	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil {
		return "", 0, false
	}

	return name, size, true
}

// tracked lista o que ainda está na árvore, e não devolve erro de propósito:
// repositório sem HEAD não tem árvore, e nesse caso todo caminho do histórico
// está de fato fora dela.
//
// O -z é obrigatório. Sem ele o ls-tree cita e escapa em C os caminhos com
// não-ASCII ou aspas, enquanto o rev-list os entrega crus, e o cruzamento dos
// dois erraria justamente nesses.
func (repo *Repo) tracked(ctx context.Context) map[string]bool {
	listed, err := repo.runner.Run(ctx, "ls-tree", "-r", "-z", "--name-only", "HEAD")
	if err != nil {
		return map[string]bool{}
	}

	present := map[string]bool{}
	for _, name := range strings.Split(listed, "\x00") {
		if name != "" {
			present[name] = true
		}
	}

	return present
}

// total é o que um clone baixa: os objetos soltos mais os empacotados. O
// count-objects informa os dois em KiB, e um repositório recém-commitado tem
// tudo solto e nada no pack.
func (repo *Repo) total(ctx context.Context) (int64, error) {
	output, err := repo.runner.Run(ctx, "count-objects", "-v")
	if err != nil {
		return 0, err
	}

	var total int64
	for _, line := range strings.Split(output, "\n") {
		key, value, found := strings.Cut(line, ":")
		if !found || (key != "size" && key != "size-pack") {
			continue
		}

		if kibibytes, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64); err == nil {
			total += kibibytes * 1024
		}
	}

	return total, nil
}
