---
titulo: SRS — riff
data: 2026-08-18
status: entregue
comando: gtr riff
apelido: whatthecommit
pacotes:
  - cmd/riff.go
  - cmd/network.go
  - internal/riff
  - internal/web
commits:
  - 18b1059
  - b20eac1
  - 679dc00
fonte_externa: whatthecommit.com, API pública sem chave
---

# SRS — riff

- **Data:** 2026-08-18
- **Feature:** `cmd/riff` + `internal/riff` + `internal/web`
- **Status:** **entregue** — commits `18b1059` (o port), `b20eac1` (domínio) e `679dc00` (comando)
- **Fonte externa:** whatthecommit.com — API pública, sem chave

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr riff`, que imprime **uma mensagem de commit aleatória e boba**, vinda do whatthecommit.com — para compor com `git commit -m "$(gtr riff)"` ou só rir da sugestão.

### 1.2 Escopo

**Entregue:** uma chamada `GET` à API pública do whatthecommit.com, sem chave nem autenticação; corpo aparado, e vazio ou só espaço virando erro; prazo de 30s; declaração explícita da chamada de rede no `--help`; não exige repositório git.

**Fora de escopo:** o comando **não commita nada sozinho** — só imprime a mensagem, e quem quiser usá-la num commit compõe por fora, com `$(...)`. O [`gtr fire`](fire.md) reaproveita o `riff` internamente para isso.

### 1.3 O nome

`riff`, termo de guitarra, combinando com o `gtr`. Apelido `whatthecommit`, o nome do próprio site de origem — quem conhece o site reconhece na hora.

---

## 2. Descrição geral

```text
internal/riff/               domínio — não imprime nada
└── repo.go                  Repo.Random; Client: a interface mínima que precisa (só Get)

internal/web/                port de rede HTTP pura
├── client.go                Client: Get(ctx, url) (string, error)
└── http_client.go           HTTPClient: um GET sem retry

cmd/riff.go                  o comando, com o --help declarando a chamada de rede
cmd/network.go               networkDeadline (30s), compartilhado com pr list, doctor --online e fire
```

O `riff` é o **primeiro comando a abrir uma porta de rede nova** — o `internal/web`. O [`pr list`](pr-list.md) e o [`doctor --online`](doctor-online.md) reaproveitam o `internal/exec` que já existia, embrulhando o `gh`; o whatthecommit.com não tem cobertura nenhuma do `gh`, então precisou de HTTP de verdade.

---

## 3. Requisitos funcionais

### RF-01 — Buscar e imprimir uma mensagem aleatória

Um `GET` simples ao endpoint público do whatthecommit.com, sem chave nem autenticação, e a mensagem sai direto no `stdout`.

### RF-02 — Corpo vazio é erro, nunca mensagem em branco

O corpo é aparado de espaço e quebra de linha; se o resultado for vazio, é erro nomeando a origem — nunca uma linha em branco impressa como se fosse a mensagem.

### RF-03 — Declarar a chamada de rede no `--help`

O texto longo do comando avisa explicitamente: **"FAZ CHAMADA DE REDE — direto por HTTP, sem passar pelo gh"**, e nomeia que é o único caminho em que o `gtr` fala com a internet sem intermediário.

### RF-04 — Prazo de rede

A chamada roda sob o `networkDeadline` de 30s, compartilhado com `pr list`, `doctor --online` e `fire`. Sem prazo, um servidor calado penduraria o comando para sempre; é a única classe de operação do `gtr` que não termina sozinha.

### RF-05 — Não exige repositório git

O `riff` não toca em git nenhum: funciona em qualquer diretório, dentro ou fora de um repositório.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **`internal/riff` declara só o método do `Client` que precisa** (`Get`), e não importa `internal/web` — desacoplamento pela interface no ponto de consumo, não pela implementação. |
| **RN-02** | **Resposta HTTP fora de 200 é erro nomeando a URL e o status**, nunca corpo interpretado às cegas quando o servidor devolve algo diferente do esperado. |
| **RN-03** | **O `riff` não exige estar dentro de um repositório git.** É o segundo comando com essa isenção, depois do `doctor` — por razão diferente: o `doctor` diagnostica antes de haver contexto, o `riff` simplesmente não precisa de git nenhum para o que faz. |

### 4.1 Regras herdadas

- **Nenhum comando sai da máquina sem dizer** — `RN-02` da [SRS — a conexão](doctor-online.md), que é a promessa reescrita do projeto: de "não faz requisição" para "não faz sem declarar".
- **Toda chamada de rede tem prazo** — `RN-03` da [SRS — a conexão](doctor-online.md).

---

## 5. Achados de implementação

### 5.1 O `internal/web` nasceu com o `riff`, não com o `pr list`

O `pr list` e o `doctor --online` já falavam com a rede — mas por dentro do `gh`, via `internal/exec`, sem porta nova nenhuma. O `riff` é quem abriu a primeira porta de HTTP puro do projeto, porque não existe comando de `gh` que fale com o whatthecommit.com.

### 5.2 O `HTTPClient` não decide prazo nem política de retry

`HTTPClient.Get` é só um `GET`: sem cabeçalho especial, sem retry. Quem decide o prazo é sempre o `context` que chega de fora — a mesma disciplina de não esconder política de rede dentro do client que o `internal/git` já segue para chamadas de processo.

O `internal/web` nasceu em commit à parte, **antes** do domínio que o consome, espelhando de propósito a forma do `internal/exec`: uma interface estreita, uma implementação de verdade, e um fake de respostas em ordem no seu próprio `webtest`. É o **terceiro port** do projeto, depois de `git` e `exec`.

### 5.3 A interface mínima evitou dependência do pacote inteiro

`riff.Client` declara só `Get` — não importa o `internal/web.Client`. O `internal/riff` nunca sabe que o `HTTPClient` existe; quem faz a ligação é o `cmd`, injetando a implementação de verdade. Mesmo padrão de porta que o `git.Runner` já usa para o restante do projeto.

---

## 6. Testes

**100% de statements** em `internal/riff` e em `cmd/riff.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | `GET` bem-sucedido | Mensagem aparada, sem erro — `RF-01`, `RF-02` |
| 2 | Falha do client | Erro propagado |
| 3 | Corpo vazio ou só espaço | Erro, nunca mensagem em branco — `RF-02` |
| 4 | `--help` | Menciona a chamada de rede — `RF-03` |
| 5 | `whatthecommit` | Roda o mesmo comando que `riff` — §1.3 |
| 6 | Argumento posicional sobrando | Erro |
| 7 | Falha de escrita no stdout | Erro |
| 8 | Contexto cancelado | `Random` recusa |
| 9 | Apenas uma chamada, para a URL exata do endpoint | `RF-01` |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `18b1059` | Nasce o `internal/web`: a interface `Client`, o `HTTPClient` e o `webtest` — §5.2 |
| `b20eac1` | `internal/riff.Repo.Random`: o `GET`, corpo aparado, corpo vazio como erro — `RF-01`, `RF-02`, `RN-01`, `RN-02` |
| `679dc00` | `gtr riff`: o comando, o prazo de rede, a declaração no `--help`, sem exigir repositório. O ponto de injeção ganha um `web.Client` — `RF-03` a `RF-05`, `RN-03` |

---

## 8. Follow-ups conhecidos

- **Sem cache nem retry.** Cada chamada é uma requisição nova; aceitável porque é comando manual, não caminho quente.
- **O [`gtr fire`](fire.md) reaproveita o `riff` internamente**, com o mesmo `networkDeadline`, mas caindo para uma mensagem fixa se a rede falhar.
