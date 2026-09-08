---
titulo: SRS — churn
data: 2026-08-17
status: entregue
comando: gtr churn
pacotes:
  - cmd/churn.go
  - cmd/churn_table.go
  - cmd/churn_document.go
  - cmd/churn_record.go
  - internal/churn
  - internal/ui
commits:
  - 89d1a76
  - 447e32e
fonte_externa: nenhuma
---

# SRS — churn

- **Data:** 2026-08-17
- **Feature:** `cmd/churn` + `internal/churn`
- **Status:** **entregue** — commits `89d1a76` (domínio) e `447e32e` (comando)
- **Fonte externa:** nenhuma

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr churn`, que responde **quais arquivos mais mudam neste repositório** — sinal de risco e acoplamento: arquivo que muda toda hora é arquivo que vale a pena olhar antes de mexer perto.

### 1.2 Escopo

**Entregue:** contagem de quantos commits tocaram cada caminho no histórico do `HEAD` atual, ranqueada do mais tocado para o menos, `--limit` (padrão 10, `0` traz todos), marcação de quem ainda está na árvore contra quem só sobrou no histórico, os quatro formatos da **ADR-004**.

**Fora de escopo:** contagem por autor — isso é o [`stats`](stats.md). Tamanho da mudança em linhas (`+`/`-` por arquivo) — o `churn` conta **toques**, em quantos commits o caminho aparece, não volume de diff. Filtro de período — sempre o histórico inteiro até o `HEAD`.

### 1.3 O nome

`churn`, o termo já consagrado para essa métrica (*code churn*) — quem chega de outra ferramenta de análise de repositório já conhece a palavra.

---

## 2. Descrição geral

```text
internal/churn/              domínio da contagem — não imprime nada
├── file.go                  File: caminho, commits, InTree
└── repo.go                  Repo.Busiest, Repo.Ensure, tracked (sondagem da árvore)

cmd/churn.go                 o comando e a flag --limit
cmd/churn_table.go           a tabela, nos quatro formatos
cmd/churn_document.go        churnDocument
cmd/churn_record.go          churnRecord

internal/ui/residence.go     DescribeResidence: "na árvore" / "só no histórico"
```

Sem dependência de outro domínio — `internal/churn` só fala com `internal/git`. A sondagem de árvore (`tracked`, via `ls-tree`) é a **mesma técnica** que o `internal/weight` usa, mas **duplicada**, não compartilhada por importação — ver §5.3.

---

## 3. Requisitos funcionais

### RF-01 — Contar toques por caminho, do mais tocado para o menos

`gtr churn` conta, no histórico do `HEAD` atual, em quantos commits cada caminho aparece, e lista do maior número de toques para o menor.

### RF-02 — Marcar onde o caminho está hoje

Cada linha diz se o caminho **ainda está na árvore atual** ou **só sobrou no histórico** — foi apagado, renomeado ou nunca chegou a esta branch. É o achado que distingue o `churn` de uma contagem cega: um arquivo apagado há muito tempo não é risco vivo, mesmo tendo sido muito tocado.

### RF-03 — `--limit`

Padrão `10`. `0` ou qualquer valor não positivo traz **todos** os caminhos, sem erro.

### RF-04 — Repositório sem histórico

Um repositório onde o `HEAD` ainda não aponta para nenhum commit devolve lista vazia, não erro. A saída textual é `Nenhum arquivo no histórico deste repositório.`.

### RF-05 — Emitir em formato estruturado

`--format text|csv|tsv|json`, com `--output`, `--separator` e `--no-header` — **ADR-004**.

O texto **não tem linha de cabeçalho de coluna** — cada linha já abre com a contagem flexionada (`1 commit` / `2 commits`, via `ui.Plural`), diferente do `stats`, que abre com `Autores (N):` e cabeçalho de colunas. No csv/tsv o cabeçalho é `caminho,commits,onde`. No json a lista mora sob a chave `files`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Commit de merge não contribui contagem.** É o comportamento padrão do próprio `git log --name-only` para commits de merge — nenhum caminho listado, a não ser que se peça `--diff-merges` explicitamente. Não foi pedido: um merge que traz 50 commits de uma branch não devia contar como 50 toques extras em cada arquivo que ela mudou. |
| **RN-02** | **`--no-renames` é fixo, não exposto como flag.** Sem ele, um caminho renomeado conta como um toque só ou como dois, dependendo do `diff.renames` configurado em cada máquina — a mesma contagem sairia diferente em duas máquinas com config diferente. Fixando, um renomeio **sempre** toca os dois caminhos, o antigo e o novo, do mesmo jeito em qualquer lugar. |
| **RN-03** | **Empate em número de toques desempata por nome de caminho**, em ordem de bytes — mesma disciplina de desempate do [`stats`](stats.md). |
| **RN-04** | **Falha ao listar a árvore atual (`ls-tree`) degrada, não derruba o comando.** Se a sondagem falhar, todo caminho é marcado como "só no histórico" e a contagem sai do mesmo jeito — mesmo raciocínio do `internal/weight`. O custo disso está em §5.5. |
| **RN-05** | **Nada vai para o `stdout` antes do `Ensure` confirmar repositório válido.** |

### 4.1 Regras herdadas

- **O domínio não escreve na tela** — `RN-11` da [SRS — branches](branches.md). `Busiest` devolve `[]File`; a tradução para linha de texto, csv, tsv ou json, e o rótulo "na árvore"/"só no histórico", moram no `cmd` e no `internal/ui`.
- **Nenhuma linha da saída de texto termina em espaço** — `RN-10` da [SRS — branches](branches.md).
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md), e é o que exige o `-z` do §5.2.

---

## 5. Achados de implementação

### 5.1 Uma passada de `git log`, contada em Go

A contagem lê `git log --format= --name-only --no-renames -z` **uma vez** e conta em Go, em vez de rodar `git diff-tree` por commit ou depender do resumo do `git shortlog`. Evita **N chamadas de processo** para um histórico de N commits — mesmo raciocínio de plumbing contratual do [`stats`](stats.md), aqui também com ganho de desempenho direto.

### 5.2 O `-z` é obrigatório pelo mesmo motivo do `weight`

Sem `-z`, tanto `git log --name-only` quanto `git ls-tree` citam e escapam à moda C caminhos com espaço, aspas ou byte fora do ASCII — e um caminho citado não bate mais com o mesmo caminho vindo de outra chamada sem citação. O `-z` desliga a citação e separa por NUL, que é também o separador da contagem em Go.

### 5.3 A sondagem de árvore foi duplicada de propósito, não compartilhada

`churn.tracked` e `weight.tracked` são **funções distintas**, cada uma no seu pacote, com o mesmo corpo: `ls-tree -r -z --name-only HEAD` e um mapa de presença. Não há importação entre `internal/churn` e `internal/weight` — domínios não se conhecem é a regra do projeto, e aqui ela se sustenta porque o que se repete é uma **técnica de oito linhas**, não conhecimento de negócio que possa divergir sem ninguém notar.

É a mesma distinção que o [`doctor`](doctor.md) faz ao justificar a importação consciente do `internal/branch`: duplicar aqui não arrisca drift porque não há regra nenhuma envolvida, só uma chamada de plumbing.

### 5.4 Merge sem arquivos é decisão herdada do git, não escolha do projeto

A ausência de contagem em merges não foi implementada — é o que o `git log --name-only` já faz por padrão. Fica registrado porque é fácil ler o código e supor que houve um filtro deliberado; na verdade não há filtro nenhum, é ausência de flag: `--diff-merges` nunca é passada.

### 5.5 Degradar em silêncio tem um custo que ainda não apareceu

Quando o `ls-tree` falha, o `tracked` devolve um mapa vazio e o comando segue — nenhum arquivo aparece como presente, mesmo que a árvore exista de verdade e a chamada tenha falhado por outro motivo, como um `GIT_DIR` inválido só para aquele comando. O resultado sai igual ao de um repositório onde tudo realmente saiu da árvore.

Aceito porque as únicas falhas observadas de `ls-tree` vêm de repositório sem `HEAD` — mesmo caso já coberto pelo caminho de histórico vazio —, mas é leitura incompleta: ver os follow-ups.

---

## 6. Testes

**100% de statements** em `internal/churn`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Histórico com toques repetidos em dois arquivos | Contagem certa, mais tocado primeiro, plural correto por linha — `RF-01` |
| 2 | Empate em número de toques | Desempata por nome — `RN-03` |
| 3 | Um caminho saiu da árvore, outro continua | "só no histórico" e "na árvore" nas linhas certas — `RF-02` |
| 4 | `--limit 1` com três caminhos de contagens diferentes | Corta os menores, mantém o maior — `RF-03` |
| 5 | `--limit 0` | Traz todos — `RF-03` |
| 6 | Repositório sem nenhum commit | Lista vazia, mensagem específica, sem erro — `RF-04` |
| 7 | Falha ao listar a árvore (`ls-tree`) | Comando não falha; nada sai marcado como presente — `RN-04` |
| 8 | `--format csv`, `--format json`, envelope na chave `files` | `RF-05` |
| 9 | Fora de um repositório | Erro do `Ensure`, stdout vazio — `RN-05` |
| 10 | Falha do `git log` | Erro propagado |
| 11 | `--format` desconhecido, e `--no-header` fora de csv/tsv | Erro antes de tocar no git |
| 12 | Nenhuma linha termina em espaço | §4.1 |
| 13 | Falha de escrita no stdout | Erro |
| 14 | Argumento posicional sobrando | Erro |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `89d1a76` | `internal/churn.Repo.Busiest`: contagem em uma passada, `--no-renames` fixo, sondagem de árvore reaproveitando a técnica do `internal/weight` — `RF-01`, `RF-02`, `RN-01` a `RN-04`, §5.1 a §5.4 |
| `447e32e` | `gtr churn`: o comando, `--limit`, a tabela nos quatro formatos, texto sem cabeçalho e com plural por linha — `RF-03` a `RF-05`, `RN-05` |

---

## 8. Follow-ups conhecidos

- **Sem filtro de período.** Igual ao `stats`, o `churn` sempre lê o histórico inteiro até o `HEAD`. Um `--since`/`--until` juntaria as duas ideias — não foi pedido até aqui.
- **Sem volume de mudança.** A métrica é toques, não linhas alteradas. `git log --numstat` daria `+`/`-` por arquivo, a um custo de parsing maior — não implementado.
- **A degradação silenciosa do `ls-tree` mistura dois motivos de falha** — §5.5. Hoje inofensivo porque a única falha observada é repositório sem `HEAD`, já coberto por outro caminho. Se o `ls-tree` puder falhar por outro motivo real, o achado "só no histórico" deixa de ser confiável, e isso merece voltar aqui.
