# gitarias

CLI de utilitários git em binário único. O binário chama `gtr`.

Não é um cliente de GitHub: não pede token, não faz requisição de rede e não
escreve em repositório remoto. O que a ferramenta faz é orquestrar o `git` que
já está na sua máquina.

## Instalação

Requer Go 1.24.7 ou superior para compilar, e `git` no `PATH` para rodar.

```bash
git clone https://github.com/LHPalma/gitarias
cd gitarias
CGO_ENABLED=0 go build -o gtr .
```

O `CGO_ENABLED=0` **não é opcional**. Sem ele, em Linux com toolchain C
presente, o binário sai dinamicamente ligado à libc e exige `GLIBC_2.34` na
máquina de destino — o que anula a promessa de distribuir copiando um arquivo.
Em macOS e Windows o problema não existe, mas a variável não atrapalha.

## Comandos

### `gtr branches`

Lista as branches locais cujo trabalho já está contido na branch base, dizendo
como cada uma chegou lá.

```
$ gtr branches
Base: main (encontrada localmente)

Branches locais já mergeadas (3):
  fix-typo     mergeada
  feat-export  rebaseada
  feat-login   squashada

Use --clean para deletar.
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--clean` | `false` | Deleta as branches listadas, após confirmação |
| `--base <branch>` | vazio | Define a base; vazio aciona a detecção automática |
| `--force` | `false` | Com `--clean`, força a deleção das squashadas e rebaseadas |

Com `--clean`, nada é apagado sem resposta afirmativa. São aceitos `y`, `yes`,
`s` e `sim`, em qualquer caixa — **qualquer outra entrada, inclusive Enter
vazio, cancela**.

**Por que existem três rótulos.** Quando um PR é mergeado por squash ou por
rebase, o commit de ponta da branch não vira ancestral da base: o merge cria
commits novos, com o mesmo conteúdo e paternidade diferente. Para o git a
branch continua "não mergeada", e é por isso que ela nunca some da sua lista
por mais que você limpe. O `gtr` compara o *conteúdo* — se o trabalho da branch
já está na base, ele diz por qual caminho.

Numa branch de um commit só, squash e rebase produzem exatamente o mesmo
resultado e não há como distinguir; nesse caso o rótulo é `squashada`.

**O `--clean` sozinho não apaga essas duas.** O `git branch -d` se recusa a
apagar branch que não seja ancestral da base, e o `gtr` não contorna isso pelas
suas costas:

```
$ gtr branches --clean
Base: main (encontrada localmente)

Branches locais já mergeadas (3):
  fix-typo     mergeada
  feat-export  rebaseada
  feat-login   squashada

2 branch(es) ficam de fora: o git recusa apagá-las com -d. Use --force para forçar.
Deletar 1 branch(es)? [y/N] n
Cancelado, nada foi deletado.
```

Com `--force` as três entram, e só então o `-D` é usado — exclusivamente nas
squashadas e rebaseadas, nunca nas demais. O que a flag libera é estreito: ela
autoriza o `gtr` a confiar na própria comparação de conteúdo quando o git se
recusa por ancestralidade, e nada além disso.

**Branch em uso por outro working tree fica de fora.** O git recusa apagá-la,
inclusive com `-D`, então oferecê-la seria prometer o que a ferramenta não pode
cumprir. Ela aparece à parte, com o caminho e as três formas de soltar:

```
1 branch(es) em uso por outro working tree, fora da lista:
  presa  ~/projeto-fix

O git recusa apagá-las, mesmo com --force. Para soltar, escolha um:
  git -C <caminho> checkout --detach   solta a branch e preserva o working tree
  git worktree remove <caminho>        apaga o working tree, inclusive arquivos ignorados
  git worktree prune                   quando o diretório já sumiu
```

O `gtr` não executa nenhuma delas. As três custam coisas diferentes — perder o
working tree, perder alterações não commitadas, ou só mover o `HEAD` —, e
`git worktree remove` apaga **arquivos ignorados** sem avisar, o que inclui
`.env` e afins.

**Como a base é escolhida**, parando no primeiro que funcionar:

1. o valor de `--base`, se informado;
2. a branch apontada por `origin/HEAD`, se existir;
3. `main` ou `master` local, a primeira que existir.

Se nenhum funcionar, o comando falha pedindo `--base`. A saída sempre informa
qual caminho foi usado.

### `gtr worktrees`

Lista os working trees do repositório. O `*` marca aquele de onde o comando foi
chamado, e acompanha você entre diretórios.

```
$ gtr worktrees
Working trees (4):
* ~/projeto           main
  ~/projeto-fix       fix-login
  ~/projeto-sumido    antiga     podável: gitdir file points to non-existent location
  ~/projeto-trancado  release    trancado: hd externo desconectado
```

No lugar da branch aparece `(bare)` ou `(HEAD destacado em <sha>)` quando não
houver uma. O comando não tem flags.

### `gtr ignore list`

Lista o que está sendo ignorado e **por qual regra** — pergunta que o git só
responde compondo dois comandos de plumbing.

```
$ gtr ignore list
Ignorados (4):
  CAMINHO             ORIGEM               PADRÃO
  app.log             .gitignore:2         *.log
  local-only/         .git/info/exclude:1  local-only/
  node_modules/       .gitignore:1         node_modules/
  relatório 2026.csv  .gitignore:4         relat*.csv

Diretório ignorado conta como uma linha só. Use --expand para listar arquivo a arquivo.
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--expand` | `false` | Lista arquivo a arquivo em vez de colapsar o diretório |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

**A coluna `ORIGEM` é o ponto do comando.** Três arquivos diferentes podem
estar ignorando algo, e eles não significam a mesma coisa: `.gitignore` é
convenção do time e vale para todo mundo, `.git/info/exclude` vale só no seu
clone, e o `core.excludesFile` vale em todos os repositórios da sua máquina.
Sem essa coluna, "por que esse arquivo está sendo ignorado" continua sem
resposta.

**Diretório inteiramente ignorado sai como uma linha.** Sem isso, um
repositório com `node_modules` cospe centenas de milhares de linhas. O
`--expand` abre.

**Arquivo já rastreado não aparece**, mesmo casando com uma regra — o git
continua rastreando quem já estava dentro. É a confusão número um do
`.gitignore`, e vale saber ao procurar algo que "deveria ter sumido".

**Para script e para planilha:**

```
$ gtr ignore list --format csv
origem,linha,padrão,caminho
.gitignore,2,*.log,app.log
.git/info/exclude,1,local-only/,local-only/
.gitignore,1,node_modules/,node_modules/
.gitignore,4,relat*.csv,relatório 2026.csv
```

**O `--output` recebe um caminho, não só um nome.** Relativo ou absoluto,
dentro ou fora do repositório:

```bash
gtr ignore list --format csv  --output ignorados             # ./ignorados.csv
gtr ignore list --format json --output /tmp/ignorados.json   # absoluto
gtr ignore list --format tsv  --output ../fora-do-repo       # ../fora-do-repo.tsv

mkdir -p relatorios/2026                                     # o diretório tem de existir
gtr ignore list --format csv  --output relatorios/2026/ign   # relatorios/2026/ign.csv
```

A extensão do formato é acrescentada **só quando falta**. Se você deu uma, ela
é respeitada, mesmo não batendo com o formato — a ferramenta não renomeia o que
você nomeou.

O `gtr` não cria árvore de diretórios por conta própria; quando falta um, ele
diz qual. E caminho terminado em `/` é recusado, porque nomeia um diretório e
não um arquivo:

```
$ gtr ignore list --format csv --output relatorios/
erro: "relatorios/" nomeia um diretório; informe o caminho do arquivo, como "relatorios/ignorados.csv"
```

**Só o arquivo leva o BOM UTF-8 na frente.** Sem ele o Excel em português lê
`relatório` como `relatÃ³rio`; num pipe, porém, os três bytes grudariam no
primeiro campo e sujariam quem for parsear. Para abrir no Excel em português,
`--separator ';'` costuma ser o par que falta.

## O que a ferramenta nunca faz

Estas não são configurações — são propriedades do código:

- **Nunca apaga branch cujo trabalho não esteja na base.** A deleção usa
  `git branch -d`, e o `-D` só aparece com `--force`, aplicado apenas às
  branches que a comparação de conteúdo provou já integradas. Branch com
  trabalho pendente não é listada, e o que não é listado não é apagado nem
  com `--force`.
- **Nunca toca em branch remota.** A consulta é restrita a `refs/heads/` e não
  existe caminho de código que chame `git push --delete`. Branch apagada
  localmente continua íntegra no servidor.
- **Nunca apaga a branch atual, a base, a `main` ou a `master`**, mesmo quando
  não são a base.
- **Nunca mexe em outro working tree.** Branch em uso em outro diretório de
  trabalho fica fora da lista, com o caminho e as formas de soltar — a escolha
  entre elas é sua, porque custam coisas diferentes.
- **Nunca roda através de um shell.** Os comandos git são invocados
  diretamente, sem `sh -c`. Uma branch com nome esquisito chega ao git como
  argumento literal.

## Saída e códigos

A listagem vai para o `stdout` e os erros para o `stderr`, então
`gtr branches | grep feat` continua limpo. Fora de um repositório git, sem base
determinável ou com falha de deleção, a saída é 1:

```
$ gtr branches --base nao-existe
erro: a branch base "nao-existe" não existe neste repositório
$ echo $?
1
```

Numa deleção parcial o comando tenta todas as branches antes de sair com 1 —
uma falha individual não aborta o resto.

## Autocompletar

O `gtr completion <bash|zsh|fish|powershell>` gera o script de autocomplete.

## Estado

Os comandos `branches`, `worktrees` e `ignore list` estão no ar. Planejados:
seleção interativa de quais branches apagar, `gtr ignore add`, `stats`,
`changelog` e arquivo de configuração opcional. O `--format` existe hoje só no
`ignore list`; o `branches` e o `worktrees` ainda não o adotaram.

## Contribuindo

Antes de abrir PR: `gofmt -l .` vazio, `go vet ./...` limpo e `go test ./...`
verde — os mesmos três portões que o CI aplica.
