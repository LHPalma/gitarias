---
titulo: SRS — commits bisect
data: 2026-08-29
status: entregue
comando: gtr commits bisect
pacotes:
  - cmd/commits.go
  - cmd/bisected_table.go
  - cmd/bisected_record.go
  - cmd/bisected_document.go
  - internal/commits
commits:
  - e07d9f1
  - 0757f2d
  - 9372be5
fonte_externa: nenhuma
---

# SRS — commits bisect

- **Data:** 2026-08-29
- **Feature:** `cmd/commits.go` (subcomando `bisect`) + `internal/commits` (`Bisect`)
- **Status:** **entregue** — commits `e07d9f1` (domínio), `0757f2d` (cmd), docs `9372be5`
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar `gtr commits bisect`, que acha o **primeiro** commit de um intervalo que não se sustenta sozinho, testando log₂(N) commits em vez de todos os N. É a mesma pergunta que motiva `git bisect run`, respondida sem tocar `HEAD`, índice ou árvore de trabalho.

O [`commits check`](quebrar-diff-em-commits.md) já resolve "quais dos N commits não se sustentam sozinhos", rodando um comando em cada um deles, isolado por extração. O `bisect` é a mesma pergunta, restrita a "qual foi o **primeiro**" — e por isso pode se dar ao luxo de testar só uma fração deles.

### 1.2 Escopo

- Busca binária sobre `base..HEAD`, reaproveitando a mesma extração isolada do `commits check` (`git archive` por padrão, `git worktree add` com `--worktree`).
- Reporta só os commits efetivamente testados, marcando qual deles é o culpado.
- Mesmas flags de saída do `commits check`: `--format`, `--output`, `--separator`, `--no-header`, `--verbose`, `--worktree`.

**Fora de escopo:**

- **`--until`** — o intervalo sempre vai até `HEAD`; não há como restringir o topo, ao contrário do [`gtr overdub`](overdub.md). Não avaliado.
- **Múltiplas transições de verde para vermelho no mesmo intervalo** — a busca binária assume uma só. Com mais de uma, o resultado ainda sai (a busca sempre converge para algum índice), mas não há garantia de que seja informativo. Não detectado nem sinalizado.
- **Retomar uma busca entre invocações** — cada chamada é isolada, sem estado entre uma e outra, ao contrário do `git bisect` de verdade, que guarda estado em `.git/BISECT_*`.

**Esta feature não é destrutiva.** Mesma extração isolada do `commits check`: nada é escrito no repositório, `HEAD` fica onde estava. Por construção, fica **fora da ADR-008**.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Commit testado** | Commit que teve o comando de verificação efetivamente rodado — subconjunto do intervalo inteiro, tipicamente log₂(N) deles. |
| **Culpado** | O primeiro commit testado que falhou — o achado da busca. Ausente quando nenhum testado falhou. |
| **Transição** | O ponto no intervalo, andando do commit mais antigo para o mais novo, onde o resultado do comando de verificação muda de verde para vermelho. A busca binária assume exatamente uma. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/commits/
├── repo.go              Bisect — reaproveita o check() privado que Check já usava
├── commit.go            Commit — inalterado
└── extractor.go         Extractor, ArchiveExtractor, WorktreeExtractor — reusados

internal/ui/             DescribeOutcome, HighlightOutcome — reaproveitados de Check

cmd/commits.go            newCommitsBisectCommand, runCommitsBisect, splitAtDash (reusado)
cmd/bisected_table.go     front de texto — header/rows/document/text
cmd/bisected_record.go    o registro com as tags json
cmd/bisected_document.go  o envelope do json
```

**Nenhuma pressão nova sobre `git.Runner` ou `exec.Runner`.** O `Bisect` reaproveita literalmente o mesmo `check()` privado que o `Check` já usava para extrair e rodar — a única diferença é a ordem e a quantidade de chamadas: busca binária em vez de varredura linear. Zero portas novas, zero capacidade nova nos runners.

### 2.2 Fluxo

1. Verifica que o diretório atual é um repositório git.
2. Resolve o intervalo `base..HEAD` via `Range`, sem mudança.
3. Busca binária: `low, high := 0, len(list)-1`; enquanto `low <= high`, testa o commit do meio.
   - Passou: o culpado, se existir, está depois — `low = mid + 1`.
   - Falhou: guarda como candidato, e o verdadeiro, se houver um antes, está antes ou é ele mesmo — `high = mid - 1`.
4. Ao final, devolve os commits testados — não o intervalo inteiro — e o culpado, se algum teste falhou.

---

## 3. Requisitos funcionais

### RF-01 — Achar o primeiro commit ruim sem testar todos

`gtr commits bisect <base> -- <comando>` acha o primeiro commit de `base..HEAD` para o qual `<comando>` sai diferente de zero, testando `⌈log₂(N)⌉` commits, não os N do intervalo.

### RF-02 — Reportar só o que foi testado

A saída mostra apenas os commits efetivamente testados, não o intervalo inteiro — com uma coluna, ou campo, `culpado` marcando qual deles é o achado.

### RF-03 — Intervalo vazio

Sem nenhum commit entre `base` e `HEAD`, o comando informa isso e sai com 0, sem testar nada.

### RF-04 — Nenhum commit falha

Se todos os testados passarem, a saída diz explicitamente que ninguém falhou, e o código de saída é 0.

### RF-05 — `--worktree`

Troca a extração de `git archive` para `git worktree add --detach`, para comando que precisa do `.git` — mesma flag e mesmo efeito do `commits check`.

### RF-06 — Formatos de saída

`--format text|csv|tsv|json`, `--output`, `--separator`, `--no-header` — mesma família de flags do `commits check`, mesmo comportamento.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Nada é escrito no repositório.** A árvore de cada commit testado sai por `git archive` — ou `git worktree add` com `--worktree` — para um diretório temporário, apagado ao fim. `HEAD`, índice e árvore de trabalho não são tocados. |
| **RN-02** | **A mesma suposição do `git bisect`, e não mais que ela.** O intervalo tem no máximo uma transição de verde para vermelho, andando do commit mais antigo para o mais novo. Sem essa suposição, a busca binária ainda termina, mas não converge para nada que se possa afirmar. |
| **RN-03** | **Código de saída.** 1 se um culpado for encontrado, 0 se todos os testados passarem. |

### 4.1 Regras herdadas

- **Nada é escrito no repositório** — `RN-02` da [SRS — quebrar um diff em commits](quebrar-diff-em-commits.md), da qual a `RN-01` acima é a aplicação literal: é o mesmo `check()` privado.
- **Comandos rodam sem shell** — `RN-06` da [SRS — branches](branches.md). O comando de verificação chega como o `argv` que veio depois do `--`, sem ser reparseado.
- **Relatório no `stdout`, erro no `stderr`, e nenhuma linha terminando em espaço** — `RN-09` e `RN-10` da [SRS — branches](branches.md).

---

## 5. Interface

```console
$ gtr commits bisect main -- go test ./...
Bisectando 47 commits possíveis: 6 testados.

  verde     a1b2c3d  feat: add the third retry
  verde     4d5e6f7  feat: add the fourth retry
  VERMELHO  8a9b0c1  refactor: extract the retry loop
      --- FAIL: TestRetry (0.00s)
  verde     2b3c4d5  feat: add the fifth retry
  VERMELHO  6e7f8a9  fix: tighten the retry window
  VERMELHO  0c1d2e3  feat: add the sixth retry

Primeiro commit ruim: 8a9b0c1  refactor: extract the retry loop
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--verbose` | `false` | Mostra também a saída dos commits testados que passaram |
| `--worktree` | `false` | Extrai com `git worktree` em vez de `git archive` |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |
| `--separator <s>` | `,` | Só com `--format csv` |
| `--no-header` | `false` | Só com `csv` ou `tsv` |

---

## 6. Testes

100% de cobertura em `internal/commits` e no que `cmd/commits.go` e `cmd/bisected_*.go` acrescentaram.

### 6.1 Domínio — `internal/commits/repo_test.go`

| Cenário | Teste |
|---|---|
| Acha o primeiro ruim testando poucos, não todos (7 commits, quebra no índice 4: testa 3, não 7) | `TestBisectFindsTheFirstFailingCommitWithoutTestingEveryOne` |
| Ninguém falha | `TestBisectWithNothingFailing` |
| Todos falham — o culpado é o mais antigo | `TestBisectWithEverythingFailingPicksTheOldest` |
| Um commit só, passando e falhando | `TestBisectWithOneCommit` |
| Lista vazia | `TestBisectWithoutCommits` |
| Roda dentro da árvore extraída e limpa o temporário | `TestBisectRunsInsideTheExtractedTreeAndCleansUp` |
| Propaga falha de extração | `TestBisectPropagatesTheExtractionFailure` |
| Propaga comando que nem começa | `TestBisectPropagatesTheCommandThatCannotStart` |
| Para quando o contexto é cancelado no meio | `TestBisectStopsWhenTheContextIsCancelledMidway` |
| Propaga workspace que não consegue criar (`TMPDIR` inválido) | `TestBisectPropagatesTheWorkspaceItCannotCreate` |

### 6.2 Apresentação — `cmd/commits_test.go`

| Cenário | Teste |
|---|---|
| Acha o culpado com poucas extrações (3 de 7) | `TestCommitsBisectFindsTheFirstFailingCommitWithFewCalls` |
| Ninguém falha | `TestCommitsBisectWithNothingFailing` |
| Intervalo vazio | `TestCommitsBisectWithNothingInTheRange` |
| Recusa sem `--` | `TestCommitsBisectRefusesWithoutTheDash` |
| `--verbose` mostra saída de quem passou | `TestCommitsBisectHidesTheOutputOfWhatPassedUnlessVerbose` |
| `--worktree` contra o `archive` padrão | `TestCommitsBisectUsesArchiveByDefaultAndWorktreeOnRequest` |
| JSON com e sem culpado | `TestCommitsBisectJSON`, `TestCommitsBisectJSONWithoutACulprit` |
| CSV marca só o culpado | `TestCommitsBisectCSVMarksTheCulprit` |
| Fora de repositório | `TestCommitsBisectOutsideRepository` |
| Falha de escrita na tabela, linha a linha | `TestBisectedTablePropagatesTheWriteFailure`, `TestBisectedTablePropagatesTheFailureOfEveryLine` |

---

## 7. Rastreabilidade

| RF/RN | Commit | Teste |
|---|---|---|
| `RF-01`, `RN-02` | `e07d9f1` | `TestBisectFindsTheFirstFailingCommitWithoutTestingEveryOne` |
| `RF-02` | `0757f2d` | `TestCommitsBisectCSVMarksTheCulprit` |
| `RF-03` | `0757f2d` | `TestCommitsBisectWithNothingInTheRange` |
| `RF-04` | `e07d9f1` | `TestBisectWithNothingFailing` |
| `RF-05` | `0757f2d` | `TestCommitsBisectUsesArchiveByDefaultAndWorktreeOnRequest` |
| `RF-06` | `0757f2d` | `TestCommitsBisectJSON`, `TestCommitsBisectCSVMarksTheCulprit`, `TestBisectedTablePropagatesTheWriteFailure` |
| `RN-01` | `e07d9f1` | `TestBisectRunsInsideTheExtractedTreeAndCleansUp` |

---

## 8. Follow-ups conhecidos

- **`--until` não implementado** — o intervalo sempre vai até `HEAD`. É pedido natural depois do [`gtr overdub`](overdub.md), que tem a flag; não avaliado aqui.
- **Múltiplas transições não detectadas** — se o histórico tiver mais de uma transição de verde para vermelho, a busca ainda termina, mas o resultado não vem com nenhum aviso de que a suposição pode não valer.
- **Sem `--online` ou variante que consulte CI de verdade** — o comando de verificação é sempre local; achar o commit que quebrou uma checagem do GitHub Actions exigiria saber reproduzir aquela checagem localmente, fora do escopo desta feature.
