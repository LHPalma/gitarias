---
titulo: SRS — pr list
data: 2026-08-17
status: entregue
comando: gtr pr list
pacotes:
  - cmd/pr.go
  - internal/forge
  - internal/ui
commits:
  - d404ad3
  - c18c27e
fonte_externa: o GitHub, pelo gh
---

# SRS — pr list

- **Data:** 2026-08-17
- **Feature:** `cmd/pr` sobre o `internal/forge`
- **Status:** **entregue** — commits `d404ad3` (o comando) e `c18c27e` (o README)
- **Fonte externa:** **o GitHub**, pelo `gh`
- **Depende de:** [SRS — a conexão: doctor --online](doctor-online.md) — o port `internal/forge`, o prazo de rede e as três regras da rede são de lá

---

## 1. Introdução

### 1.1 Propósito

Especificar o **`gtr pr list`**: listar os pull requests abertos do repositório do diretório atual. É a primeira aplicação da entrada de rede, e o motivo pelo qual ela foi aberta.

### 1.2 O que este documento não repete

A decisão de embrulhar o `gh` em vez de falar HTTP, a mudança da promessa do projeto e o prazo de 30 segundos estão na [SRS — a conexão](doctor-online.md). Aqui só o que é do comando.

O resumo em uma linha, para quem chega direto nesta página: **quem fala com o GitHub é o `gh`, e por isso o `gtr` nunca vê o token.**

---

## 2. Descrição geral

```text
internal/forge/cli.go        PullRequests: gh pr list --json
cmd/pr.go                    o grupo pr e o subcomando list
internal/ui/pull_request.go  os rótulos de estado, em português
```

O comando entra pelo `Source`, não pelo `gh` — o `cmd` não sabe que existe uma CLI do outro lado. É isso que torna os testes de `cmd` capazes de encenar rede sem rede.

---

## 3. Requisitos funcionais

### RF-01 — Listar os pull requests abertos

```console
$ gtr pr list
  #7  aberto    feat-parser  muda o parser
  #8  rascunho  feat-outra   ainda cozinhando
```

Os campos vêm de `gh pr list --json`, que é **contratual**: o `gh` recusa nome de campo que não conhece, então um erro de digitação falha alto em vez de devolver silêncio.

**Rascunho vence o estado aberto** no rótulo, porque é o que muda o que dá para fazer: rascunho não se revisa nem se mergeia.

Vale os três formatos do projeto — texto, `csv` e `json` — com os dois vocabulários de sempre: rótulo em português na tela e no csv, token em inglês no json.

### RF-02 — Separar as duas recusas

| Situação | Erro |
|---|---|
| `gh` ausente | *"rode gtr setup para ver como instalar"* — `ErrUnavailable` |
| `gh` presente e recusado | a mensagem do próprio `gh`, **sem sugerir instalar** |

Há teste afirmando que a segunda **não** menciona o `setup`: mandar instalar o que já está lá não ajuda ninguém.

### RF-03 — Declarar a rede

O `--help` do `pr list` diz que o comando sai da máquina, e há teste afirmando. O requisito nasce na [SRS — a conexão](doctor-online.md); aqui ele é cumprido.

---

## 4. Regras de negócio que incidem

Este documento não estabelece regra própria. Cinco incidem sobre o comando:

| Regra | Onde aparece aqui |
|---|---|
| **O `gtr` nunca vê a credencial** — `RN-01` da [SRS — a conexão](doctor-online.md) | quem autentica é o `gh` |
| **Nenhum comando sai da máquina sem dizer** — `RN-02` de lá | o `--help` declara a saída |
| **Toda chamada de rede tem prazo** — `RN-03` de lá | a chamada roda sob o `networkDeadline` de 30s |
| **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md) | `gh pr list --json`; nada de parsear a saída humana |
| **O domínio não imprime** — `RN-11` da [SRS — branches](branches.md) | o `internal/forge` devolve dados; quem escreve é o `cmd` |

---

## 5. Achados de implementação

### 5.1 O `state` do `gh` não é o rótulo do usuário

O `gh` devolve `OPEN`, `MERGED`, `CLOSED` e um booleano `isDraft` à parte. Na tela isso vira **um** campo, com quatro valores, porque quem lê quer saber o que dá para fazer com o PR — e rascunho aberto não é a mesma coisa que aberto.

No `json` os dois campos continuam separados: token em inglês, sem inflexão. Os dois vocabulários de sempre.

### 5.2 O limite tem de chegar ao `gh`

`--limit` não filtra depois de receber; vai como argumento. Há teste inspecionando a chamada, porque o modo de falha silencioso aqui seria trazer 30 e mostrar 5.

---

## 6. O caminho feliz precisa de rede de verdade

Num ambiente cujo proxy de saída só serve um conjunto fixo de operações, o caminho feliz simplesmente não roda:

```console
$ gtr pr list
erro: o gh não conseguiu listar os pull requests: HTTP 403: This GraphQL query
is not enabled for this session…
```

Ali, só o caminho de erro é observável. Contra o GitHub de verdade os dois são: testado num repositório de terceiros com cinco PRs abertos, nos três formatos, com `--limit` respeitado:

```console
$ gtr pr list
  #537  aberto  fix/external-invoice-installment-leading-zero  fix: normaliza parcela...
  #535  aberto  dependabot/maven/software.amazon.awssdk-lambda-2.53.0  build(deps)...
  ...
```

E os caminhos da `RF-02` também: `gh` ausente, credencial recusada (401) e sem credencial nenhuma (código 4) — todos batendo com o desenhado, sem precisar mudar uma linha de código.

---

## 7. Testes

**100% de statements** em `cmd` e `internal/forge`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Dois PRs, um rascunho | Ambos, com o rótulo certo — `RF-01` |
| 2 | Os três formatos | Texto alinhado, csv contratual, json válido |
| 3 | Os campos pedidos ao `gh` | Todos os que o parser lê |
| 4 | `gh` ausente; `gh` recusado | Mensagens diferentes, e a segunda **não** cita o `setup` — `RF-02` |
| 5 | Resposta ilegível; lista vazia | Erro; e mensagem específica |
| 6 | `--limit 5` | O limite chega ao `gh` — §5.2 |
| 7 | Fora de repositório | Recusa antes de tocar a rede |
| 8 | `--no-header` com `--format json` | Recusado: flag setada de propósito e descartada calada é o pior modo de falha |
| 9 | `--help` | Declara a rede — `RF-03` |
| 10 | Nenhuma linha termina em espaço | `RN-10` da [SRS — branches](branches.md) |

Todos os cenários acima rodam com `gh` roteirizado. O comando também foi exercitado contra o `gh` e o GitHub reais — §6.

---

## 8. Rastreabilidade

| Commit | Entrega |
|---|---|
| `d404ad3` | O `gtr pr list` — `RF-01`, `RF-02` |
| `c18c27e` | O README, com a seção do comando — `RF-03` |

---

## 9. Follow-ups conhecidos

- **Cruzar o `pr list` com o [`branches --tree`](branches-tree.md).** É o que motivou a feature e ainda não existe: mostrar quais camadas da pilha já têm PR.
- **Não há filtro por estado nem por autor.** Só abertos, só `--limit`.
- **Escrita continua fora**, sob as três guardas da **ADR-008**. Ler o GitHub foi por onde a rede entrou; abrir e fechar PR é outra conversa.
