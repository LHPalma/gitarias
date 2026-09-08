---
titulo: SRS — worktrees
data: 2026-08-05
status: entregue
comando: gtr worktrees • gtr worktrees remove • gtr worktrees release
pacotes:
  - cmd/worktrees.go
  - internal/worktree
  - internal/git
commits:
  - 05addb1
  - 5997c8a
  - 0b2db6f
  - 75dbf49
  - 5a41f68
  - f150939
  - e8cc0a0
  - 7b0ccf1
  - a73d0f0
  - c5c22b8
fonte_externa: nenhuma
---

# SRS — worktrees

- **Data:** 2026-08-05
- **Feature:** `cmd/worktrees` + `internal/worktree`
- **Status:** **entregue** — a listagem, o `remove` e o `release`
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar o comando `gtr worktrees`, que lista os working trees do repositório com branch, estado e marcação do atual.

### 1.2 Escopo

- Listagem de todos os working trees, em colunas alinhadas.
- Marcação do working tree de onde o comando foi invocado.
- Reporte de estado: `trancado` e `podável`, com o motivo quando o git informa um.
- `remove` — apaga um working tree **dizendo antes o que o git apagaria em silêncio**.
- `release` — solta a branch presa, pelo único caminho comprovadamente não destrutivo.

A listagem **não tem flags próprias** além das de saída compartilhadas por todo comando; o `remove` e o `release` não têm flag nenhuma, nem `--force`.

As decisões por trás dos dois subcomandos — inclusive por que embrulhar o `git worktree remove` só fazia sentido com guarda mais forte que a dele — estão na **[ADR-007](../adr/007-soltar-branch-presa.md)**.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Working tree** | Diretório de trabalho ligado a um repositório. Um repo pode ter vários, cada um com um checkout diferente. |
| **Bare** | Repositório sem diretório de trabalho — só o banco de objetos. |
| **Trancado** (`locked`) | Working tree marcado para não ser podado automaticamente. Típico de mídia removível. |
| **Podável** (`prunable`) | Working tree cujo diretório sumiu — o git pode limpar a referência. |
| **`--porcelain` (flag)** | **Significa o oposto da categoria de mesmo nome**, e é armadilha conhecida do vocabulário do git. Como *categoria*, porcelain é o que o humano lê; como *flag*, pede a saída estável feita para script. O `gtr` consome as duas coisas certas: `for-each-ref` (comando plumbing) e `worktree list --porcelain` (flag de saída estável). |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/git/                execução de git — compartilhado com o domínio de branches

internal/worktree/           domínio — não imprime nada
├── worktree.go              Worktree
├── unquote.go               desfaz a citação em C do motivo — §6.2
└── repo.go                  Ensure, List, parse, IgnoredFiles, Remove, Release, Dirty

cmd/worktrees.go             front de texto: a listagem, o remove e o release
cmd/worktrees_table.go       a tabela, nos quatro formatos
```

**O `internal/git` nasceu por causa desta feature.** O plano original previa só `internal/branch`. Um working tree não é uma branch, e fazer o pacote `worktree` importar `branch` só para executar git seria mentira de dependência — o mesmo apareceria de novo no `stats` e no `changelog`.

### 2.2 Fluxo

1. Verifica que o diretório atual é um repositório git.
2. Roda `git worktree list --porcelain` e faz o parse dos blocos.
3. Descobre o worktree atual com `rev-parse --show-toplevel` e marca a linha correspondente.
4. Imprime em colunas alinhadas: marcador, caminho, branch (ou estado do checkout) e estado do worktree.

---

## 3. Requisitos funcionais

### RF-01 — Listar working trees

`gtr worktrees` imprime todos os working trees do repositório, um por linha, em colunas alinhadas: caminho, branch em checkout e estado.

O worktree de onde o comando foi invocado recebe `*`. **A marcação acompanha o usuário entre diretórios** porque vem da comparação com `rev-parse --show-toplevel`, não de posição na lista.

### RF-02 — Reportar o estado de cada working tree

No lugar da branch, quando não houver uma: `(bare)` ou `(HEAD destacado em <sha7>)`.

Na coluna de estado: `trancado` e/ou `podável`, cada um seguido do motivo quando o git informa um — `trancado: hd externo desconectado`.

**O booleano e o motivo são campos separados** porque `git worktree lock` sem `--reason` emite a linha `locked` pelada, e aí string vazia não poderia significar "não trancado".

### RF-03 — Remover um working tree dizendo o que se perde

`gtr worktrees remove <caminho>` lista, **antes de perguntar**, os arquivos ignorados que existem naquele working tree — e só então pede confirmação.

É a razão de o comando existir: `git worktree remove` sozinho recusa apagar arquivo versionado sujo ou não rastreado, mas **arquivo ignorado não é nenhum dos dois**. Um `.env` ou um `node_modules/` esquecido ali some junto, com exit 0, e **sem recuperação possível** — arquivo ignorado nunca esteve no git.

A listagem sai de `git -C <caminho> ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z`, rodada **contra o working tree alvo**, independente de onde quem chamou está.

### RF-04 — Soltar a branch sem mexer em mais nada

`gtr worktrees release <branch>` roda `git -C <caminho> checkout --detach` no working tree que prende a branch, descobrindo o caminho a partir do nome — o `List` já devolve isso.

Antes de perguntar, diz onde a branch está. Quando há trabalho não commitado lá, **avisa**: ele sobrevive intacto, mas passa a viver num `HEAD` destacado, e quem não conhece o estado se assusta.

### RF-05 — Nenhum dos dois força coisa alguma

Nem o `remove` nem o `release` têm `--force`. O que o git recusar continua recusado, com o erro dele propagado — no `release`, é o que acontece com uma rebase pela metade, e é a defesa que dispensa escrever guarda própria.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O `remove` só se justifica por dizer o que o git não diz.** Embrulhar o `git worktree remove` sem acrescentar a listagem de ignorados seria a mesma armadilha com outro nome. A guarda é a informação, não a flag. |
| **RN-02** | **A listagem de ignorados roda contra o working tree alvo, não contra o diretório de quem chamou.** `-C <caminho>` em toda chamada; sem isso o comando mostraria os ignorados do lugar errado, que é pior do que não mostrar nenhum. |
| **RN-03** | **Nenhum dos dois subcomandos tem `--force`.** O que o git recusa continua recusado. O `release` não precisa de guarda própria justamente porque o git se defende sozinho — com rebase pela metade, o `detach` recusa e o erro é propagado. |
| **RN-04** | **O `release` não é chamado por ninguém automaticamente.** Alterar o estado de outro diretório nunca acontece como efeito colateral de limpar branch: quem executa é quem digitou o comando para isso — **ADR-007**. |

### 4.1 Regras herdadas

Estas não nascem aqui, e a numeração delas é a do documento de origem:

- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). Working trees saem de `git worktree list --porcelain`, não da listagem humana; formato contratual não é localizado, então `LANG` em qualquer idioma não quebra o parse. **Ressalva não verificada:** não se sabe se o motivo do `prunable` dentro do `--porcelain` passa pelo gettext do git. Se passar, é a única string localizada que entra no parse.
- **Nenhuma linha de saída termina em espaço** — `RN-10` da [SRS — branches](branches.md). A coluna de estado só é emitida quando existe conteúdo. Ver §6.1: aqui isso foi um defeito encontrado, não uma precaução teórica.
- **Nada no domínio escreve na tela** — `RN-11` da [SRS — branches](branches.md). O domínio devolve `[]Worktree`; `(bare)`, `trancado` e `podável` são escolhidos na apresentação a partir de booleanos.

---

## 5. Interface

```console
$ gtr worktrees
Working trees (4):
* ~/rmain      main
  ~/rsimples   simples
  ~/rsumido    sumido    podável: gitdir file points to non-existent location
  ~/rtrancado  trancado  trancado: hd externo desconectado
```

```console
$ gtr worktrees remove ../projeto-fix
~/projeto-fix tem 3 arquivos ignorados que serão perdidos:
  .env
  node_modules/
  dist/

Remover ~/projeto-fix? [y/N]
```

```console
$ gtr worktrees release presa
presa está em ~/projeto-fix. git checkout --detach vai soltar a branch, sem tocar em mais nada ali.
~/projeto-fix tem trabalho não commitado: ele sobrevive, mas passa a viver num HEAD destacado.

Soltar presa? [y/N]
```

---

## 6. Testes

### 6.1 O motivo do lock vem citado em C, e o valor cru ia para a tela

O `worktree list --porcelain` **cita e escapa à moda C** o motivo do lock quando ele tem não-ASCII, aspas, barra invertida ou tab. O valor cru chegava à saída assim:

```text
trancado: "revis\303\243o em andamento"
```

E o `--format` piorava, em vez de escapar de novo por cima: o csv citava o que o git já tinha citado, e o campo chegava com três níveis de aspas.

Daí o `unquote.go`: se o valor abre com aspas, ele passa por `strconv.Unquote`, e **volta como veio se a decodificação falhar** — um motivo malformado não pode derrubar a listagem inteira.

**A regra foi levantada rodando contra o git, não de memória**, e a medição também disse o que *não* é citado: o **caminho** do worktree sai cru sempre. Escapar os dois seria corrigir um campo que nunca esteve errado.

### 6.2 O defeito do espaço no fim da linha

Com `tabwriter`, uma linha cuja **última célula está vazia** sai com espaços no fim — a coluna anterior é preenchida até a largura da mais larga. Invisível na tela, visível em `grep` e em comparação de saída.

Encontrado com `gtr worktrees | grep ' $'`, e corrigido emitindo a coluna de estado só quando há conteúdo. É a aplicação local da `RN-10` da [SRS — branches](branches.md).

### 6.3 Cenários executados manualmente

| # | Cenário | Resultado |
|---|---|---|
| 1 | Principal, linked, detached e trancado | Quatro linhas corretas, colunas alinhadas |
| 2 | Rodado de dentro de outro worktree | O `*` mudou de linha — acompanha o diretório, não a ordem |
| 3 | Repo sem worktree extra | Uma linha, a do próprio repo |
| 4 | `worktree lock --reason` e worktree com diretório removido | Motivo de `trancado` e de `podável` exibidos — `RF-02` |
| 5 | `gtr worktrees \| grep ' $'` | Nenhuma linha com espaço no fim — §6.2 |
| 6 | Motivo de lock com acento e com aspas | Sai decodificado na tela e citado uma vez só no csv — §6.1 |

### 6.4 Cobertura automatizada

**100% de statements em `internal/worktree`.** Cobre todas as formas de bloco do `--porcelain`: simples, detached, bare, trancado com e sem motivo, podável, entrada vazia, e bloco final sem linha em branco depois.

**No `cmd`, o `remove` e o `release` não estão a 100%:** `findWorktree` está em 87,5% e `reportWhatWouldBeLost` em 88,9%. São os dois pontos deste comando abaixo da barra do projeto, e ficam registrados em vez de arredondados.

**Uma mutação sobreviveu à primeira bateria**, e vale registrar como exemplo do método funcionando: remover a guarda `currentPath != ""` não quebrou nenhum teste. Nenhum caso combinava `rev-parse` vazio com worktree de caminho vazio — par alcançável se o git emitir uma linha `worktree ` malformada. Fechado com `TestListDoesNotMarkEmptyPathAsCurrent`.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `05addb1` | `internal/worktree` e `cmd/worktrees.go`. `feat` porque muda o que o usuário vê — `RF-01` |
| `5997c8a` | O motivo de `locked`/`prunable` deixa de ser descartado. O parse já o separava para ler a palavra-chave — passou a guardar as duas metades — `RF-02` |
| `0b2db6f` | 6 funções, 17 subtestes. 100% do domínio |
| `75dbf49` | O `tabwriter` passa a vir do helper compartilhado `columns` |
| `5a41f68` | Testes do comando de ponta a ponta, incluindo a coluna de estado |

---

## 8. Follow-ups conhecidos

- **`findWorktree` e `reportWhatWouldBeLost` abaixo de 100%** — §6.4.
- **O `remove` não distingue ignorado de ignorado que importa.** Ele lista `.env` e `dist/` com o mesmo peso, e quem lê decide. Classificar exigiria heurística sobre nome de arquivo, que é o tipo de adivinhação que o projeto evita.
- **Verificar se o motivo do `prunable` é localizado** pelo gettext do git — única string do parse potencialmente sujeita a `LANG`.
