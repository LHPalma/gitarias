---
titulo: SRS — ai-trailers strip
data: 2026-08-20
status: entregue
comando: gtr ai-trailers strip
pacotes:
  - cmd/ai_trailers.go
  - internal/aitrailers
commits:
  - 4c78418
  - ec77b46
fonte_externa: nenhuma
---

# SRS — ai-trailers strip

- **Data:** 2026-08-20
- **Feature:** `cmd/ai_trailers.go` (subcomando `strip`) + `internal/aitrailers`
- **Status:** **entregue** — commits `4c78418` (domínio) e `ec77b46` (comando)
- **Fonte externa:** nenhuma
- **Relacionados:** [SRS — ai-trailers list](ai-trailers-list.md), onde o que aqui se remove é detectado, e [SRS — author](author.md), de onde vem o padrão de reescrita em duas passagens

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr ai-trailers strip`, que **remove** trailers de autoria de IA já reconhecidos pelo `list`, com confirmação — para quem quer que o crédito no histórico vá só a humanos.

### 1.2 Escopo

**Entregue:** sem `--since`/`--until`, mexe só no `HEAD`, com um `commit --amend`; com qualquer um dos dois, reescreve toda a cadeia do período via rebase, um passo por commit através de um subcomando oculto do próprio `gtr`; prévia idêntica à tabela do `list`; as três guardas da **ADR-008**.

**Fora de escopo:** trocar autor ou committer de verdade — isso é o [`gtr author`](author.md). Remover trailer humano — nunca acontece, por construção.

### 1.3 O nome

`strip`, termo comum para "remover" nesse contexto. Sem apelido próprio — herda o `synth` do comando pai.

---

## 2. Descrição geral

```text
internal/aitrailers/
├── strip.go                 Strip(rawMessage, rawBlock, unfoldedBlock) (string, bool) — função pura
└── repo.go                  PlanStrip, Strip, StripHead, window; const StripStepCommand

cmd/ai_trailers.go           newAITrailersStripCommand, runAITrailersStrip, o subcomando oculto
```

Reaproveita o padrão de rebase em duas passagens que o `internal/author.Rewrite` estabeleceu primeiro, com uma diferença: lá a identidade nova é a **mesma** para toda a faixa e cabe no ambiente do processo; aqui a mensagem de substituição **difere por commit**, porque cada um pode ter um conjunto de trailers diferente.

Daí o **subcomando oculto** (`StripStepCommand`), invocado a cada passo pelo `--exec`, com cada chamada lendo e reescrevendo só o `HEAD` daquele passo.

---

## 3. Requisitos funcionais

### RF-01 — Sem período, mexe só no `HEAD`

Um `commit --amend` só, se houver trailer reconhecido para remover.

### RF-02 — Com período, reescreve a cadeia inteira

`--since` e/ou `--until` disparam uma rebase cobrindo o período, um `StripHead` por commit, reencaixando por cima o que vem depois de `until` sem reescrever conteúdo nenhum ali.

### RF-03 — Remove só o que bate, preserva o resto

Apaga só as linhas de trailer que casam uma assinatura de IA conhecida; corpo e trailers humanos saem **byte a byte** idênticos.

### RF-04 — Prévia e confirmação

Mostra a mesma tabela que o `list` mostraria para o mesmo período, mais `Recuperável com: git reset --hard <head>`; só executa com `[y/N]` confirmado. **Sem `--force`.**

### RF-05 — Nada para remover

Sem nenhum achado no período: mensagem específica, sem perguntar nada.

### RF-06 — Exige o subcomando no `PATH`

`gtr ai-trailers-strip-step` precisa estar acessível sob esse nome exato — mesma exigência que o `gh` já impõe para o `pr` e o `doctor --online`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **`Strip` é função pura.** Dados `(rawMessage, rawBlock, unfoldedBlock)`, devolve `(novaMensagem, mudou)`; nenhuma chamada de git dentro dela — só manipulação de string. |
| **RN-02** | **`rawBlock` (`%(trailers:only)`, sem unfold) precisa ser sufixo literal, byte a byte, de `rawMessage` (`%B`).** Medido contra o git de verdade antes de confiar nisso — é o que permite reconstruir a mensagem cortando exatamente esse sufixo do fim. |
| **RN-03** | **`unfoldedBlock` (`%(trailers:only,unfold)`, uma linha por trailer) é o único jeito confiável de casar cada linha contra as assinaturas.** Sem unfold, um trailer quebrado em mais de uma linha no bloco cru confundiria o casamento linha a linha. |
| **RN-04** | **Trailer humano que originalmente quebrasse em mais de uma linha sai reconstruído numa linha só** — perde a quebra, não o conteúdo. Limite conhecido e aceito: nenhum trailer que este pacote reconhece quebra assim na prática. |
| **RN-05** | **Bloco esvaziado por completo não deixa parágrafo em branco para trás**; se sobra ao menos um trailer humano, ele fica precedido pela linha em branco que o separa do corpo. |
| **RN-06** | **"Sem reescrever" é conteúdo, nunca é hash.** A cauda preservada mantém árvore, mensagem e autoria idênticas, mas o SHA dela muda mesmo assim — o hash do pai entra no cálculo do commit, e qualquer reescrita a montante cascateia para a frente. Medido contra o git real, não suposto. Só o que já estava **antes** do início do período fica literalmente intocado, hash incluso. |
| **RN-07** | **`commit --amend` precisa de `--allow-empty`.** A operação só mexe na mensagem; um commit que já era vazio de propósito não pode ser bloqueado por uma checagem irrelevante ao que o `Strip` faz — achado testando contra repositório real. |
| **RN-08** | **O subcomando oculto nunca é chamado direto por um humano.** Existe porque cada passo da rebase precisa reescrever uma mensagem **diferente**, e o `--exec` só aceita um comando fixo: quem varia por commit é o próprio `StripHead`, rodando de novo a cada `HEAD` do passo. |

---

## 5. Achados de implementação

### 5.1 Por que precisou de subcomando oculto, diferente do `author`

No `author.Rewrite`, a identidade nova é a **mesma** para toda a faixa — cabe inteira no `--exec`, via variável de ambiente. Aqui a mensagem resultante depende do **conteúdo de cada commit**, ou seja, de quais trailers ele carrega, então não existe um `--exec` fixo que sirva para todos.

A solução foi delegar a decisão a um subcomando que lê o próprio `HEAD` a cada passo, em vez de tentar embutir lógica condicional dentro da string do `--exec`.

### 5.2 `--allow-empty` foi descoberto testando, não suposto

O `commit --amend` recusava manter um commit vazio sem a flag — e isso só apareceu testando contra um repositório de verdade com um commit `--allow-empty` pré-existente no meio da faixa. Como o `Strip` e o `StripHead` só tocam a mensagem, a emptiness da árvore é irrelevante, e a flag entrou sem reserva.

### 5.3 O sufixo literal foi verificado, não documentado

Toda a lógica de reconstrução do `strip.go` se apoia nisso: cortar o `rawBlock` do fim do `rawMessage` só funciona porque os dois foram medidos contra o git real e confirmados idênticos byte a byte naquele ponto. **Não é garantia documentada pelo git** — é comportamento observado, e travado por teste.

---

## 6. Testes

**100% de statements** na parte de remoção do `internal/aitrailers` e nos arquivos de `cmd` do `strip`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Remove só o trailer de IA, preserva humano e corpo | `RF-03`, `RN-01` |
| 2 | Bloco inteiro batendo | Removido por completo, sem parágrafo vazio — `RN-05` |
| 3 | Mensagem sem nenhum trailer, ou nenhum bate | Devolvida intocada, `changed=false` |
| 4 | Linha malformada no bloco | Mantida defensivamente |
| 5 | `StripHead` com e sem trailer de IA | Amend roda ou não — `RF-01` |
| 6 | `StripHead` com registro incompleto do log | Erro |
| 7 | `StripHead`: falha do log, falha do amend | Erro propagado |
| 8 | `PlanStrip` no `HEAD`, com e sem achado | `RF-01` |
| 9 | `PlanStrip` com período delega ao `List` | `RF-02` |
| 10 | `window`: extremos do período; nada no período; falha do log | Base da reescrita por faixa |
| 11 | `Strip` no `HEAD` delega ao `StripHead` | `RF-01` |
| 12 | `Strip` com faixa: rebase, reencaixe, `HEAD` destacado (fallback), período vazio, falha em cada etapa | `RF-02`, `RN-06` |
| 13 | Strip no `HEAD` pergunta e reescreve; cancelado nada muda; sem achado não pergunta | `RF-04`, `RF-05` |
| 14 | Strip de uma faixa pergunta e reescreve | `RF-02`, `RF-04` |
| 15 | `--since`/`--until` inválida | Erro |
| 16 | Fora de um repositório | Erro |
| 17 | Falha do plano, da reescrita, de escrita (com e sem achados), de leitura da confirmação | Propagadas |
| 18 | Argumento posicional sobrando | Erro |
| 19 | `ai-trailers-strip-step` amenda o `HEAD` direto | `RF-06`, `RN-08` |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `4c78418` | `internal/aitrailers`: `Strip` puro, `PlanStrip`, `Strip`, `StripHead`, `window` e o `--allow-empty` — `RF-01` a `RF-03`, `RN-01` a `RN-07`, §5.1 a §5.3 |
| `ec77b46` | `gtr ai-trailers strip`: o comando, a prévia idêntica à do `list`, a confirmação e o subcomando oculto — `RF-04` a `RF-06`, `RN-08` |

---

## 8. Follow-ups conhecidos

- **Trailer humano multi-linha reconstruído numa linha só** — `RN-04`, limite conhecido e aceito.
- **`%(trailers:only,unfold)` não confirmado contra um git 2.22 real** — mesma lacuna da [SRS — ai-trailers list](ai-trailers-list.md).
- **Sem `--force`** — mesma disciplina consciente do [`gtr author`](author.md), decisão da **ADR-008**.
