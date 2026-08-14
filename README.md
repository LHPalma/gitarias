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
| `--tree` | `false` | Mostra **todas** as branches locais como árvore, cada uma sob aquela em que foi empilhada |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

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

**Nos formatos estruturados a tabela é uma só**, e a branch presa entra nela
com a coluna `presa` marcada, em vez de ficar na seção à parte:

```
$ gtr branches --format csv
branch,merge,presa,worktree
fix-typo,mergeada,não,
feat-login,squashada,não,
presa,mergeada,sim,~/projeto-fix
```

Omiti-la seria mentir por omissão: ela **está** mergeada na base. O que a
listagem de texto responde é outra pergunta — *o que eu posso limpar* —, e é
por isso que lá ela sai da lista.

**A base não é linha de tabela, então cada formato a carrega do seu jeito.** No
`csv` e no `tsv` ela vai para o `stderr`, para não entrar no meio das colunas;
no `json` ela viaja dentro do envelope:

```bash
$ gtr branches --format csv > branches.csv
Base: main (encontrada localmente)          # stderr, na tela
```

```json
{
  "base": { "name": "main", "source": "local" },
  "branches": [
    { "branch": "fix-typo", "merge": "ancestry", "held": false, "worktree": "" }
  ]
}
```

**O `merge` do JSON é token, não rótulo:** `ancestry`, `squash` e `rebase`,
enquanto o `csv` traz `mergeada`, `squashada` e `rebaseada`. Planilha é lida
por gente e JSON é lido por código — um script que casasse com `"squashada"`
quebraria calado no dia em que as mensagens forem traduzidas.

**`--clean` com `--format` é recusado.** O `--clean` pergunta, deleta e imprime
o que apagou; cruzá-lo com um formato de tabela ou jogaria o prompt no meio do
CSV ou descartaria a flag em silêncio:

```
$ gtr branches --clean --format csv
erro: --format csv não vale com --clean; --clean é interativo e imprime o que deletou
```

**`--tree` responde outra pergunta.** A listagem normal diz *o que eu posso
limpar* e por isso só mostra o que está mergeado. A árvore diz *como minhas
branches se relacionam*, e mostra todas:

```
$ gtr branches --tree
Base: main (encontrada localmente)

main
└─ camada-1        squashada
   └─ camada-2     não mergeada
      └─ camada-3  não mergeada
```

**O git não guarda quem é o pai de uma branch** — `branch.<x>.merge` aponta
para o upstream remoto, não para a branch local de onde ela saiu. O `gtr`
infere do grafo: se a base de B é exatamente a ponta de A, então B foi
empilhada sobre A, e o pai é o candidato mais próximo. Não depende de
ferramenta nenhuma de PR empilhado; funciona para quem empilha com
`git rebase --onto` desde sempre.

**Onde isso muda uma decisão.** No exemplo acima, a listagem normal diria
apenas "1 branch mergeada, use --clean para deletar". A árvore mostra que essa
branch é a **base de uma pilha viva**, com duas camadas ainda abertas em cima —
o mesmo estado, lido de outro jeito.

Nos formatos estruturados a árvore vira uma tabela plana com a coluna `pai`,
da base para o topo, que é o suficiente para um script remontar a hierarquia:

```
$ gtr branches --tree --format csv
branch,pai,estado
camada-1,main,squashada
camada-2,camada-1,não mergeada
camada-3,camada-2,não mergeada
```

`--tree` com `--clean` é recusado: a árvore lista de propósito o que não pode
ser deletado.

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
houver uma.

| Flag | Padrão | Efeito |
|---|---|---|
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

**Nos formatos estruturados as frases viram campos.** `(bare)` e
`(HEAD destacado em <sha>)` são montados para o olho; na tabela cada pedaço tem
sua coluna, e o `head` sai inteiro — truncar em sete caracteres é decisão de
tela, não do dado:

```
$ gtr worktrees --format csv
caminho,atual,branch,head,destacado,bare,trancado,motivo_trancado,podável,motivo_podável
~/projeto,sim,main,0e3dcb1e9e6353173394deb2ec915029a1b2451c,não,não,não,,não,
~/projeto-fix,não,fix-login,c98e37198e107407a5f7a82fe8d5ce97fb39479d,não,não,não,,não,
~/projeto-velho,não,,a1b7559cd4982f88bb8e44f31b7b095e2d63ba67,sim,não,sim,revisão,não,
```

No `json` os booleanos são booleanos, e não `sim`/`não`.

### `gtr commits check`

Roda um comando em **cada commit** de um intervalo, com a árvore daquele commit
isolada dos que vieram depois, e diz quais não se sustentam sozinhos.

```
$ gtr commits check main -- go test ./...
Verificando 3 commit(s) sobre main.

  verde     099cd54  refactor: share the output machinery
  VERMELHO  d901b71  feat: give branches the shared output formats
      # gitarias/cmd
      cmd/branches.go:47:9: undefined: emitBranches
  verde     ce4104a  feat: give worktrees the shared output formats
```

**A pergunta não é se o topo passa** — isso o CI já responde. É se cada commit
passa **sozinho**. Um commit que só compila junto com o seguinte não incomoda
ninguém no dia em que é escrito, e cobra a conta meses depois: é onde o
`git bisect` para sem saber dizer nada, e é o commit que não dá para reverter
isolado.

| Flag | Padrão | Efeito |
|---|---|---|
| `--verbose` | `false` | Mostra também a saída dos commits que passaram |
| `--worktree` | `false` | Extrai com `git worktree` em vez de `git archive`, para comando que precisa do `.git` |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

**O comando vem depois de `--`, e não como texto entre aspas.** A diferença não
é de estilo:

```bash
gtr commits check main -- go test ./...          # certo
gtr commits check main --run "go test ./..."     # não existe, e de propósito
```

Aceitar uma string obrigaria a ferramenta a decidir onde ela se divide, e fazer
isso direito é reimplementar um shell — com aspas, escape e expansão. Depois do
`--`, quem já separou os argumentos foi o shell de quem chamou, e eles chegam
inteiros. Um argumento com espaço continua sendo **um** argumento.

**Nada é escrito no repositório.** A árvore de cada commit sai por
`git archive` para um diretório temporário, apagado ao fim. Nenhuma ref é
criada ou movida, o índice e a árvore de trabalho não são tocados, e o `HEAD`
fica onde estava — então dá para rodar a verificação **com trabalho em
andamento**, sem guardar nada antes.

O `git` chega perto disso com `git rebase --exec`, mas por outro caminho:

```bash
git rebase --exec 'go test ./...' main
```

Ele **reescreve o histórico** para checar, e **para no primeiro vermelho**.
Quem só quer saber o estado não quer reescrever nada, e quer a lista inteira —
saber que o 1 e o 4 quebraram é diagnóstico; saber que "o 1 quebrou" é o que
obriga a rodar tudo de novo depois de cada correção.

**A árvore extraída não tem `.git`.** Comando que precise de metadado de git —
contar commits, ler tags, gerar versão a partir do `describe` — falha ali. O
`--worktree` troca a extração por `git worktree add --detach`, que resolve isso
ao custo de escrever em `.git/worktrees/` enquanto roda. **O padrão é o que não
escreve.**

A saída é 1 se qualquer commit falhar, 0 se todos passarem.

### `gtr doctor`

Confere se a máquina tem o que o `gtr` precisa.

```
$ gtr doctor
  ok  git 2.43.0
```

```
$ gtr doctor
  falta  git não encontrado no PATH
         o gtr orquestra o git da máquina e não funciona sem ele; instale em https://git-scm.com
erro: 1 checagem(ns) falharam
```

Sai 1 se alguma checagem falhar. **É o único comando que não precisa de um
repositório git** — ele diagnostica a máquina, não o repositório, e roda de
qualquer diretório.

Aceita `--format`, `--output`, `--separator` e `--no-header` como os demais. No
`json` o `state` é token — `ok`, `warning`, `failure` —, e no `csv` é o rótulo
de tela.

Hoje ele checa só o `git`. A lista cresce conforme o `gtr` ganhar dependências.

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

**O JSON de todos os comandos é um objeto, nunca um array solto.** A lista mora
sob a chave do comando — `ignored`, `branches`, `worktrees` —, e é essa casca
que dá lugar ao metadado que não cabe em coluna, como a base do `branches`:

```json
{
  "ignored": [
    { "source": ".gitignore", "line": 2, "pattern": "*.log", "path": "app.log" }
  ]
}
```

As chaves ficam em inglês porque aquela saída é lida por código; os nomes de
coluna do `csv` ficam em português porque planilha é lida por gente.

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
  argumento literal, e o comando de verificação do `commits check` chega como o
  `argv` que veio depois do `--`, sem ser reparseado.

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

Os comandos `branches`, `worktrees`, `commits check` e `ignore list` estão no
ar, os quatro com `--format`, mais o `licenses` e o `doctor`. Planejados: seleção interativa de quais branches
apagar, `gtr split` para quebrar a árvore suja em vários commits,
`gtr ignore add`, `stats`, `changelog` e arquivo de configuração opcional.

## Licença

MIT — ver [LICENSE](LICENSE). Use, modifique e redistribua, inclusive
comercialmente; basta manter o aviso de copyright.

O binário é estaticamente ligado, então o código das dependências viaja dentro
dele, e as licenças delas exigem acompanhar a redistribuição. Elas estão em
[THIRD-PARTY-LICENSES](THIRD-PARTY-LICENSES) e **embutidas no próprio
binário**, o que mantém de pé a promessa de distribuir copiando um arquivo:

```
$ gtr licenses
  github.com/spf13/cobra                Apache-2.0
  github.com/spf13/pflag                BSD-3-Clause
  github.com/inconshreveable/mousetrap  Apache-2.0    (só no binário Windows)
```

Nenhuma é copyleft e nenhuma impõe condição ao código do `gtr`. Um teste
compara o `go.mod` com o arquivo e falha se entrar dependência cujas licenças
não estejam ali.

## Contribuindo

Antes de abrir PR: `gofmt -l .` vazio, `go vet ./...` limpo e `go test ./...`
verde — os mesmos três portões que o CI aplica.
