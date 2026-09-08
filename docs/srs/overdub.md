---
titulo: SRS — overdub
data: 2026-08-29
status: entregue
comando: gtr overdub
pacotes:
  - cmd/overdub.go
  - internal/overdub
  - internal/commits
  - internal/exec
commits:
  - 5eab3f3
  - a8bd32c
  - 9ae7bd8
  - 321b6cb
  - 58b2885
  - ff22831
  - 161d9c9
  - 00af8ab
fonte_externa: nenhuma
---

# SRS — overdub

- **Data:** 2026-08-29
- **Feature:** `cmd/overdub.go` + `internal/overdub`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma — só o `git` local, e o próprio `gtr`, invocado de novo via `PATH`

---

## 1. Introdução

### 1.1 Propósito

Especificar `gtr overdub`, que conserta **um** commit no lugar — rodando um comando arbitrário só na árvore dele — e recoloca o resto do histórico por cima, hash em cascata. É o par do [`gtr commits bisect`](commits-bisect.md): um acha o commit ruim, o outro remenda ali mesmo, sem deixar um commit de correção solto no topo.

O problema que resolve: corrigir um bug introduzido num commit do meio da história — não o `HEAD` — hoje exige ou um `git rebase -i` manual (decorado: `pick`→`edit`, rodar o conserto, `add`, `commit --amend`, `continue`), ou aceitar um commit de correção solto no topo, que deixa aquele trecho da história permanentemente quebrado para quem revisar ou reverter só até ali.

### 1.2 Escopo

- Consertar exatamente um commit (`sha`), rodando um comando arbitrário na **árvore de trabalho real** — ao contrário do `commits check`/`bisect`, que extraem para um diretório descartável, porque aqui o resultado precisa ser commitado de volta.
- Recolocar `sha^..until` por cima, hash em cascata (`--until`, vazio significa `HEAD`).
- `--verify`: depois do conserto, roda o mesmo mecanismo do `commits check` no intervalo reescrito, reportando se algum commit ainda não se sustenta.
- `--yes`: pula a confirmação interativa, para uso por script ou agente.
- Confirmação interativa por padrão, com prova de recuperação (`HEAD` atual, dica de `git reset --hard`) — sob a **ADR-008**.

**Fora de escopo:**

- **Consertar mais de um commit numa chamada só** — isso é o que `gtr author` e `gtr ai-trailers strip` já fazem, reescrevendo um pipeline fixo em cada commit do intervalo. O `overdub` roda o comando de conserto **uma vez**, só no alvo.
- **Commit raiz** — `<sha>^` não existe quando `<sha>` não tem pai, e o git recusa a faixa.
- **Checagem de `gtr` no `PATH` como item do `gtr doctor`** — a checagem existe, mas só dentro do próprio `overdub`, na hora em que importa.

**Isso mexe em histórico de verdade** — ao contrário de `commits check`/`bisect`, que não tocam nada. Cai sob a **ADR-008**: prova de recuperação, autorização explícita, confirmação dimensionando o alcance.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Commit-alvo** | O commit que o `overdub` conserta (`sha`). |
| **Comando de conserto** | O `argv` depois do primeiro `--`. Roda **uma vez**, na árvore de trabalho real do commit-alvo, sem shell. |
| **Comando de verificação** | O `argv` depois do segundo `--`, opcional. Roda via `commits.Repo.Check` no intervalo reescrito inteiro, depois do reencaixe. |
| **Editor de sequência** | O `GIT_SEQUENCE_EDITOR` que a rebase interativa invoca. Aqui é sempre `gtr overdub-sequence-step <sha>` — nunca um editor de verdade, nunca edição humana. |
| **Reencaixe** | O `git rebase --onto` final, que mantém o que vem depois do intervalo reescrito no lugar, só com hash novo. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/overdub/
├── plan.go      Plan: Head, Target, Subject, Count
├── result.go    Result: NewTarget, NewHead
├── step.go      SequenceStepCommand, MarkForEdit
└── repo.go      Repo: Ensure, Plan, Overdub, fix (privado), abort (privado)

internal/commits/  Interval (generalização de Range), Check, ArchiveExtractor — reusados pelo --verify
internal/exec/     CommandRunner — a mensagem de "executável não achado" nasce aqui

cmd/overdub.go     ensureGtrOnPath, splitAtOverdubDash, runOverdub, runOverdubVerify
cmd/checked_table  o --verify usa a mesma tabela do commits check, sem front próprio
```

**O `Interval` generaliza o `Range`**: `Range(base)` é o atalho de `Interval(base, "HEAD")`. Sem essa generalização, o `--verify` não teria como listar só o intervalo reescrito quando `--until` não é o `HEAD` de verdade.

**A mensagem própria para executável ausente** é uma pressão que nasce aqui e mora no runner compartilhado — mesmo padrão da **ADR-003**, em que a necessidade de uma feature vira capacidade geral: `commits check` e `bisect` também passam a nomear o comando ausente, em vez de repassar o erro cru do Go.

### 2.2 Fluxo

1. Verifica que `gtr` está no `PATH` — ver §8.1.
2. Verifica que o diretório atual é um repositório git.
3. Monta o `Plan`: `HEAD` atual (curto), `sha` resolvido (curto), assunto, contagem de `sha^..until`.
4. Mostra o plano e pede confirmação (`--yes` pula).
5. Confirmado: resolve `sha` e `until` para sha completo, validado como hex puro (`^[0-9a-f]{40}$`) — nunca o texto que o usuário digitou.
6. Descobre a ref atual (`symbolic-ref --short HEAD`, ou `rev-parse HEAD` se o `HEAD` já estava destacado).
7. `git -c core.abbrev=40 rebase -i <sha>^ <until>`, com `GIT_SEQUENCE_EDITOR=gtr overdub-sequence-step <sha>` — a rebase para exatamente no commit-alvo. O `core.abbrev=40` força sha completo no todo da rebase, para o editor de sequência casar por igualdade exata, sem ambiguidade de prefixo.
8. Roda o comando de conserto na árvore de trabalho real.
   - **Falhou** — executável não achado, saída diferente de zero, ou qualquer passo do git depois: `git rebase --abort` e `git checkout <ref>` de volta. Ver §8.1.
9. `git add -A`, `git commit --amend --no-edit --allow-empty`, `git rebase --continue`.
10. `git rebase --onto <novo-until> <until-antigo> <ref>` — recoloca o resto por cima.
11. Com `--verify`: `commits.Repo.Interval(novo-alvo^, novo-until)` mais `commits.Repo.Check` no intervalo reescrito, e reporta.

---

## 3. Requisitos funcionais

### RF-01 — Consertar um commit no lugar

`gtr overdub <sha> -- <comando>` roda `<comando>` uma vez, na árvore de trabalho real do commit `<sha>`, e emenda o resultado nele.

### RF-02 — Recolocar o resto por cima

Tudo entre `<sha>` (exclusive) e `--until` (`HEAD` se vazio) é reencaixado por cima do commit consertado, com hash novo em cascata e conteúdo idêntico.

### RF-03 — Confirmação com prova de recuperação

Antes de mexer em qualquer coisa, mostra quantos commits serão reescritos, o assunto do alvo e o `HEAD` atual, para o `git reset --hard` manual. Só prossegue com confirmação explícita.

### RF-04 — `--yes`

Pula a pergunta `[y/N]`, assumindo confirmado. É a única forma de chamar o `overdub` por script ou agente.

### RF-05 — `--verify`

Com um segundo `--` depois do comando de conserto, roda esse segundo comando via `commits check` no intervalo reescrito inteiro, e reporta quais commits ainda não se sustentam.

### RF-06 — Checagem de `gtr` no `PATH`

Antes de tocar em qualquer coisa — inclusive antes do `Ensure` de repositório — verifica que `gtr` é achável via `PATH`. A rebase interativa precisa invocar `gtr overdub-sequence-step` de novo, e quem resolve esse nome é o `git`, pelo `PATH`, nunca pelo caminho usado para chamar o comando atual.

### RF-07 — Abortar automaticamente quando o conserto falha

Se o comando de conserto não roda, sai diferente de zero, ou qualquer passo do git entre o início da rebase e o `rebase --continue` falha, o `overdub` roda `git rebase --abort` e `git checkout <ref>` sozinho, antes de reportar o erro.

### RF-08 — Mensagem própria para executável não achado

Quando o comando de conserto, ou o de verificação, não é achável no `PATH`, a mensagem nomeia o comando e sugere informar o caminho completo, em vez de repassar o erro cru do Go.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Nunca passa o comando de conserto pelo `--exec` da rebase.** O `--exec` roda a string por um shell, e o comando é arbitrário — misturar os dois abriria a mesma injeção que o `gtr author` já evita para a identidade. Roda direto pelo `internal/exec.Runner`, argv intacto. |
| **RN-02** | **`sha` e `until` só entram no `GIT_SEQUENCE_EDITOR` depois de resolvidos e validados como hex puro** (`^[0-9a-f]{40}$`). Nunca o texto que o usuário digitou — hex puro não tem metacaractere de shell para escapar. |
| **RN-03** | **ADR-008 completa**: prova de recuperação (`HEAD` atual impresso antes de confirmar), autorização explícita (`[y/N]` ou `--yes`) e confirmação dimensionando o alcance (quantos commits, qual o assunto do alvo). |
| **RN-04** | **Qualquer falha depois que a rebase começou dispara `git rebase --abort` e `git checkout <ref>`.** Nunca deixa o repositório em `HEAD` destacado com `.git/rebase-merge` no lugar — ver §8.1. |
| **RN-05** | **Se o abort ou o checkout de volta também falharem, isso é dito junto do erro original**, nunca engolido em silêncio: quem chama sempre sabe se o repositório ficou limpo. |
| **RN-06** | **Contexto próprio, com timeout, para o abort e o checkout de limpeza** — nunca o `ctx` da chamada que falhou, que pode já estar cancelado (`Ctrl+C` no meio do conserto). Um `ctx` cancelado recusaria o próprio `--abort`, e a rebase ficaria presa mesmo assim. |
| **RN-07** | **Não funciona no commit raiz** — `<sha>^` não existe quando `<sha>` não tem pai; o git recusa a faixa e o comando propaga o erro. |

### 4.1 Regras herdadas

- **Comandos rodam sem shell** — `RN-06` da [SRS — branches](branches.md), do qual a `RN-01` acima é a aplicação no caso mais perigoso: comando arbitrário do usuário dentro de uma rebase.
- **Saída no `stdout`, erro no `stderr`, e nenhuma linha terminando em espaço** — `RN-09` e `RN-10` da [SRS — branches](branches.md).

---

## 5. Interface

```console
$ gtr overdub 8a9b0c1 -- gofmt -w retry.go -- go test ./...
Isso vai reescrever 6 commits a partir de 8a9b0c1 "refactor: extract the retry loop".
HEAD atual: 0c1d2e3. Se o conserto falhar no meio, o gtr tenta git rebase --abort
sozinho; se isso também falhar, ou se o resultado final não agradar, git reset
--hard 0c1d2e3 desfaz.

Confirma? [y/N] y
Consertado. Novo HEAD: 4f8a1c9
Verificando 6 commits sobre 91e5a02^.

  verde  a92f6e1  refactor: extract the retry loop
  verde  ...

Os 6 se sustentam sozinhos.
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--until <sha>` | vazio | Até onde reescrever; vazio significa `HEAD` |
| `--yes` | `false` | Pula a confirmação interativa |

---

## 6. Testes

Cobertura: 100% em `cmd/overdub.go`, 100% em `internal/exec`, 99% em `internal/overdub` — o gap está documentado no próprio código: no `fix()`, uma colisão do fake de teste ao reusar `"rev-parse HEAD"` para dois momentos distintos. Comportamento real validado à mão.

### 6.1 Domínio — `internal/overdub/repo_test.go`, `step_test.go`

| Cenário | Teste |
|---|---|
| Caminho feliz: conserta só o alvo, reencaixa o resto | `TestOverdubRunsTheFixOnlyOnTheTargetAndReattachesTheRest` |
| `HEAD` já destacado antes de começar | `TestOverdubFallsBackToTheHeadSHAWhenDetached` |
| Sem `symbolic-ref` nem `rev-parse HEAD` para o reencaixe | `TestOverdubPropagatesTheDetachedFallbackFailure` |
| Conserto falha — não segue para o amend | `TestOverdubPropagatesTheFixCommandFailure` |
| **Aborta a rebase presa quando o conserto falha** | `TestOverdubAbortsARebaseStuckMidwayAfterTheFixFails` |
| Se o abort também falhar, isso é dito | `TestOverdubMentionsWhenTheAbortItselfFails` |
| Se o checkout de volta também falhar, isso é dito | `TestOverdubMentionsWhenTheCheckoutBackAlsoFails` |
| Comando de conserto que nem começa também aborta | `TestOverdubAbortsWhenTheCommandCannotStart` |
| Cada falha de git no meio vira erro | `TestOverdubPropagatesEachGitFailure` (tabela) |
| Sem `--until`, fecha em `HEAD` | `TestOverdubWithoutUntilDefaultsToHEAD` |
| `rev-parse` que não devolve sha completo é recusado | `TestOverdubRefusesAnAbbreviatedResolution` |
| `MarkForEdit` troca `pick`→`edit` só na linha certa, nunca por prefixo parcial | `TestMarkForEditTurnsPickIntoEditOnTheMatchingLine`, `TestMarkForEditNeverMatchesAPrefixOfAnotherSHA` |

### 6.2 Apresentação — `cmd/overdub_test.go`

| Cenário | Teste |
|---|---|
| Mostra o plano e pede confirmação | `TestOverdubShowsThePlanAndAsksConfirmation` |
| `--yes` pula a pergunta | `TestOverdubYesSkipsTheConfirmationPrompt` |
| Recusado sem confirmar, nada tocado | `TestOverdubDeclinedNeverTouchesGit` |
| `--verify` reporta sucesso e falha | `TestOverdubWithVerifyReportsSuccess`, `TestOverdubWithVerifyReportsFailure` |
| **Sem `gtr` no `PATH`, recusa antes de tocar em qualquer coisa** | `TestOverdubRefusesWithoutGtrOnPath` |
| Parsing dos dois `--` (conserto e verificação) | `TestOverdubRefusesWithoutTheDash`, `TestOverdubRefusesAnEmptyFixBeforeTheSecondDash`, `TestOverdubRefusesAnEmptyVerifyAfterTheSecondDash` |
| `--until` chegando ao domínio | `TestOverdubUsesTheUntilFlag` |
| Subcomando oculto marca o arquivo de todo de verdade | `TestOverdubSequenceStepCommandMarksTheTodoFile` |
| Falha de escrita em cada etapa da saída | `TestOverdubPropagatesWriteFailureAtEveryStep` |

### 6.3 `internal/exec` — `command_runner_test.go`

| Cenário | Teste |
|---|---|
| Executável não achado nomeia o comando e sugere caminho completo | `TestCommandRunnerFailsWhenTheCommandCannotStart` |
| Falha que **não** é executável ausente (sem permissão) não leva a dica errada | `TestCommandRunnerPropagatesAFailureThatIsNotAMissingExecutable` |

---

## 7. Rastreabilidade

| RF/RN | Commit | Teste |
|---|---|---|
| `RF-01`, `RF-02` | `a8bd32c` | `TestOverdubRunsTheFixOnlyOnTheTargetAndReattachesTheRest` |
| `RF-03` | `9ae7bd8` | `TestOverdubShowsThePlanAndAsksConfirmation` |
| `RF-04` | `161d9c9` | `TestOverdubYesSkipsTheConfirmationPrompt` |
| `RF-05` | `9ae7bd8` | `TestOverdubWithVerifyReportsSuccess`, `TestOverdubWithVerifyReportsFailure` |
| `RF-06` | `161d9c9` | `TestOverdubRefusesWithoutGtrOnPath` |
| `RF-07`, `RN-04`, `RN-05`, `RN-06` | `58b2885` | `TestOverdubAbortsARebaseStuckMidwayAfterTheFixFails`, `TestOverdubMentionsWhenTheAbortItselfFails`, `TestOverdubMentionsWhenTheCheckoutBackAlsoFails` |
| `RF-08` | `ff22831` | `TestCommandRunnerFailsWhenTheCommandCannotStart` |
| `RN-01`, `RN-02` | `a8bd32c` | `TestMarkForEditNeverMatchesAPrefixOfAnotherSHA`, `TestOverdubRefusesAnAbbreviatedResolution` |
| `RN-03` | `9ae7bd8` | `TestOverdubShowsThePlanAndAsksConfirmation` |
| `RN-07` | — | **Não testado** — ver §8.2 |
| `Interval` | `5eab3f3` | generaliza o `Range` para o `--verify` reusar |

---

## 8. Achados e lacunas

### 8.1 O que rodar contra um projeto real revelou

Usar o `overdub` num repositório de verdade, em Windows com PowerShell, para consertar um commit ao qual faltava um arquivo de configuração, expôs quatro coisas — e as quatro estão no comportamento de hoje:

1. **Sem `gtr` no `PATH`** (o binário chamado por caminho relativo), o que aparecia era o erro cru do editor de sequência do git, não uma mensagem do `overdub`. Daí a `RF-06`: a checagem vem antes de tudo, porque quem resolve `gtr overdub-sequence-step` é o git, pelo `PATH`.
2. **`cp` do PowerShell é alias de `Copy-Item`, não executável.** O `internal/exec.Runner` nunca passa por shell, de propósito, então o alias não resolve — e o erro cru do Go (`exec: "cp": executable file not found in %PATH%`) não dizia isso. Daí a `RF-08`, que mora no runner compartilhado e beneficia `commits check` e `bisect` também.
3. **O mais sério: comando de conserto que falha deixava a rebase parada no meio**, em `HEAD` destacado, com `.git/rebase-merge` no lugar. A recuperação exigia inspeção manual e `rebase --continue` na mão. Pior: a mensagem de confirmação prometia `git reset --hard`, que **não basta** nesse estado — deixaria `.git/rebase-merge` órfão. Daí a `RF-07` e a `RN-04`.
4. **Abortar sozinho não bastava.** `git rebase -i <sha>^ <until>` sempre destaca o `HEAD` antes do primeiro passo, porque `<until>` é um sha cru e não uma branch — é assim que o reencaixe final funciona. Então `git rebase --abort` sozinho devolve para `HEAD` destacado, não para a branch original. Daí o `git checkout <ref>` depois do abort.

Os quatro foram reproduzidos à mão contra um repositório real, antes e depois da correção correspondente.

### 8.2 Não testado, não avaliado

- **Commit raiz (`RN-07`) não tem teste automatizado** — o comportamento, erro do próprio git propagado, foi confirmado por raciocínio sobre `sha^`, não por um teste que exercita um repositório de um commit só.
- **O cenário de Windows/PowerShell não tem teste automatizado** — validado à mão contra um repositório real; o CI roda só `ubuntu-latest`, então a combinação exata (PowerShell, `cp` como alias) não é coberta.
- **O `gtr doctor` não ganhou uma checagem de `gtr` no `PATH`** — decisão de escopo: a checagem existe só dentro do próprio `overdub`.
- **`--until` combinado com `--verify` quando `--until` não é o `HEAD` real** — o `--verify` reconstrói o intervalo a partir de `NewTarget`/`NewHead`, que já refletem o `--until` pedido, então deveria funcionar; não há teste dedicado a essa combinação.
