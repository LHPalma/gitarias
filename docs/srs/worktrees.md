---
titulo: SRS — worktrees
data: 2026-08-05
status: entregue
comando: gtr worktrees
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
fonte_externa: nenhuma
---

# SRS — worktrees

- **Data:** 2026-08-05
- **Feature:** `cmd/worktrees` + `internal/worktree`
- **Status:** **entregue** — na `main`, 100% de cobertura no domínio
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar o comando `gtr worktrees`, que lista os working trees do repositório com branch, estado e marcação do atual.

### 1.2 Escopo

- Listagem de todos os working trees, em colunas alinhadas.
- Marcação do working tree de onde o comando foi invocado.
- Reporte de estado: `trancado` e `podável`, com o motivo quando o git informa um.

A listagem **não tem flags próprias** além das de saída compartilhadas por todo comando.

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
└── repo.go                  Repo: Ensure, List, parse

cmd/worktrees.go             front de texto
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

---

## 4. Regras de negócio

Este documento não estabelece regra própria. Aplica três já estabelecidas em outro lugar, e a numeração delas é a do documento de origem:

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

---

## 6. Testes

### 6.1 O defeito do espaço no fim da linha

Com `tabwriter`, uma linha cuja **última célula está vazia** sai com espaços no fim — a coluna anterior é preenchida até a largura da mais larga. Invisível na tela, visível em `grep` e em comparação de saída.

Encontrado com `gtr worktrees | grep ' $'`, e corrigido emitindo a coluna de estado só quando há conteúdo. É a aplicação local da `RN-10` da [SRS — branches](branches.md).

### 6.2 Cenários executados manualmente

| # | Cenário | Resultado |
|---|---|---|
| 1 | Principal, linked, detached e trancado | Quatro linhas corretas, colunas alinhadas |
| 2 | Rodado de dentro de outro worktree | O `*` mudou de linha — acompanha o diretório, não a ordem |
| 3 | Repo sem worktree extra | Uma linha, a do próprio repo |
| 4 | `worktree lock --reason` e worktree com diretório removido | Motivo de `trancado` e de `podável` exibidos — `RF-02` |
| 5 | `gtr worktrees \| grep ' $'` | Nenhuma linha com espaço no fim — §6.1 |

### 6.3 Cobertura automatizada

**100% em `internal/worktree`** — 6 funções, 17 subtestes. Cobre todas as formas de bloco do `--porcelain`: simples, detached, bare, trancado com e sem motivo, podável, entrada vazia, e bloco final sem linha em branco depois.

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

- **`gtr worktrees remove` e `gtr worktrees release` não estão especificados aqui.** Os dois são subcomandos entregues depois, sob a **ADR-007**, e este documento cobre só a listagem. Merecem especificação própria.
- **Verificar se o motivo do `prunable` é localizado** pelo gettext do git — única string do parse potencialmente sujeita a `LANG`.
