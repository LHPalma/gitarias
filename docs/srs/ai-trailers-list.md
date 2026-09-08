---
titulo: SRS — ai-trailers list
data: 2026-08-20
status: entregue
comando: gtr ai-trailers list
apelido: synth
pacotes:
  - cmd/ai_trailers.go
  - cmd/ai_trailers_table.go
  - cmd/ai_trailers_document.go
  - cmd/ai_trailers_record.go
  - internal/aitrailers
commits:
  - 2caec35
  - 72d43e9
fonte_externa: nenhuma
---

# SRS — ai-trailers list

- **Data:** 2026-08-20
- **Feature:** `cmd/ai_trailers.go` (subcomando `list`) + `internal/aitrailers`
- **Status:** **entregue** — commits `2caec35` (domínio) e `72d43e9` (comando)
- **Fonte externa:** nenhuma
- **Relacionados:** [SRS — ai-trailers strip](ai-trailers-strip.md), que remove o que aqui se detecta, e [SRS — blame-ai](blame-ai.md), que fabrica o oposto — os três dividem o `internal/aitrailers`

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr ai-trailers list`, que **detecta** commits cujo trailer indica autoria ou coautoria de IA — Claude Code, GitHub Copilot — sem alterar nada.

### 1.2 Escopo

**Entregue:** leitura de trailers via `%(trailers:only,unfold)`; reconhecimento de Claude Code (`Claude-Session` com qualquer valor, ou `Co-Authored-By` com o e-mail da ferramenta) e de GitHub Copilot (`Co-Authored-By` com nome contendo "copilot" **e** e-mail `@users.noreply.github.com`); filtro `--since`/`--until`; os quatro formatos da **ADR-004**.

**Fora de escopo:** remover o trailer — isso é o [`strip`](ai-trailers-strip.md). Acrescentar — isso é o [`blame-ai`](blame-ai.md). O `list` é **só leitura**, de propósito: as outras duas mexem em histórico e caem sob a **ADR-008**.

### 1.3 O nome

`ai-trailers`, com hífen — o nome sem prefixo, `trailers`, fica reservado para um leitor genérico de trailers que ainda não existe. Apelido `synth`: sintetizador, no mesmo campo semântico sonoro do `gtr`, brincando com "autoria sintética".

Todo identificador e arquivo deste comando em `cmd` carrega o prefixo `ai`/`AI`, porque `cmd` é um namespace único e plano onde um simples `trailers` colidiria; o `internal/aitrailers` não precisa do prefixo nos próprios tipos exportados (`Trailer`, `Finding`, `Repo`), por já ser pacote isolado.

---

## 2. Descrição geral

```text
internal/aitrailers/         domínio — não imprime nada
├── trailer.go               Trailer, Finding
├── detect.go                tool(), coAuthorTool(), claudeEmail
└── repo.go                  List, Ensure, parseRecord, empty

cmd/ai_trailers.go           o comando pai e o list, aiTrailersPeriodOptions
cmd/ai_trailers_table.go     a tabela; toolNames, aiRawTrailers
cmd/ai_trailers_document.go  aiTrailersDocument
cmd/ai_trailers_record.go    aiFindingRecord, aiTrailerRecord
```

---

## 3. Requisitos funcionais

### RF-01 — Listar commits com trailer de IA reconhecido

Do mais recente para o mais antigo, no histórico do `HEAD` atual.

### RF-02 — Reconhecer Claude Code

`Claude-Session` com **qualquer** valor, ou `Co-Authored-By` cujo e-mail seja exatamente o do Claude Code.

### RF-03 — Reconhecer GitHub Copilot

`Co-Authored-By` cujo nome contenha "copilot", sem diferenciar maiúscula, **e** cujo e-mail termine em `@users.noreply.github.com` — as **duas** condições juntas, nunca uma só.

### RF-04 — Só o trailer que bate aparece

Um commit com `Co-Authored-By` de uma pessoa de verdade ao lado de um de IA mostra, no achado, **só o de IA** — o humano nunca aparece na lista de trailers do `Finding`.

### RF-05 — `--since`/`--until` filtram o período

Sem limite quando ausentes, igual ao [`changelog`](changelog.md) — nenhuma flag significa o histórico inteiro.

### RF-06 — Repositório sem histórico

Lista vazia, não erro.

### RF-07 — Emitir em formato estruturado

`--format text|csv|tsv|json`. No texto: hash curto, ferramenta ou ferramentas, e assunto por linha. Em csv, tsv e json: cada **trailer bruto** — chave, valor, ferramenta — do achado.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O e-mail `@users.noreply.github.com` sozinho nunca basta para reconhecer o Copilot.** Precisa também que o nome contenha "copilot" — sem essa segunda condição, qualquer humano escondendo o e-mail real atrás do endereço noreply do GitHub, que é a opção padrão de privacidade da plataforma, seria confundido com o bot. |
| **RN-02** | **Um rodapé solto como "Generated with Claude Code" não é trailer de verdade, e sai de graça.** O `%(trailers:only)` já não o lista, porque o próprio `git interpret-trailers` não reconhece aquele texto como bloco de trailer — não precisou de regra manual para excluí-lo. |
| **RN-03** | **Ferramenta duplicada num mesmo commit aparece uma vez só na coluna de tela**, mas cada trailer bruto continua separado em csv, tsv e json. Um commit com `Claude-Session` e `Co-Authored-By` do Claude ao mesmo tempo mostra "Claude Code" uma vez no texto, e os dois trailers nos dados estruturados. |
| **RN-04** | **A chave do trailer casa sem diferenciar maiúscula**, mas o valor só casa pelo que a regra exige: e-mail comparado em minúsculas, nome comparado contendo "copilot" em minúsculas — nunca comparação exata do valor inteiro. |
| **RN-05** | **O `list` é só leitura, mesmo achando trailers.** Remover ou acrescentar são comandos deliberadamente separados, porque caem sob a disciplina de reescrita de histórico da **ADR-008** — o `list` nunca precisa de confirmação porque nunca muda nada. |

### 4.1 Regras herdadas

- **O domínio não escreve na tela** — `RN-11` da [SRS — branches](branches.md). O `List` devolve `[]Finding`; tabela, rótulos e agrupamento moram inteiros no `cmd`.
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md), e é o que dá a `RN-02` de graça.

---

## 5. Achados de implementação

### 5.1 A forma curta do placeholder é aposta de compatibilidade, não certeza

A forma **curta** do `%(trailers:only,unfold)`, sem `=true`, foi escolhida por ser a mais antiga entre as duas que o `pretty-format` aceita — aposta de que sobrevive até a mínima de **2.22** que o [`doctor`](doctor.md) exige.

**Não foi medida contra um 2.22 de verdade**, só contra o 2.43. Fica registrado como lacuna, não como certeza.

### 5.2 O rodapé de atribuição solto não precisou de filtro manual

Medido antes de escrever qualquer regra: um rodapé como "Generated with Claude Code", sem a forma `Chave: Valor` de um trailer de verdade, simplesmente não aparece em `%(trailers:only)` — o próprio git já concorda que aquilo não é trailer.

Uma exclusão que saiu de graça por causa da leitura via plumbing contratual, em vez de regex sobre o corpo inteiro.

### 5.3 Por que o Copilot exige duas condições, e o Claude só uma

O e-mail canônico do Claude Code é específico o bastante para bastar sozinho — nenhum humano tem esse e-mail por acaso. Já o `@users.noreply.github.com` é o e-mail de privacidade **padrão** que qualquer pessoa no GitHub pode escolher: bastar sozinho geraria falso positivo em massa. Exigir também "copilot" no nome resolve isso sem tocar no domínio do e-mail.

---

## 6. Testes

**100% de statements** na parte de detecção e listagem do `internal/aitrailers` e nos arquivos de `cmd` do `list`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Claude Code via `Co-Authored-By` | `RF-02` |
| 2 | `Claude-Session` presente, independente do `Co-Authored-By` | `RF-02` |
| 3 | GitHub Copilot (nome + e-mail) | `RF-03` |
| 4 | Chave `Co-Authored-By` casada sem diferenciar maiúscula | `RN-04` |
| 5 | Coautor humano de verdade | Ignorado — `RN-01` |
| 6 | Humano usando e-mail noreply do GitHub | Ignorado — `RN-01` |
| 7 | Chave de trailer não relacionada | Ignorada |
| 8 | Valor de `Co-Authored-By` sem e-mail | Ignorado |
| 9 | Linha de trailer malformada (sem `: `) | Ignorada |
| 10 | Registro faltando um campo, ou faltando o campo de trailers | Ignorado, sem estourar |
| 11 | Commit sem nenhum trailer batendo | Fica de fora da lista |
| 12 | Só o trailer de IA aparece ao lado de um humano | `RF-04` |
| 13 | Ordem: mais recente primeiro | `RF-01` |
| 14 | `--since`, `--until`, os dois juntos | `RF-05` |
| 15 | Repositório sem nenhum commit | Lista vazia — `RF-06` |
| 16 | Falha do `rev-parse` ou do `git log` | Erro propagado |
| 17 | `--format csv` e `json`, envelope na chave `achados` | `RF-07` |
| 18 | `--since`/`--until` fora do formato | Erro antes do git |
| 19 | Fora de um repositório | Erro do `Ensure` |
| 20 | `--format` desconhecido, flag fora do formato | Erro |
| 21 | Nenhuma linha termina em espaço | Disciplina compartilhada |
| 22 | Falha de escrita, argumento sobrando | Erro |
| 23 | Mesma ferramenta duas vezes num commit | Uma só na tela, duas nos dados — `RN-03` |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `2caec35` | `internal/aitrailers.Repo.List`: leitura via `%(trailers:only,unfold)`, `tool` e `coAuthorTool` — `RF-01` a `RF-04`, `RN-01` a `RN-04`, §5.1 a §5.3 |
| `72d43e9` | `gtr ai-trailers list`, apelido `synth`, e o prefixo `ai`/`AI` no `cmd` — `RF-05` a `RF-07`, `RN-05`, §1.3 |

---

## 8. Follow-ups conhecidos

- **`%(trailers:only,unfold)` não confirmado contra um git 2.22 real** — §5.1.
- **O nome `trailers` está reservado, não implementado** — um leitor genérico de trailers, não só de IA, fica para quando for pedido.
- **A lista de assinaturas reconhecidas cresce por medição, e não é exaustiva.** Outros assistentes que deixem trailer diferente não são reconhecidos.
