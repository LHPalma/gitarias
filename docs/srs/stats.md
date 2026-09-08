---
titulo: SRS — stats
data: 2026-08-17
status: entregue
comando: gtr stats
pacotes:
  - cmd/stats.go
  - cmd/stats_table.go
  - cmd/stats_document.go
  - cmd/stats_record.go
  - internal/stats
commits:
  - 1303f85
  - 70f7667
fonte_externa: nenhuma
---

# SRS — stats

- **Data:** 2026-08-17
- **Feature:** `cmd/stats` + `internal/stats`
- **Status:** **entregue** — commits `1303f85` (domínio) e `70f7667` (comando)
- **Fonte externa:** nenhuma

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr stats`, que responde **quem commitou quanto neste repositório**: um ranking de autores pelo número de commits no histórico do `HEAD` atual.

### 1.2 Escopo

**Entregue:** contagem por autor a partir de `git log`, filtro `--author` repetível e somado por OR, repositório sem histórico tratado como lista vazia (não erro), os quatro formatos da **ADR-004**.

**Fora de escopo:** métricas de linhas alteradas — isso é o [`churn`](churn.md). Período customizado (`--since`/`--until`) — isso é o [`profile --commit-count`](profile.md). O `stats` só conta commits, sempre do início do histórico até o `HEAD`.

### 1.3 O nome

`stats`, genérico de propósito: é o primeiro comando de métricas do projeto, e reserva o nome mais específico (`churn`, `profile`) para quando a métrica for outra coisa que não "quantos commits por autor".

---

## 2. Descrição geral

```text
internal/stats/              domínio da contagem — não imprime nada
├── author.go                Author: nome, email, commits
└── repo.go                  Repo.ByAuthor, Repo.Ensure

cmd/stats.go                 o comando e a flag --author
cmd/stats_table.go           a tabela, nos quatro formatos
cmd/stats_document.go        statsDocument, o envelope json
cmd/stats_record.go          authorRecord
```

Sem dependência de outro domínio: `internal/stats` só fala com `internal/git`.

---

## 3. Requisitos funcionais

### RF-01 — Contar commits por autor

`gtr stats` conta, no histórico do `HEAD` atual, quantos commits cada autor tem, e lista do maior número de commits para o menor. A chave de agrupamento é o par exato (nome, e-mail) que aparece em cada commit — nomes diferentes com o mesmo e-mail, ou o mesmo nome com e-mails diferentes, contam como autores distintos.

### RF-02 — Filtrar por autor

`--author`, repetível. Casa por substring, exatamente como o `--author` do próprio `git log` já faz, e múltiplos valores somam por **OR**: qualquer commit que bata com qualquer um dos padrões entra na conta.

### RF-03 — Repositório sem histórico

Um repositório onde o `HEAD` ainda não aponta para nenhum commit devolve lista vazia, não erro. A saída textual é a frase `Nenhum commit encontrado.`, nunca uma tabela vazia.

### RF-04 — Emitir em formato estruturado

`--format text|csv|tsv|json`, com `--output`, `--separator` e `--no-header` — **ADR-004**. No texto, o total de autores abre a saída (`Autores (N):`). No csv/tsv o cabeçalho é `nome,email,commits`. No json a lista mora sob a chave `authors`, nunca como array solto — inclusive vazia, sem cair na frase em português do formato texto.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Empate em número de commits desempata por nome, em ordem de bytes — não alfabética "humana".** É comparação Go de `string` (`<`), e maiúscula pesa menos que minúscula no ASCII: `"NataLia"` vem antes de `"anteninha"`. Não há normalização de caixa nem de acentos antes do desempate. |
| **RN-02** | **Linha do `git log` sem o separador esperado é ignorada, nunca conta como autor.** Protege a contagem de uma mudança de formato na chamada do git que passasse despercebida — silenciosa em vez de estourar em índice fora do array. |
| **RN-03** | **Nada vai para o `stdout` antes do `Ensure` confirmar repositório válido.** Fora de um repositório, a falha sai só pelo caminho de erro, sem nenhuma escrita anterior. |

### 4.1 Regras herdadas

- **O domínio não escreve na tela** — `RN-11` da [SRS — branches](branches.md). `ByAuthor` devolve `[]Author`; a tradução para linha de texto, csv, tsv ou json mora inteira no `cmd`.
- **Nenhuma linha da saída de texto termina em espaço** — `RN-10` da [SRS — branches](branches.md), pela mesma disciplina de `columns()` e `trimmingWriter` do resto do projeto.
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md), e é o que decide o §5.1.

---

## 5. Achados de implementação

### 5.1 `git log --format`, não `git shortlog -sn`

A contagem é feita em Go sobre `git log --format=%an%x00%ae`, e não sobre o resumo que `git shortlog -sn` já calcula pronto. A escolha mantém a leitura em **plumbing contratual**: o formato do `shortlog` é pensado para olho humano, alinhado e ordenado à moda dele, sujeito a mudar de aparência entre versões — o tipo de saída que o projeto evita depender.

### 5.2 O separador é um byte que não pode aparecer em nome ou e-mail

Os dois campos por linha são unidos por `\x00` (NUL). Um nome com vírgula, tab ou até quebra de linha em tese ainda é nome válido de commit; NUL não é caractere que o git deixe entrar em nenhum dos dois campos, e por isso é o separador seguro — o mesmo raciocínio das outras leituras com `%x00` no projeto.

### 5.3 O desempate por nome herdou a ordem de bytes, não a alfabética

A agregação ordena com `sort.Slice` comparando `string` diretamente. Isso é ordem de **bytes ASCII**, onde toda letra maiúscula vale menos que toda minúscula — `TestByAuthorTiebreaksByName` fixa isso de propósito com um par (`anteninha`, `NataLia`) escolhido por cair nesse caso.

Não houve normalização de caixa porque a contagem não precisa de ordem alfabética "correta" para humano, só de uma ordem **determinística** entre nomes empatados — e byte a byte já entrega isso sem biblioteca extra.

---

## 6. Testes

**100% de statements** em `internal/stats` e nos arquivos de `cmd/stats*.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Histórico com 3 commits, 2 autores | Contagem certa, ordenada por commits, desempate por nome — `RF-01`, `RN-01` |
| 2 | `--author` com um valor | Filtra por substring, repassado ao `git log` — `RF-02` |
| 3 | `--author` repetido | Soma por OR, os dois autores aparecem — `RF-02` |
| 4 | Repositório sem nenhum commit | `Nenhum commit encontrado.`, sem erro — `RF-03` |
| 5 | `--format` desconhecido | Erro antes de tocar no git — `RF-04` |
| 6 | Fora de um repositório | Erro do `Ensure`, stdout vazio — `RN-03` |
| 7 | Falha do `git log` | Erro propagado |
| 8 | Nenhuma linha termina em espaço | §4.1 |
| 9 | Falha de escrita no stdout | Erro |
| 10 | `--format csv` e `--format json`, inclusive lista vazia em json | Cabeçalho `nome,email,commits`; `{"authors": []}`, nunca a frase em português — `RF-04` |
| 11 | json | Lista mora sob a chave `authors`, nunca array solto — `RF-04` |
| 12 | Argumento posicional sobrando | Erro |
| 13 | Linha do log sem separador | Ignorada, não conta como autor — `RN-02` |
| 14 | Contexto cancelado | `Ensure` e `ByAuthor` recusam — mesma disciplina de cancelamento do resto do domínio |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `1303f85` | `internal/stats.Repo.ByAuthor`: contagem, filtro por `--author`, repositório vazio como lista vazia — `RF-01` a `RF-03`, `RN-01`, `RN-02` |
| `70f7667` | `gtr stats`: o comando, a tabela nos quatro formatos, a mensagem para lista vazia — `RF-04`, `RN-03` |

---

## 8. Follow-ups conhecidos

- **Sem filtro de período.** Quem quer commits de um autor num intervalo de datas usa [`gtr profile --commit-count`](profile.md), que é escopado à identidade local configurada, não a um autor arbitrário do histórico. Um `--since`/`--until` no `stats` juntaria as duas coisas — não foi pedido até aqui.
- **Sem métrica de linhas alteradas.** O `stats` conta commits, não churn. Ver [SRS — churn](churn.md) para a métrica de tamanho de mudança.
