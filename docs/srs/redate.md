---
titulo: SRS — redate
data: 2026-10-04
status: entregue
comando: gtr redate
apelido: reissue
pacotes:
  - cmd/redate.go
  - internal/redate
commits:
  - 683e5a8
fonte_externa: nenhuma
---

# SRS — redate

- **Data:** 2026-10-04
- **Feature:** `cmd/redate.go` + `internal/redate`
- **Status:** **entregue** — commit `683e5a8`
- **Fonte externa:** nenhuma
- **Relacionados:** [SRS — author](author.md), de onde vem o padrão de reescrita em duas passagens, e [SRS — ai-trailers strip](ai-trailers-strip.md), de onde vem o `--since`/`--until`

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr redate`, que **troca a data de commits já existentes** — a de autoria e a de committer — com prévia e confirmação. É o `GIT_COMMITTER_DATE=<data> git commit --amend --date=<data> --no-edit` à mão, e o mesmo para um período inteiro.

### 1.2 Escopo

**Entregue:** sem `--since`/`--until`, só o `HEAD`; com qualquer um dos dois, o período por rebase, com o que vem depois reencaixado; `--since` sozinha até hoje; as três guardas da **ADR-008**.

**Fora de escopo:** datas diferentes por commit — todo commit do período recebe a mesma. Deslocar datas (`+2h`) — a data é absoluta. Trocar autor ou committer — isso é o [`gtr author`](author.md).

### 1.3 O nome

**`redate`**, direto: "trocar a data" sem prometer mais do que isso.

**`reissue` é apelido**, e um easter egg: relançar um disco antigo com data de lançamento nova, sem regravar nada — é o que o comando faz com um commit. Fica fora da ajuda da raiz para que o nome anunciado seja o óbvio; aparece em `gtr redate --help`.

---

## 2. Descrição geral

```text
internal/redate/
├── plan.go        Plan e Commit: o que Rewrite afetaria
└── repo.go        Parse, Plan, Rewrite, window; ErrMergesInTail

cmd/redate.go      o comando, as três flags e o ritual de confirmação
```

Mesma forma do [`author`](author.md): a data nova é **igual** para toda a faixa, então cabe inteira no `--exec` da rebase e no ambiente do processo — nada de subcomando oculto como no `strip`.

---

## 3. Requisitos funcionais

### RF-01 — Sem período, mexe só no `HEAD`

`git commit --amend --no-edit --allow-empty --date=<data>`, com `GIT_COMMITTER_DATE=<data>` no ambiente.

### RF-02 — Com período, reescreve a cadeia inteira

`--since` e/ou `--until` disparam `git rebase --rebase-merges <oldest>^ <newest> --exec 'git commit --amend ...'`, e depois `git rebase --onto` para reencaixar o que vem depois de `<newest>`. **`--until` vazio vale hoje**: o período não tem teto, e a cauda é vazia.

### RF-03 — Prévia e confirmação

Imprime os commits que de fato serão reescritos (hash, data atual, assunto), a cauda quando houver, e `Recuperável com: git reset --hard <head>`. Só executa com `[y/N]` confirmado. **Sem `--force`.**

### RF-04 — Nada no período

Mensagem específica, sem perguntar nada.

### RF-05 — O commit raiz entra

Se o commit mais antigo do período não tem pai, a rebase usa `--root` em vez de `<oldest>^`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **`--date` é validado e reescrito antes de chegar ao `--exec`.** O `--exec` roda por um shell; `Parse` lê o texto em `AAAA-MM-DDTHH:MM:SS` e devolve a forma normalizada, de modo que só dígitos, `-`, `:` e `T` passam. Mesmo cuidado do `author`, que leva a identidade pelo ambiente pelo mesmo motivo. |
| **RN-02** | **A data de committer viaja por `GIT_COMMITTER_DATE`**, herdada pela rebase e por cada amend. **A de autoria vai em `--date`**, dentro do `--exec`. As duas valem o mesmo instante. |
| **RN-03** | **`--rebase-merges` na rebase do período.** Medido contra o git de verdade: sem a flag, a rebase descarta o commit de merge e os commits do branch ficam em linha reta. Com ela, o merge e a forma do histórico sobrevivem e o `--exec` roda também sobre o merge. |
| **RN-04** | **Merge na cauda é recusado** (`ErrMergesInTail`). A segunda passagem, `rebase --onto`, achata merge sem `--rebase-merges` e, com ele, **duplica o histórico antigo** de qualquer branch que nasceu dentro do período: o branch continua apoiado nos commits velhos. Medido, não suposto. Só vale com `--until` fechando antes do `HEAD`; sem `--until` a cauda é vazia. |
| **RN-05** | **A prévia lista o que a rebase reescreve, não o que o filtro de data casou.** É `<oldest>^..<newest>` — por isso inclui o commit de branch mergeado cuja data é anterior ao `--since`. |
| **RN-06** | **A cauda preservada perde a data de committer.** Árvore, mensagem e data de autoria ficam, mas a rebase carimba a hora de agora como committer, e o hash muda — o do pai entra no cálculo. Só o que está antes do período fica intocado, hash incluso. |
| **RN-07** | **`--allow-empty` no amend.** A operação só mexe na data; um commit vazio de propósito não pode ser barrado por isso. Mesma lição do `strip` (RN-07 de lá). |
| **RN-08** | **`--since` vira `<data> 00:00:00` e `--until`, `<data> 23:59:59`**, explícitas — `git log --since=<data>` sozinho vale a hora de agora, não o começo do dia (mesma razão do `gtr profile`). |

---

## 5. Achados de implementação

### 5.1 O primeiro desenho achatava o histórico

A primeira versão copiou a rebase do `author` e do `strip`, sem `--rebase-merges`. Rodando contra um repositório com merge, o commit de merge sumiu e o branch virou linha reta — e este próprio repositório tem merge de PR em quase todo trecho. Daí `RN-03`.

### 5.2 `--rebase-merges` na segunda passagem era pior

Levar a flag também ao `rebase --onto` parecia o caminho óbvio. Medido: com um merge na cauda cujo branch nasceu dentro do período, o histórico antigo reapareceu **ao lado** do novo — commits duplicados, o branch ainda apoiado nos pais velhos. Achatar e duplicar destroem a forma do histórico sem avisar, então o comando recusa (`RN-04`) em vez de escolher um.

### 5.3 O commit raiz precisou de `--root`

`<oldest>^` não existe para o commit raiz, e `--since` cobrindo o histórico inteiro é um uso natural. Daí `RF-05`. O `author` ainda não alcança a raiz (`RN-10` de lá).

---

## 6. Testes

| # | Cenário | Esperado |
|---|---|---|
| 1 | `Parse` normaliza; recusa vazio, só data, espaço no lugar do `T`, mês 13 e injeção de shell | `RN-01` |
| 2 | `Plan` no `HEAD`; propaga a falha | `RF-01` |
| 3 | `Plan` de período até o `HEAD`, janela vazia, raiz incluída | `RF-02`, `RF-04`, `RF-05` |
| 4 | `Plan` com cauda linear conta; com merge recusa | `RN-04` |
| 5 | `Rewrite` no `HEAD` amenda com as duas datas | `RF-01`, `RN-02` |
| 6 | `Rewrite` de período: rebase e reencaixe; raiz com `--root`; `HEAD` destacado cai no SHA; janela vazia não toca em nada; falha da rebase propaga | `RF-02`, `RF-05` |
| 7 | Comando no `HEAD` pergunta e amenda; cancelado (`n`, Enter, EOF) não reescreve | `RF-03` |
| 8 | `--since` sozinha não manda `--until` ao git | `RF-02` |
| 9 | Período vazio; merge na cauda recusado antes de perguntar | `RF-04`, `RN-04` |
| 10 | Sem `--date`; `--date`, `--since` ou `--until` inválidos não tocam no git | `RN-01` |
| 11 | Fora de repositório; falha do amend sem `Pronto.`; argumento sobrando | Erro |
| 12 | `reissue` e a ajuda da raiz | Mesma saída; apelido não anunciado |

Também rodado à mão contra o git real, em repositório descartável: `HEAD` só, `--since` sozinha com merge no período, histórico inteiro com a raiz, cauda com merge recusada, cancelamento e data inválida.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `683e5a8` | `internal/redate` e `gtr redate`: o comando, a prévia, a confirmação, a rebase do período e as recusas — `RF-01` a `RF-05`, `RN-01` a `RN-08` |

---

## 8. Follow-ups conhecidos

- **Todo commit do período recebe a mesma data.** Preservar a ordem e o espaçamento exigiria uma data por commit, e portanto um subcomando oculto como o do `strip`. Não pedido.
- **Cauda com merge é recusada, não suportada.** Reescrever só um trecho do meio de uma história com merges fica de fora até alguém precisar.
- **Sem `--force`** — mesma disciplina consciente do [`gtr author`](author.md), decisão da **ADR-008**.
