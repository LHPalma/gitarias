---
titulo: SRS — author
data: 2026-08-18
status: entregue
comando: gtr author
apelido: blame-someone-else
pacotes:
  - cmd/author.go
  - internal/author
  - internal/git
commits:
  - 07568dc
  - 5162e6a
  - 6e3b753
  - 372ed9e
  - f889c8f
fonte_externa: nenhuma
---

# SRS — author

- **Data:** 2026-08-18
- **Feature:** `cmd/author` + `internal/author`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr author`, que **reescreve a autoria de commits já existentes** — nome e e-mail, autor e committer — quando a identidade de git em vigor no momento do commit estava errada. Cobre desde o caso mais comum (o commit mais recente, autoria trocada por config errada na máquina) até uma faixa inteira.

O `--reset` é o companheiro de recuperação: descarta o que a reescrita, ou qualquer outra coisa, deixou para trás.

### 1.2 Escopo

**Entregue:** reescrita do commit mais recente sem `--base`; reescrita de uma faixa `base..HEAD` ou `base..until`, com o que vem depois de `until` preservado; `--commit` como açúcar sobre um commit único arbitrário; `--reset <sha>` como wrapper de `git reset --hard`; prova de recuperação, autorização explícita e confirmação escopada ao tamanho do estrago — **ADR-008**.

**Fora de escopo:** qualquer reescrita que não seja nome e e-mail — mudar mensagem, dividir ou espremer commits não é o `author`. Commit raiz via `--commit` — não tem pai, e é a única borda que fica em aberto (`RN-10`).

### 1.3 O nome

`author`, direto. O apelido `blame-someone-else` é referência ao projeto de mesmo nome — e a piada é literal, porque o comando atribui a autoria a **qualquer** nome e e-mail informados, não só corrige a identidade de quem roda. Mesma família de humor que o [`blame-ai`](blame-ai.md) puxaria depois.

---

## 2. Descrição geral

```text
internal/author/             domínio da reescrita — não imprime nada
├── plan.go                  Plan: o que Rewrite afetaria
├── reset_plan.go            ResetPlan: o que Reset afetaria
└── repo.go                  Plan, Rewrite, PlanReset, Reset, Ensure

cmd/author.go                o comando, as seis flags e o ritual de confirmação
```

O `Runner` deste domínio **estende** o `git.Runner` com `RunWithEnv` — a identidade nova viaja pelo ambiente do processo, nunca por dentro do argumento que o rebase manda para um shell. Ver `RN-01`.

---

## 3. Requisitos funcionais

### RF-01 — Reescrever o commit mais recente

Sem `--base`, `gtr author --name --email` roda um `commit --amend --no-edit --reset-author` com a identidade nova no ambiente. Só o SHA do topo muda.

### RF-02 — Reescrever uma faixa, com o rabo preservado

Com `--base`, reescreve `base..HEAD` — ou `base..until`, quando `--until` é dado — inteira. O que vem depois de `until` é **preservado**, com mesmo conteúdo e mesma autoria, e apenas reencaixado em cima do trecho reescrito.

### RF-03 — `--commit`, açúcar sobre um commit único arbitrário

`--commit <sha>` reescreve só aquele commit, em qualquer ponto do histórico, preservando tudo antes e tudo depois. Por baixo equivale a `--base <sha>^ --until <sha>` — sem mecanismo novo no domínio —, mas a prévia nomeia o commit diretamente, sem vazar a sintaxe de `^` do git. Incompatível com `--base` e `--until`.

### RF-04 — Prévia e confirmação obrigatórias

Antes de tocar em qualquer coisa, mostra o que vai mudar — quantos commits, quem são os autores atuais no intervalo, quantos serão preservados — e a linha `Recuperável com: git reset --hard <head>`.

Só executa com confirmação explícita (`y`, `yes`, `s`, `sim`); qualquer outra resposta, inclusive Enter vazio, cancela sem tocar em nada. **Não há `--force`.**

### RF-05 — `--reset <sha>`

Wrapper de `git reset --hard <sha>`. Mesmo ritual de prévia e confirmação da `RF-04`, mais um aviso separado quando há mudança não commitada em arquivo rastreado — o `--hard` descarta isso **sem deixar rastro nenhum, nem no reflog**. Mutuamente exclusivo com todas as flags de reescrita.

### RF-06 — Sem `--format`

O `author` é interativo, como o [`undo`](undo.md) — não existe consumo programático da saída dele, só leitura humana antes de confirmar.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **A identidade nova viaja pelo ambiente do processo** (`GIT_AUTHOR_NAME`/`EMAIL`, `GIT_COMMITTER_NAME`/`EMAIL`), **nunca por dentro do argumento `--exec` do rebase.** O `--exec` roda a string por um shell; interpolar nome ou e-mail vindos de quem chama abriria injeção pela própria informação que o comando existe para reatribuir. Medido com um nome contendo `$(...)` e crase antes de decidir por essa forma: com o ambiente, o git tratou o texto como identidade e nada executou; interpolado no `--exec`, teria executado. |
| **RN-02** | **Faixa que não vai até o `HEAD` exige duas passadas de rebase.** A primeira reescreve `base..until` com o `HEAD` destacado nele; a segunda reencaixa (`rebase --onto`) o que ficou para trás, sem tocar no conteúdo. Quando `until` já é o `HEAD`, o reencaixe não tem nada para mover e só atualiza a ref — testado assim contra git de verdade, sem atalho separado para esse caso. |
| **RN-03** | **Preservado é conteúdo, nunca é identidade de commit.** "Mesmo conteúdo, mesma autoria" vale para o rabo reencaixado, mas os **SHAs** desses commits mudam mesmo assim — o hash de um commit inclui o hash do pai, e o pai mudou. Vale dizer explicitamente porque a mesma técnica é reaproveitada em `internal/aitrailers`, com o mesmo efeito. |
| **RN-04** | **Faixa sem nenhum commit é erro nomeando o intervalo**, nunca operação silenciosa sobre nada. |
| **RN-05** | **Contagem ilegível do `rev-list` é erro**, não zero silencioso — uma saída inesperada não pode virar "nenhum commit" por acidente. |
| **RN-06** | **A prévia lista os autores distintos atuais no intervalo**, e nunca afirma um autor só fora do caso de commit único. `--base` e `--until` cortam por alcançabilidade, não por autoria: numa `main` desatualizada, o intervalo pode incluir commits de outra pessoa, e a lista existe para pegar isso antes de reatribuir tudo. |
| **RN-07** | **`--reset` é mutuamente exclusivo com `--name`, `--email`, `--base`, `--until` e `--commit`.** São dois modos do mesmo comando, nunca combináveis. |
| **RN-08** | **Confirmação nunca é pedida sem que o plano tenha sido calculado com sucesso primeiro.** Falha ao planejar é erro antes de qualquer pergunta — nunca se põe o operador diante de um `[y/N]` sem saber o que está confirmando. |
| **RN-09** | **Toda operação é recuperável, e a linha de recuperação sai antes da pergunta, nunca depois** — **ADR-008**. Não há `--force` para pular a confirmação em nenhum dos dois modos. |
| **RN-10** | **`--commit` não funciona no commit raiz.** Ele não tem pai, e `<raiz>^` não resolve. O erro que sai é o do próprio git, cru, sem tradução — a única borda que este açúcar deixa em aberto, por decisão de não tratar um caso raro com código extra. |

---

## 5. Achados de implementação

### 5.1 Por que uma passada só não bastava

`git rebase base untilSHA --exec '...'` sozinho já reescreve exatamente `base..untilSHA` — mas deixa o `HEAD` **destacado** na ponta nova, sem mover a ref original da branch. A ref original ainda aponta para o estado antigo, rabo incluído.

A segunda passada (`rebase --onto <novo until> <until antigo> <ref>`) pega essa ref original e reencaixa `until antigo..ref` — o rabo, sem mudança nenhuma de conteúdo — em cima do novo `until`, só então movendo a ref para lá. É o mecanismo por trás da `RN-02`.

### 5.2 A defesa contra injeção foi medida, não assumida

A suspeita óbvia com qualquer `--exec` de rebase é injeção de shell. A defesa da `RN-01` foi confirmada empiricamente: `TestRewriteNeverInterpolatesTheIdentityIntoTheExecString` roda com um nome literal contendo `$(touch /tmp/pwned)` e crases, e confere que a chamada de `--exec` continua sendo o mesmo literal fixo, byte a byte, independente do que o autor mandar como nome.

### 5.3 `HEAD` destacado exige um fallback para nomear a ref

O reencaixe final precisa de uma referência para onde apontar a branch depois. `symbolic-ref --short HEAD` dá o nome da branch quando existe; sem branch — `HEAD` já destacado antes de rodar o `author` — cai para `rev-parse HEAD`, e o próprio SHA serve de "nome" para o `rebase --onto`, porque não há branch para mover.

### 5.4 `--commit` é tradução de flag, não mecanismo novo

A prova de que `--commit <sha>` é só açúcar: nenhuma mudança em `internal/author` acompanha `372ed9e`, o commit que introduz a flag. Toda a tradução (`<sha>^` como base, `<sha>` como until) e o cuidado de não vazar o `^` na prévia vivem inteiros em `cmd/author.go`.

### 5.5 `--reset` reaproveita o ritual, não o código de reescrita

O `--reset` não passa por `Rewrite`; é um `Reset` à parte que só chama `git reset --hard`. O que se repete entre os dois modos é a **forma** da interação — planejar, mostrar, perguntar, só então agir — e essa forma é a mesma que o [`undo`](undo.md) já usava antes do `author` existir.

---

## 6. Testes

**100% de statements** em `internal/author` e em `cmd/author.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Sem `--base`, confirmado | `commit --amend --reset-author` com a identidade no ambiente — `RF-01` |
| 2 | Sem `--base`, recusado (Enter vazio) | Nada roda, mensagem de cancelamento — `RF-04` |
| 3 | `--base main`, faixa até o `HEAD` | Uma passada de rebase, sem aviso de rabo preservado — `RF-02` |
| 4 | `--base main --until <sha>`, com rabo | Duas passadas, aviso do rabo, `rebase --onto` roda — `RF-02`, `RN-02` |
| 5 | `HEAD` já destacado antes de rodar | Fallback pelo SHA em vez do nome de branch — §5.3 |
| 6 | Nome com `$(...)` e crase | O `--exec` continua o mesmo literal fixo — `RN-01`, §5.2 |
| 7 | `--commit <sha>` | Traduz para `--base <sha>^ --until <sha>`; a prévia nomeia o commit sem `^` visível — `RF-03` |
| 8 | `--commit` com `--base` ou `--until` | Erro antes de tocar no git — `RF-03` |
| 9 | Faixa sem nenhum commit | Erro nomeando o intervalo — `RN-04` |
| 10 | Contagem ilegível do `rev-list` | Erro, não zero — `RN-05` |
| 11 | Faixa com dois autores distintos | Os dois listados, em ordem — `RN-06` |
| 12 | `--reset <sha>`, árvore limpa, confirmado | `reset --hard` roda, `Pronto.` — `RF-05` |
| 13 | `--reset` com árvore suja | Aviso separado de descarte sem rastro — `RF-05` |
| 14 | `--reset` recusado | Nada roda — `RF-04` |
| 15 | `--reset` combinado com qualquer flag de reescrita (as cinco, em tabela) | Erro antes de tocar no git — `RN-07` |
| 16 | `--name`/`--email` faltando, em cada combinação | Erro antes de tocar no git |
| 17 | `--until` sem `--base` | Erro |
| 18 | Falha ao planejar, nos dois modos | Erro antes de perguntar `Confirma?` — `RN-08` |
| 19 | Falha na leitura da confirmação (stdin quebrado), nos dois modos | Erro |
| 20 | Falha do amend, do rebase, do reencaixe e do reset, cada uma | Erro propagado |
| 21 | Fora de um repositório, nos dois modos | Erro do `Ensure` |
| 22 | `blame-someone-else` roda o mesmo comando, e não aparece na ajuda da raiz | §1.3 |
| 23 | Argumento posicional sobrando | Erro |
| 24 | Contexto cancelado, em toda operação do domínio | Recusa |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `07568dc` | O port ganha `RunWithEnv` — a identidade pelo ambiente, `RN-01` |
| `5162e6a` | `internal/author.Repo`: `Plan` e `Rewrite` nas três formas (topo, faixa até `HEAD`, faixa com rabo) — `RF-01`, `RF-02`, `RN-01` a `RN-05`, §5.1, §5.2 |
| `6e3b753` | `gtr author`: o comando, `--base`/`--until`, prévia com autores atuais, confirmação obrigatória, sem `--format` — `RF-04`, `RF-06`, `RN-06`, `RN-08`, `RN-09` |
| `372ed9e` | `--commit`, açúcar sobre `--base <sha>^ --until <sha>`, sem mudança no domínio — `RF-03`, `RN-10`, §5.4 |
| `f889c8f` | `--reset`, o wrapper de `git reset --hard` com o mesmo ritual — `RF-05`, `RN-07`, §5.5 |

---

## 8. Follow-ups conhecidos

- **Sem `--force` em nenhum dos dois modos** — decisão consciente da **ADR-008**. Se algum dia um uso programático (script, hook) precisar pular a pergunta, isso reabre a discussão de autorização explícita sem prompt interativo.
- **`--commit` no commit raiz permanece sem tratamento** — `RN-10`. Aceito porque é caso raro e o erro cru do git já é compreensível o bastante.
- **A técnica de reescrita em duas passadas (`RN-02`) é reaproveitada** em `internal/aitrailers` — ver [SRS — ai-trailers strip](ai-trailers-strip.md) e [SRS — blame-ai](blame-ai.md) para as variações que cada uma precisou: um `--exec` fixo contra um por commit, via subcomando oculto.
