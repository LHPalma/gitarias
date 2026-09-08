---
titulo: SRS — blame-ai
data: 2026-08-20
status: entregue
comando: gtr blame-ai
pacotes:
  - cmd/blame_ai.go
  - internal/aitrailers
commits:
  - e6da228
  - 15c7af8
fonte_externa: nenhuma
---

# SRS — blame-ai

- **Data:** 2026-08-20
- **Feature:** `cmd/blame_ai.go` + `internal/aitrailers`
- **Status:** **entregue** — commits `e6da228` (domínio) e `15c7af8` (comando)
- **Fonte externa:** nenhuma
- **Relacionados:** [SRS — ai-trailers list](ai-trailers-list.md), cuja detecção valida o que aqui se fabrica, e [SRS — author](author.md), de onde vem a ideia de reatribuir crédito — mas sem trocar identidade real

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr blame-ai`, que **acrescenta** um trailer de coautoria de IA fabricado a um commit — o oposto do [`ai-trailers strip`](ai-trailers-strip.md) —, mantendo autor e committer reais intactos.

Nasceu como piada, o caminho contrário ao `strip`, mas usa mecanismo real e testado contra git de verdade.

### 1.2 Escopo

**Entregue:** `--tool` obrigatória (`claude` ou `copilot`); `--commit` opcional, com `HEAD` como padrão; trailer canônico fabricado por `Signature`, compartilhando a constante de e-mail com o `detect.go` para nunca divergir da detecção; prévia e confirmação (**ADR-008**); reaproveitamento do rebase-e-reencaixe para alvo fora do `HEAD`, **sem** subcomando oculto.

**Fora de escopo:** mudar autor ou committer de verdade — isso é o [`gtr author`](author.md). Funcionar no commit raiz via `--commit` — sem pai, mesma limitação que o `gtr author --commit` já tem.

### 1.3 O nome

`blame-ai`, em inglês — combina com o `blame-someone-else`, apelido do `author`, e com o padrão de nomes-piada em inglês já estabelecido no projeto.

---

## 2. Descrição geral

```text
internal/aitrailers/
├── signature.go             Signature(name) — trailer canônico para claude e copilot
├── plan.go                  BlamePlan: o HEAD, o commit-alvo, o assunto e o trailer a acrescentar
└── repo.go                  Blame, blameHead, PlanBlame

cmd/blame_ai.go              o comando, --commit/--tool, prévia e confirmação
```

O `Signature` compartilha a constante de e-mail com o `detect.go` — a mesma, no mesmo arquivo de origem — para que fabricação e detecção nunca divirjam sobre "o que é" a assinatura de cada ferramenta. Provado por teste de *round-trip*: o trailer que o `Signature` fabrica é exatamente o que a detecção reconheceria de volta.

---

## 3. Requisitos funcionais

### RF-01 — `--tool` obrigatória

`claude` ou `copilot`; qualquer outro valor é erro nomeando os dois aceitos, verificado **antes** de tocar no git.

### RF-02 — Alvo do trailer

Sem `--commit`, o `HEAD`; com `--commit <sha>`, o commit nomeado, em qualquer ponto do histórico.

### RF-03 — Acrescentar sem substituir

Adiciona o trailer canônico (`Co-Authored-By: <Nome> <e-mail>`) preservando corpo e trailers humanos existentes — nunca troca autor ou committer real, ao contrário do `gtr author`.

### RF-04 — Prévia e confirmação

Nomeia o commit alvo, o assunto dele e a linha exata do trailer a acrescentar, mais a linha de recuperação; `[y/N]` obrigatório.

### RF-05 — Commit raiz via `--commit`

Falha do mesmo jeito que o `gtr author --commit` já falha: sem pai, `<raiz>^` não resolve, e sai o erro cru do git, sem tradução.

### RF-06 — Não duplica

Rodar de novo sobre um commit que já tem o mesmo trailer não duplica a linha — comportamento nativo do `git interpret-trailers`, não lógica escrita à mão.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **`Signature` devolve exatamente o trailer que a detecção reconheceria de volta** — garantido por `TestSignatureRoundTripsThroughDetection`. Fabricação e detecção nunca podem divergir sobre a mesma assinatura. |
| **RN-02** | **Só "claude" e "copilot" são nomes aceitos** — os mesmos dois que o `detect.go` sabe reconhecer. Qualquer outro é erro, nunca um trailer genérico ou inventado. |
| **RN-03** | **O ID numérico da conta bot do Copilot na fabricação é plausível, não verificado contra a conta real do GitHub.** A detecção não depende desse número — só do nome "copilot" e do domínio `@users.noreply.github.com` —, então o valor exato não muda se bate ou não na volta. |
| **RN-04** | **A colocação do trailer é delegada inteira ao `git interpret-trailers`, nunca reimplementada à mão.** Ele já resolve sozinho se abre parágrafo novo ou entra em bloco existente, e não duplica um trailer idêntico já presente — medido contra o git antes de confiar. |
| **RN-05** | **A entrada para o `interpret-trailers` precisa terminar em quebra de linha** para abrir parágrafo novo quando a mensagem não tem corpo, só assunto — sem isso, o trailer gruda direto no assunto, sem a linha em branco que o git exige para reconhecer aquilo como trailer depois. |
| **RN-06** | **Alvo fora do `HEAD` reaproveita a mesma dança de rebase-e-reencaixe do `strip`, mas sem subcomando oculto.** O trailer a acrescentar é fixo do início ao fim da chamada, então o `--exec` pode ser um pipeline literal de git puro (`log \| interpret-trailers \| commit --amend`), com **nada do chamador** interpolado nele. |
| **RN-07** | **`commit --amend` usa `--allow-empty`, mesma razão do `strip`:** a operação só muda a mensagem, e a emptiness da árvore é irrelevante. |

---

## 5. Achados de implementação

### 5.1 O bug real: a quebra de linha final apagada pelo `Run`

O `Run` do `gtr` apara a saída do processo, então `git log -1 --format=%B` de um commit **sem corpo**, só com assunto, chegava ao `interpret-trailers` sem a quebra de linha final que ele exige para abrir um parágrafo novo.

O trailer grudava direto no assunto, sem a linha em branco separadora — e depois **nada** reconhecia aquilo como trailer, nem o próprio `ai-trailers list`.

Corrigido restaurando a quebra (`strings.TrimRight(message, "\n")+"\n"`) antes de repassar, com regressão coberta por `TestBlameHeadRestoresTheTrailingNewline`. **Achado testando contra repositório real**, não contra o fake — que não reproduz o comportamento de apara do `Run` de verdade.

### 5.2 Três soluções diferentes para "o `--exec` roda por um shell"

Vale comparar as três features que reescrevem histórico por rebase:

- **`author.Rewrite`** leva a identidade nova pelo **ambiente do processo**, porque ela é a mesma para toda a faixa;
- **`ai-trailers strip`** usa um **subcomando oculto**, porque a mensagem de substituição difere por commit;
- **`blame-ai`** usa um **pipeline literal de git puro**, porque o trailer a acrescentar é fixo do início ao fim — nenhum dado do chamador precisa entrar na string do `--exec`.

As três resolvem a mesma ameaça de injeção de shell, escolhendo a técnica pelo que **varia** por commit.

### 5.3 O `interpret-trailers` evitou duplicar lógica que o `strip` precisou escrever

O `strip` teve de decidir manualmente onde cortar o bloco de trailers e como reconstruir a mensagem sem ele. O `blame` não precisa do equivalente para **adicionar**: o `git interpret-trailers` já sabe abrir parágrafo ou completar bloco existente sozinho — medido antes de confiar, e mais simples do que teria sido reimplementar o posicionamento à mão.

---

## 6. Testes

**100% de statements** na parte de fabricação do `internal/aitrailers`. **Em `cmd/blame_ai.go`, o `runBlameAI` mede 95,2%** — o que sobra são ramos de erro de escrita, não do caminho que fabrica o trailer.

| # | Cenário | Esperado |
|---|---|---|
| 1 | `HEAD`, sem trailer nenhum ainda | Pergunta e adiciona — `RF-02`, `RF-04` |
| 2 | Cancelado | Nada muda |
| 3 | `--tool` desconhecida | Erro antes de tocar no git — `RF-01`, `RN-02` |
| 4 | Commit específico via `--commit` | Rebase e reencaixe rodam — `RF-02`, `RN-06` |
| 5 | Fora de um repositório | Erro |
| 6 | Falha do plano, do `Blame`, de leitura, de escrita | Propagadas |
| 7 | Argumento posicional sobrando | Erro |
| 8 | `blameHead` adiciona o trailer; restaura a quebra de linha final | `RN-05`, §5.1 |
| 9 | Commit vazio (`""`) e `"HEAD"` como alvo | Tratados como o mesmo caso |
| 10 | `--tool` desconhecida rejeitada no domínio | Antes de qualquer chamada ao git |
| 11 | Falha do log, do `interpret-trailers`, do amend | Propagadas |
| 12 | Commit específico: rebase e reencaixe, fallback de `HEAD` destacado, e **nunca** interpolar o SHA no `--exec` | `RN-06` |
| 13 | Falha em cada etapa do caminho de commit específico | Propagada |
| 14 | `PlanBlame`: `HEAD` e commit específico; `--tool` rejeitada antes do git; falha de `rev-parse` e de `log`; registro incompleto | `RF-01`, `RF-02` |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `e6da228` | `internal/aitrailers`: `Signature`, `Blame`, `blameHead`, `PlanBlame`, e a correção do bug da quebra de linha — `RF-01` a `RF-03`, `RF-06`, `RN-01` a `RN-07`, §5.1 a §5.3 |
| `15c7af8` | `gtr blame-ai`: o comando, `--commit`/`--tool`, prévia e confirmação — `RF-02`, `RF-04`, `RF-05` |

---

## 8. Follow-ups conhecidos

- **O ID numérico da conta bot do Copilot não é verificado contra a conta real** — `RN-03`, aceito porque não afeta a detecção.
- **`--commit` no commit raiz permanece sem tratamento**, mesma limitação herdada do `gtr author --commit` — `RF-05`.
- **Ferramentas fabricáveis limitadas a `claude` e `copilot`, mesma lista que o `list` reconhece** — cresce junto se o `detect.go` crescer.
