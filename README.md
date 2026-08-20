# gitarias

CLI de utilitários git em binário único. O binário chama `gtr`.

O que a ferramenta faz é orquestrar o `git` que já está na sua máquina.
**Nenhum comando sai da máquina sem dizer que sai** — hoje `gtr pr` e
`gtr doctor --online` saem falando com o GitHub através do `gh`, e
`gtr riff` e `gtr fire` saem falando HTTP puro com uma API pública
(`whatthecommit.com`), sem chave nem token. Os quatro anunciam isso no
`--help`. Nem os dois primeiros pedem token: quem fala com o GitHub é o `gh`,
e o `gtr` nunca vê a sua credencial.

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

Confere se o `gtr` funciona aqui, agora.

```
$ gtr doctor
  ok  git          2.43.0
  ok  temporário
  ok  repositório
  ok  árvore       sem operação em curso
  ok  identidade   Luiz Palma <luiz@exemplo.com>
  ok  base         main, declarada pelo remoto
  ok  gh           2.45.0
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--online` | `false` | Acrescenta a checagem de conexão com o GitHub; **faz chamada de rede** |
| `--strict` | `false` | Trata aviso como falha, para quem roda o `doctor` num portão de CI |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |

**Quatro estados, e a diferença entre eles é o que o comando tem de útil:**

| Estado | Significa | Afeta a saída? |
|---|---|---|
| `ok` | está no lugar | não |
| `falta` | o `gtr` não funciona sem isso | **sai 1** |
| `aviso` | quebra um comando, não a ferramenta | só com `--strict` |
| `--` | não se aplica aqui | nunca |

**É o único comando que roda fora de um repositório git**, e ali as checagens
que dependem de um não falham — elas se declaram inaplicáveis:

```
$ cd /tmp && gtr doctor
  ok  git          2.43.0
  --  repositório  o diretório atual não é um repositório git
  --  base         depende de estar num repositório
```

**Base indeterminável é aviso, não falha.** Ela só é necessária para o
`gtr branches`, e mesmo lá há saída pelo `--base`:

```
$ gtr doctor        # num repositório sem main, master nem origin/HEAD
  ok     git          2.43.0
  ok     repositório
  aviso  base         não determinável aqui
                      só o gtr branches precisa dela; informe com --base <branch> ou crie main ou master
```

Sem `--strict` isso sai 0. Com `--strict`, sai 1 — é a flag para quem quer que
o portão de CI reclame de tudo que não está redondo.

**O `gh` é opcional e sempre será.** Ele só interessa aos comandos de PR; o
resto do `gtr` funciona sem ele, então a ausência é aviso e não falha:

```
$ gtr doctor
  ok     git          2.43.0
  ok     repositório
  ok     base         main
  aviso  gh           não encontrado no PATH
                      só é preciso para os comandos de PR; o resto do gtr funciona sem ele. Instale em https://cli.github.com
```

**Com `--online`, pergunta ao GitHub quem você é.** Guitarra desplugada toca
sozinha; plugada precisa do cabo — daí o apelido `--plugged` valer pela flag. É a única checagem que sai
da máquina, e por isso é flag e não padrão — o resto do `doctor` é local e
termina sozinho, o que o torna barato de rodar por curiosidade.

```
$ gtr doctor --online
  ...
  ok     conexão      LHPalma

$ gtr doctor --online        # sem credencial
  falta  conexão      sem credencial para o GitHub
                      entre com gh auth login, ou exporte GH_TOKEN com um token de acesso
```

O que ela afirma é estreito de propósito: que existe credencial e que o
servidor a aceitou. **Não afirma que o token é válido** — um proxy que
reautentica no caminho responde igual, e o `doctor` não afirma o que não mediu.

No `json` o `state` é token — `ok`, `warning`, `failure`, `skipped` —, e no
`csv` é o rótulo de tela.

### `gtr undo`

Recria as branches que o `gtr branches --clean` deletou por último. Apelido:
`gtr rewind`.

```
$ gtr undo
Deletadas em 2026-08-15 15:56 (2):
  feat-a  0504044
  feat-b  6756275

Recriar 2 branches? [y/N] y
  - feat-a recriada
  - feat-b recriada
```

**O git não serve de apoio aqui, e é por isso que o `gtr` mantém diário
próprio.** O reflog de uma branch é apagado junto com ela, e o reflog do `HEAD`
só guarda a ponta se você esteve nela. O `gtr` anota nome e ponta em
`<git-common-dir>/gtr/deleted` no momento em que deleta — e a ponta vem de
`rev-parse`, não da frase que o git imprime.

**Recusa em dois casos, e os separa:**

```
  - ocupada: já existe uma branch com esse nome, e o gtr não sobrescreve
  - podada: o commit não está mais no repositório; o git já podou o que ficou inalcançável
```

O segundo é mais raro do que parece: branch mergeada por ancestralidade tem os
commits alcançáveis a partir da base, e o `gc` nunca os leva. É a squashada e a
rebaseada — as que só o `--force` apaga — que ficam inalcançáveis. **A rede é
mais fina justamente onde a deleção era mais arriscada.**

### `gtr author`

Reescreve a autoria de commits. Apelido: `gtr blame-someone-else`, uma
referência direta ao projeto de mesmo nome — e a piada é literal: o comando
pode atribuir a autoria a qualquer nome e e-mail informados, não só corrigir
a sua própria identidade.

```
$ gtr author --name "Fulano Sênior" --email fulano@empresa.com
Vai reescrever o commit mais recente, hoje em nome de Estagiário <intern@empresa.com>, para Fulano Sênior <fulano@empresa.com>.
Recuperável com: git reset --hard 0b521de
Confirma? [y/N] y
Pronto.
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--name <nome>` | vazio | O nome a atribuir (obrigatório) |
| `--email <e-mail>` | vazio | O e-mail a atribuir (obrigatório) |
| `--base <ref>` | vazio | Reescreve `<ref>..HEAD` em vez de só o commit mais recente; `<ref>` fica de fora |
| `--until <ref>` | vazio | Com `--base`, fecha o intervalo antes do `HEAD`; o que vem depois de `<ref>` é preservado |
| `--commit <sha>` | vazio | Reescreve só esse commit, preservando tudo antes e depois; incompatível com `--base` e `--until` |
| `--reset <sha>` | vazio | Wrapper de `git reset --hard <sha>`; descarta os commits depois de `<sha>` e qualquer mudança não commitada; incompatível com `--name`, `--email`, `--base`, `--until` e `--commit` |

**É uma reescrita de história de verdade.** Sem `--base`, é um
`commit --amend --reset-author` mais direto — só o SHA do topo muda. Com
`--base`, é um `git rebase <base> --exec 'commit --amend --reset-author'` —
toda a cadeia de SHAs dali para a frente muda, e todo clone e PR que já
tinham os commits antigos quebram.

**`--base` some tudo até o `HEAD`; `--until` fecha antes disso, preservando o
resto.** Com `--base main --until <ref>`, os commits depois de `<ref>`
continuam exatamente como estavam — mesmo conteúdo, mesma autoria —, só
reencaixados em cima do trecho reescrito:

```
$ gtr author --name "Fulano Sênior" --email fulano@empresa.com --base main --until e953185
Vai reescrever 2 commits (main..e953185) para Fulano Sênior <fulano@empresa.com>.
Autores atuais no intervalo: Real Person <real@real.com>.
Mais 2 commits depois de e953185 serão preservados, só reencaixados em cima.
Recuperável com: git reset --hard 28d1dc2
Confirma? [y/N] y
Pronto.
```

Por baixo são duas passadas: a primeira reescreve `base..until` com o `HEAD`
destacado nele; a segunda reencaixa em cima (`rebase --onto`) o que ficou
para trás, sem tocar no conteúdo. Testado também a partir de `HEAD` já
destacado (sem branch) e com `--until` igual ao próprio `HEAD`, onde o
reencaixe não tem nada para mover.

**`--commit <sha>` reescreve um commit só, em qualquer lugar do histórico**,
preservando tudo antes e tudo depois — o "cherry-pick de autoria" que o nome
sugere. É açúcar sobre o mesmo mecanismo: por baixo, equivale a
`--base <sha>^ --until <sha>`, mas sem expor a sintaxe de `^` do git na
prévia:

```
$ gtr author --name "Fake Name" --email fake@fake.com --commit f74b936
Vai reescrever o commit f74b936 (hoje em Real Person <real@real.com>) para Fake Name <fake@fake.com>.
Mais 2 commits depois de f74b936 serão preservados, só reencaixados em cima.
Recuperável com: git reset --hard 3ba78c2
Confirma? [y/N] y
Pronto.
```

**Não funciona no commit raiz** — ele não tem pai, e `<raiz>^` não resolve.
O erro que sai é o do próprio git, cru, sem tradução: é a única borda que
`--commit` deixa em aberto.

**A prévia sempre lista quem são os autores atuais do intervalo**, antes de
perguntar. `--base`/`--until` cortam por alcançabilidade, não por autoria —
numa `main` desatualizada, o intervalo pode incluir commits de outras
pessoas, e a lista é o que avisa disso antes de reatribuir tudo.

**Sempre pergunta antes, e sempre imprime como desfazer.** A linha
`Recuperável com: git reset --hard <sha>` sai antes da pergunta, não depois —
`[y/N]`, qualquer coisa que não seja `y`/`yes`/`s`/`sim` cancela, inclusive
Enter vazio. Não tem `--force` para pular a pergunta.

**Autor e committer mudam juntos**, e os dois vão pelo ambiente do processo —
`GIT_AUTHOR_NAME`/`GIT_AUTHOR_EMAIL`/`GIT_COMMITTER_NAME`/
`GIT_COMMITTER_EMAIL` —, nunca por um argumento interpolado dentro do
`--exec` do rebase. O `--exec` roda por um shell; um nome com `$(...)` ou
crase interpolado ali seria injeção de shell pelo próprio dado que o comando
existe para reatribuir. Medido contra um nome assim antes de decidir pela
forma com variável de ambiente: com ela, o git trata o texto como identidade
e nada executa.

**`--reset <sha>` é um wrapper puro**, sem reescrita nenhuma por trás: é
`git reset --hard <sha>` com a mesma prévia e a mesma confirmação do resto do
comando, incompatível com `--name`, `--email`, `--base`, `--until` e
`--commit`. A prévia conta quantos commits deixam de ser alcançáveis a partir
da branch, e avisa separadamente se há mudança não commitada em arquivo
rastreado — o `--hard` descarta as duas coisas, mas só os commits têm uma
linha de recuperação:

```
$ gtr author --reset 91d918f
Vai voltar para 91d918f com git reset --hard, descartando 2 commits que deixam de ser alcançáveis a partir daqui.
Recuperável com: git reset --hard 7434de9
Confirma? [y/N] y
Pronto.
```

Se a árvore de trabalho tiver mudança não commitada em arquivo rastreado, a
prévia ganha uma linha a mais antes da confirmação — essa mudança some sem
deixar rastro nenhum, nem no reflog, e não tem como recuperar:

```
Há mudança não commitada em arquivo rastreado: será descartada sem deixar rastro nenhum, nem no reflog.
```

### `gtr weight`

Mostra o que mais pesa no histórico. Apelido: `gtr roadie`.

```
$ gtr weight
Um clone deste repositório baixa 2.9 MB.

  2.9 MB  1 versão   build.zip  só no histórico
  34 B    2 versões  app.go     na árvore
```

**Apagar um arquivo não o tira do repositório.** O commit em que ele existia
continua no histórico, o objeto segue alcançável, e todo clone continua
baixando aquele blob — inclusive quem entrou anos depois e nunca vai vê-lo. O
`git gc` não resolve: o objeto não é lixo, é história.

A coluna `só no histórico` é o comando inteiro — é onde costuma estar o peso
que ninguém explica.

| Flag | Padrão | Efeito |
|---|---|---|
| `--limit <n>` | `10` | Quantos caminhos mostrar; `0` traz todos |

O tamanho medido é o **em disco**, não o lógico: é o que um clone paga. Na tela
sai legível (`2.9 MB`); no `csv` e no `json`, cru (`3000000`).

**Diagnostica e não conserta.** Expurgar um blob é reescrever o histórico —
todo SHA dali para frente muda, e todo clone e PR existente quebra.

### `gtr churn`

Quais arquivos mais mudam no histórico do `HEAD` atual — não quanto pesam
(isso é o `weight`), quantas vezes foram tocados.

```
$ gtr churn
  3 commits  core.go    na árvore
  1 commit   README.md  na árvore
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--limit <n>` | `10` | Quantos caminhos mostrar; `0` traz todos |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

**Arquivo que muda toda hora concentra risco**, mesmo pesando pouco — é o
oposto do que o `weight` mostra. `README.md` pode pesar quase nada e mudar
duzentas vezes; um binário grande pode nunca mais ser tocado depois do
primeiro commit.

Vem de `git log --name-only`, uma passada só pelo histórico, com
`--no-renames` travado: sem essa flag, um arquivo renomeado conta como um
caminho só ou como dois dependendo do `diff.renames` configurado em cada
máquina. Aqui um renomeio sempre toca os dois caminhos, o antigo e o novo, do
mesmo jeito em qualquer lugar que rodar. Commit de merge não entra na
contagem — é o comportamento padrão do `git log` para `--name-only`, e conta
só quem de fato tocou o arquivo, não quem só reuniu o trabalho de outros.

### `gtr stats`

Quantos commits cada autor tem no histórico do `HEAD` atual.

```
$ gtr stats
Autores (2):
  AUTOR                            COMMITS
  anteninha <anteninha@teste.com>  3
  NataLia <natalia@teste.com>      1
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--author <padrão>` | vazio | Filtra por nome ou e-mail, casando substring; repetível, soma quem casar com qualquer um |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

`--author` é a mesma flag do `git log`, passada direto para o git — casa por
substring, e é repetível: várias ocorrências **somam** quem casar com
qualquer uma delas, não exigem casar com todas.

```
$ gtr stats --author NataLia
Autores (1):
  AUTOR                        COMMITS
  NataLia <natalia@teste.com>  1
```

É leitura local, sem rede — só `git log`. Repositório sem nenhum commit ainda
não é erro: `Nenhum commit encontrado.`

### `gtr profile`

Métricas sobre a **sua própria** identidade de git neste repositório —
diferente do `gtr stats`, que conta todo mundo. Cada métrica é uma flag
própria; hoje só existe uma:

| Flag | Padrão | Efeito |
|---|---|---|
| `--commit-count` | `false` | A métrica: quantos commits seus caem no período. Obrigatória — sem ela, o comando recusa |
| `--since <data>` | hoje | Início do período, `AAAA-MM-DD`. Sem `--until`, vai até hoje |
| `--until <data>` | hoje | Fim do período, `AAAA-MM-DD`. Sem `--since`, começa hoje |

Sem nenhuma das duas datas, o período é só hoje:

```
$ gtr profile --commit-count
1 commit em 2026-08-18.
```

Mesmo valor nas duas conta um dia certo; valores diferentes contam um
intervalo:

```
$ gtr profile --commit-count --since 2026-08-15 --until 2026-08-15
2 commits em 2026-08-15.
$ gtr profile --commit-count --since 2026-08-15 --until 2026-08-16
3 commits entre 2026-08-15 e 2026-08-16.
```

**A identidade é a que já está configurada no repositório** —
`git config user.email`, ou `user.name` se só ele estiver — nunca uma flag
`--author`: é o seu perfil, não o de qualquer um. Sem nenhum dos dois
configurados, o erro manda configurar.

**As duas pontas do dia são explícitas por baixo, nunca a data nua.**
`git log --since=2026-08-15` sozinho não vale meia-noite daquele dia — vale a
hora corrente de agora, naquele dia —, e `--since` igual a `--until` dava
**zero** mesmo com commit dentro do dia. Medido contra o git antes de decidir:
por baixo, `--since` vira `<data> 00:00:00` e `--until` vira
`<data> 23:59:59`, sempre.

É leitura local, sem rede — só `git log` e `git config`.

### `gtr setup`

Diz o que falta e qual comando resolve **na sua máquina**. Apelido:
`gtr luthier`.

```
$ gtr setup
Detectei ubuntu 24.04, com apt-get.

git — não encontrado no PATH:
    sudo apt-get update
    sudo apt-get install -y git
```

Detecta entre `apt-get`, `dnf`, `pacman`, `zypper`, `apk`, `brew`, `winget`,
`choco` e `scoop`, e põe `sudo` só quando o gerenciador precisa e você não é
root. O nome do pacote nem sempre é o do comando — no Arch o `gh` é
`github-cli`, no winget o git é `Git.Git`.

**Imprime e não executa.** Sem gerenciador reconhecido, cai na página oficial
em vez de inventar comando. E só ferramenta ausente ganha linha de instalação:
um git velho demais não se resolve com `apt-get install`, e prometer que sim
engana.

### `gtr pr list`

Lista os pull requests abertos. **Faz chamada de rede** — é o primeiro comando
do `gtr` que sai da máquina, e diz isso no `--help`.

```
$ gtr pr list
  #7  aberto    feat-parser  muda o parser
  #8  rascunho  feat-outra   ainda cozinhando
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--limit <n>` | `30` | Quantos pull requests trazer |

Quem fala com o GitHub é o `gh`, que resolve autenticação, host de Enterprise e
qual repositório é o do diretório atual — **o `gtr` nunca vê o seu token**. Sem
o `gh`, o erro manda rodar `gtr setup`; com ele presente e recusado, sobe a
mensagem do próprio `gh`.

Há um prazo de 30 segundos. Toda outra operação do `gtr` é local e termina
sozinha; um servidor calado não termina.

### `gtr riff`

Imprime uma mensagem de commit aleatória do `whatthecommit.com`. Apelido:
`gtr whatthecommit`. **Faz chamada de rede** — é o único comando do `gtr` que
fala com a internet sem passar pelo `gh`: HTTP puro contra uma API pública,
sem chave e sem autenticação.

Numa máquina com saída de rede liberada para esse domínio, a saída é a
mensagem, sozinha, numa linha — mesma fonte testada ao vivo nesta sessão
(`No cap`, `Blaming regex.`, `Landed.`, entre outras). Neste ambiente de
desenvolvimento o proxy de saída bloqueia o domínio, e é isso que a chamada
real contra o binário mostra:

```
$ gtr riff
erro: Get "https://whatthecommit.com/index.txt": Forbidden
```

Serve para alimentar outro comando, como o próprio `git commit`:

```
$ git commit -m "$(gtr riff)"
```

É a mesma fonte que o `gtr fire` usa para a mensagem do commit — ver a seção
a seguir, inclusive o que acontece quando ela falha.

### `gtr fire`

Comita tudo que está sujo — rastreado ou não, staged ou não — e empurra para
uma branch nova no remoto, sem perguntar nada. É o botão de pânico, inspirado
no projeto de piada `git-fire`. Apelido: `gtr jam` — "estar num jam" é estar
numa enrascada, e jam também é a sessão de improviso. A branch local atual
também fica com o commit; só o destino remoto é novo, para não mexer no que
já estava rastreado lá.

```
$ gtr fire
Salvo em fire/0484d95: "🔥 fire"
Recuperável com: git reset --hard a9dff83
```

A branch nova nasce do SHA do próprio commit (`fire/<sha>`), não de um
carimbo de hora — não depende do relógio e não colide entre duas chamadas
rápidas que produzam commits diferentes. Sem nada sujo para salvar, o comando
não toca em nada:

```
$ gtr fire
Nada sujo para salvar.
```

**Faz chamada de rede** para a mensagem do commit, pela mesma fonte do
`gtr riff` — se ela falhar, ou este ambiente não tiver saída para o domínio,
a mensagem cai para uma fixa (`🔥 fire`) e o comando segue em frente: pânico
não espera a internet, e é por isso que a chamada por trás nunca pode falhar
o `fire` inteiro.

**Sem confirmação, de propósito** — é a única exceção nesse sentido no `gtr`.
O resto da ferramenta sempre pergunta antes de mexer no remoto (**ADR-008**);
aqui, a piada inteira é não perguntar. A linha `Recuperável com:` continua
saindo, porque nada que muda estado local deixa de dizer como desfazer —
só que aqui ninguém espera a confirmação para agir.

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
| `--expand-dir <caminho>` | vazio | Expande só o(s) diretório(s) informado(s); repetível, incompatível com `--expand` |
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
`--expand` abre todos de uma vez; para abrir só um, sem engolir os outros:

```
$ gtr ignore list --expand-dir node_modules/
Ignorados (5):
  CAMINHO                ORIGEM               PADRÃO
  app.log                .gitignore:2         *.log
  local-only/            .git/info/exclude:1  local-only/
  node_modules/react.js  .gitignore:1         node_modules/
  node_modules/vue.js    .gitignore:1         node_modules/
  relatório 2026.csv     .gitignore:4         relat*.csv

Diretório ignorado conta como uma linha só. Use --expand para listar arquivo a arquivo.
```

A flag é repetível (`--expand-dir a/ --expand-dir b/`) e só aceita o caminho
**exatamente como o `gtr ignore list` colapsou**; pedir um diretório que não
está colapsado é erro, nomeando os que estão. Com o script de completion do
Cobra carregado no shell (`source <(gtr completion bash)` ou equivalente), o
tab sugere só esses diretórios.

**Arquivo já rastreado não aparece**, mesmo casando com uma regra — o git
continua rastreando quem já estava dentro. É a confusão número um do
`.gitignore`, e vale saber ao procurar algo que "deveria ter sumido".

### `gtr changelog`

Gera o `CHANGELOG.md` a partir do histórico do `HEAD` atual, classificando
cada commit pelo [Conventional Commits](https://www.conventionalcommits.org/)
e agrupando por tipo.

```
$ gtr changelog
## [Unreleased]

### Features

- gtr changelog agrupa por tipo do Conventional Commits (`a1b2c3d`)

### Bug Fixes

- corrige o parser de escopo (`9f8e7d6`)
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` — o `gtr` não escreve o `CHANGELOG.md` sozinho |
| `--separator <s>` | `,` | Só com `--format csv`. Aceita `,` `;` `\|` e `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

`--format text` (o padrão) é o único que produz o Markdown de verdade; para
gravar o arquivo, `gtr changelog --output CHANGELOG.md`. Os outros três
formatos servem para consumo por script: uma linha por commit, com as
colunas `hash`, `tipo`, `escopo`, `breaking` e `assunto` — o `tipo` sai como
o token do Conventional Commits (`feat`, `fix`...), nunca traduzido.

**Sem tag nenhuma no repositório, tudo cai em `[Unreleased]`.** O `gtr` ainda
não sabe fatiar por versão — quando o projeto começar a taguear releases,
isso é trabalho futuro, não uma promessa desta versão.

**Onze tipos reconhecidos:** `feat`, `fix`, `perf`, `refactor`, `docs`,
`test`, `build`, `ci`, `chore`, `style` e `revert`, cada um com sua seção,
nessa ordem, e só aparecem as que têm commit. **Todo tipo aparece por
padrão** — não há como esconder `chore`/`test`/`ci`/`style`; isso é trabalho
do `.gtr.yaml` (ADR-004), que ainda não existe. Commit que não segue a
convenção cai em `Miscellaneous`, assunto cru e tudo, e nunca é descartado.

**`!` antes dos dois-pontos marca mudança incompatível** (`feat!: ...`),
mostrado como `⚠ BREAKING:` na frente do assunto. O rodapé `BREAKING
CHANGE:` no corpo do commit — a outra forma que a spec permite — **não** é
lido: exigiria o corpo inteiro do commit, não só o assunto, e não há nenhum
commit real no histórico do `gitarias` para medir o parser contra ele.

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
- **Nunca vê a sua credencial do GitHub.** Os comandos que falam com o GitHub
  chamam o `gh`, que já resolve autenticação, host de Enterprise e qual
  repositório é o do diretório atual. Nenhum token passa pelo `gtr`, nem por
  variável de ambiente, nem por `argv`.
- **Nunca sai da máquina sem dizer.** `gtr pr`, `gtr doctor --online`,
  `gtr riff` e `gtr fire` fazem requisição de rede, e os quatro declaram isso
  no `--help`. Os dois primeiros falam com o GitHub através do `gh`; os dois
  últimos falam HTTP puro com uma API pública, sem chave e sem token. O
  `doctor` sem a flag, e todo o resto, é local — é isso que o torna barato de
  rodar por curiosidade.
- **Nunca instala nada.** O `gtr setup` imprime o comando de instalação da sua
  máquina; quem executa é você. Um binário que pede senha de root para agir
  sozinho é um hábito que não vale ensinar.

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

No ar: `branches` (com `--tree`), `worktrees`, `commits check`, `ignore list`,
`licenses`, `doctor`, `undo`, `author`, `weight`, `churn`, `setup`, `pr list`
e `stats`. Todos aceitam `--format`, menos o `licenses`, que imprime texto de
licença, e o `undo` e o `author`, que são interativos.

Planejados: seleção interativa de quais branches apagar, `gtr split` para
quebrar a árvore suja em vários commits, `gtr ignore add`, `stats`, `changelog`
e arquivo de configuração opcional.

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
