---
titulo: SRS — changelog
data: 2026-08-20
status: entregue
comando: gtr changelog
pacotes:
  - cmd/changelog.go
  - cmd/changelog_table.go
  - cmd/changelog_document.go
  - cmd/changelog_record.go
  - internal/changelog
  - internal/format
commits:
  - 354c95e
  - 64c05ee
  - 991247d
  - a732039
  - cbd51fb
  - bf7c64b
fonte_externa: nenhuma
---

# SRS — changelog

- **Data:** 2026-08-20
- **Feature:** `cmd/changelog` + `internal/changelog`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr changelog`, que gera um `CHANGELOG.md` a partir do histórico, agrupado pelo tipo do **Conventional Commits** — a única saída do `gtr` pensada para ser lida **fora** da CLI.

### 1.2 Escopo

**Entregue:** classificação de cada commit pelos onze tipos padrão do Conventional Commits, com `Miscellaneous` para o que não bate na assinatura; Markdown de verdade no `--format text`, uma seção por tipo, só as que têm commit; `csv`/`tsv`/`json` como uma linha por commit; `--since`/`--until` filtrando o período, sem limite quando ausentes; `--output` sem extensão gerando `.md`, não `.txt`.

**Fora de escopo:** o rodapé `BREAKING CHANGE:` no corpo do commit — só o `!` na assinatura é lido; esconder tipos (`chore`, `test`, `ci`, `style`) por configuração — depende do arquivo de configuração da **ADR-004**, que ainda não existe; qualquer noção de release ou tag — tudo cai em `[Unreleased]`.

### 1.3 O nome

`changelog`, o termo padrão do ecossistema Conventional Commits — sem apelido, é utilitário, não piada.

---

## 2. Descrição geral

```text
internal/changelog/          domínio — não imprime nada
├── type.go                  Type, os onze + Miscellaneous, known()
├── entry.go                 Entry, subjectPattern (regex do Conventional Commits), parse()
└── repo.go                  Entries(ctx, since, until), Ensure

internal/ui/changelog.go     DescribeSection: o rótulo em inglês de cada seção

cmd/changelog.go             o comando, --since/--until, validateChangelogPeriod
cmd/changelog_table.go       a tabela; textExtension() = ".md"; monta o Markdown
cmd/changelog_document.go    changelogDocument
cmd/changelog_record.go      changelogRecord

internal/format/format.go    PathWithExtension, de que o Path virou wrapper fino
```

---

## 3. Requisitos funcionais

### RF-01 — Classificar cada commit pelo Conventional Commits

Um regex fiel à gramática 1.0.0 (`tipo(escopo)!: assunto`, escopo e `!` opcionais) separa tipo, escopo, marca de *breaking change* e assunto. O que não bate na assinatura — ou bate com um tipo fora dos onze reconhecidos — vira `Miscellaneous` com o assunto **cru**, nunca descartado.

### RF-02 — Onze tipos reconhecidos, mais Miscellaneous

`feat`, `fix`, `perf`, `refactor`, `docs`, `test`, `build`, `ci`, `chore`, `style`, `revert` — os onze do padrão —, com `Miscellaneous` sempre por último.

### RF-03 — Markdown de verdade no `--format text`

Uma seção `## [Unreleased]`, com subseção `### <nome em inglês>` por tipo, **só as que têm commit**, na ordem de `changelog.Types`. Cada linha: ``- [⚠ BREAKING: ][**escopo:** ]assunto (`hash curto`)``.

### RF-04 — `csv`/`tsv`/`json`: uma linha por commit

Diferente do texto, que é agrupado, os formatos estruturados emitem **um registro por commit**, com `hash`, `tipo`, `escopo`, `breaking` e `assunto`. O `tipo` é sempre o **token** (`feat`, `fix`…), nunca traduzido — só os cabeçalhos de seção do Markdown recebem o nome em inglês.

### RF-05 — `--since`/`--until` filtram o período

`AAAA-MM-DD`, validados antes de tocar no git. Ausentes, cada lado fica **sem limite** — diferente do [`profile`](profile.md), onde a ausência das duas assume hoje: aqui é changelog, não métrica diária, então nenhuma flag significa o histórico inteiro.

### RF-06 — `--output` sem extensão gera `.md`

O `--format text` do `changelog` é Markdown de verdade, não texto de tela — `--output` sem extensão tem de produzir `CHANGELOG.md`, nunca `CHANGELOG.txt`.

### RF-07 — Vazio nomeia o lado do período que foi aplicado

Sem commit no intervalo, a lista vem vazia e a mensagem diz **qual** limite estava valendo: `Nenhum commit entre <since> e <until>.`, `Nenhum commit desde <since>.`, `Nenhum commit até <until>.` — e, só sem nenhuma das duas flags, `Nenhum commit no histórico deste repositório.`. Ver §5.4.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Nenhum commit some.** O que não bate na assinatura do Conventional Commits vira `Miscellaneous` com o assunto cru — nunca é descartado do relatório. |
| **RN-02** | **`Miscellaneous` não é uma assinatura que um commit escreve, é o que sobra de quem não escreveu nenhuma** — por isso fica de fora da lista de tipos válidos que o `known()` reconhece, mesmo aparecendo em `changelog.Types` para efeito de ordenação de seção. |
| **RN-03** | **Tipo é sempre token em dados estruturados, nunca traduzido.** `csv`, `tsv` e `json` carregam `feat`, `fix` e os demais crus — só o Markdown, pensado para leitura humana fora da CLI, ganha `Features`, `Bug Fixes` e companhia. |
| **RN-04** | **O inglês nos cabeçalhos de seção é exceção deliberada ao vocabulário em português do restante do `gtr`.** O changelog é o único artefato pensado para ser lido fora da CLI, e esses são os nomes que a própria convenção já usa. |
| **RN-05** | **`since` e `until` viram meia-noite e `23:59:59` explícitas antes de chegar ao git**, nunca a data nua — mesma disciplina do [`profile`](profile.md), pela mesma razão medida lá. |
| **RN-06** | **Ausência de `--since`/`--until` não assume "hoje", ao contrário do `profile`.** Um changelog sem filtro é o histórico inteiro — são duas perguntas diferentes ("o que mudou sempre" contra "o que mudou hoje") que compartilham o mecanismo de datas. |
| **RN-07** | **Seção vazia não aparece no Markdown.** Só tipos com pelo menos um commit no período geram `### <seção>` — um changelog com onze cabeçalhos e dez vazios seria ruído. |
| **RN-08** | **`--output` sem extensão respeita o formato de fato escrito, não o nome genérico do formato escolhido.** `--format text` aqui é Markdown; a extensão padrão tem de refletir isso — ver §5.3. |
| **RN-09** | **Mensagem de vazio não pode confundir "não há commit" com "não há commit no período".** Num repositório com centenas de commits, todos fora do intervalo pedido, dizer que o repositório está vazio é falso — ver §5.4. |

---

## 5. Achados de implementação

### 5.1 O regex segue a gramática formal, não uma aproximação

`^([a-zA-Z]+)(\(([^()]+)\))?(!)?: (.+)$` — tipo, escopo entre parênteses **sem** parênteses aninhados, `!` opcional, dois-pontos e espaço obrigatórios antes do assunto. Testado contra os onze tipos, com e sem escopo, com e sem `!`, e com `!` **e** escopo juntos (`feat(auth)!: ...`).

### 5.2 `BREAKING CHANGE:` no corpo ficou de fora por falta de espécime real

A especificação permite marcar *breaking change* de duas formas: o `!` na assinatura, que é lido, e um rodapé `BREAKING CHANGE:` no corpo, que não é. A segunda exigiria o corpo inteiro do commit (`%B`), não só o assunto (`%s`) — e **não há nenhum commit real no histórico do `gitarias` para medir o parser contra ele**. Ficou de fora por decisão de não implementar às cegas, e está nos follow-ups, não como bug.

### 5.3 `Format.Path` assumia que o nome do formato é a extensão

`--output CHANGELOG`, sem extensão, produzia `CHANGELOG.txt`, porque o `Format.Path` sempre derivava a extensão do próprio formato escolhido (`text` → `.txt`) — válido para todo comando **menos** o `changelog`, cujo `--format text` é Markdown.

A correção criou `PathWithExtension(path, name, extension)`, com o `Path` virando wrapper fino sobre ela, e uma interface opcional no `cmd`:

```go
type ownTextExtension interface {
    textExtension() string
}
```

O `emit()`, compartilhado por todo comando, só consulta essa interface quando o formato escolhido é `text`; o `changelogTable` é a primeira e, até aqui, única tabela que a implementa. `csv`, `tsv` e `json` seguem com as extensões de sempre.

### 5.4 O vazio genérico mentia num repositório cheio

`Repo.Entries` devolve a mesma lista vazia tanto para repositório genuinamente sem commit quanto para um intervalo sem correspondência — e o texto tinha uma mensagem só para os dois casos. Num repositório com centenas de commits, todos fora do período pedido, a saída dizia `Nenhum commit no histórico deste repositório.`, que é falso.

Agora o `changelogTable` carrega `since` e `until`, e a mensagem nomeia o lado do período que estava valendo. Sem nenhuma das duas flags, a mensagem original continua igual.

Os dois defeitos desta seção e da §5.3 apareceram rodando o binário contra um repositório real, de centenas de commits — não contra o fake, que respondia exatamente o que se esperava dele.

### 5.5 Os onze tipos sempre aparecem, sem esconder nenhum

Não existe ainda arquivo de configuração para escolher quais tipos mostrar por repositório (**ADR-004**). Esconder `chore`, `test`, `ci` ou `style` por padrão seria decisão de experiência que este comando **não** deveria tomar sozinho — fica para quando a configuração existir.

---

## 6. Testes

**100% de statements** em `internal/changelog` e nos arquivos de `cmd/changelog*.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Um commit de cada um dos onze tipos | Classificados certo — `RF-01`, `RF-02` |
| 2 | Escopo, `!`, e os dois juntos | Parseados certo — `RF-01`, §5.1 |
| 3 | Assunto que não bate na assinatura | `Miscellaneous`, assunto cru preservado — `RN-01` |
| 4 | Seção sem nenhum commit | Não aparece no Markdown — `RN-07` |
| 5 | Hash curto na linha do Markdown | 7 caracteres — `RF-03` |
| 6 | `--format csv` e `--format json` | Uma linha por commit, tipo como token — `RF-04`, `RN-03` |
| 7 | json envelopado na chave `entries` | `RF-04` |
| 8 | `--since`, `--until`, os dois juntos | Filtram certo, sem limite no lado ausente — `RF-05`, `RN-06` |
| 9 | `--since`/`--until` fora do formato | Erro antes de tocar no git |
| 10 | Fora de um repositório | Erro do `Ensure` |
| 11 | Falha do `git log` | Erro propagado |
| 12 | `--format` desconhecido, e flag fora do seu formato | Erro antes de tocar no git |
| 13 | Repositório sem nenhum commit, e período sem correspondência | Mensagens diferentes, nomeando o lado aplicado — `RF-07`, `RN-09` |
| 14 | Nenhuma linha termina em espaço | Disciplina compartilhada com o resto do projeto |
| 15 | Falha de escrita no stdout | Erro |
| 16 | `--output CHANGELOG` sem extensão, e `--output relatorio.csv` com uma já dada | `.md` no primeiro caso, `.csv` preservado no segundo — `RF-06`, `RN-08`, §5.3 |
| 17 | Ordem: mais recente primeiro; linha malformada do log ignorada | `RF-01` |
| 18 | Argumento posicional sobrando | Erro |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `354c95e` | `internal/changelog.Repo`: classificação pelos onze tipos, `Miscellaneous` como reserva — `RF-01`, `RF-02`, `RN-01`, `RN-02`, §5.1 |
| `64c05ee` | `gtr changelog`: o comando, Markdown agrupado, csv/tsv/json linha a linha, tipo como token — `RF-03`, `RF-04`, `RN-03`, `RN-04`, `RN-07` |
| `991247d` | `Entries` ganha `since`/`until`, sem limite quando ausentes — `RF-05`, `RN-05`, `RN-06` |
| `a732039` | `--since`/`--until` no comando, validados como no `profile` — `RF-05` |
| `cbd51fb` | `--output` sem extensão gera `.md`: nasce o `PathWithExtension` e a interface `ownTextExtension` — `RF-06`, `RN-08`, §5.3 |
| `bf7c64b` | A mensagem de vazio nomeia o lado do período aplicado — `RF-07`, `RN-09`, §5.4 |

---

## 8. Follow-ups conhecidos

- **`BREAKING CHANGE:` no corpo do commit não é lido** — §5.2. Só o `!` na assinatura conta. Precisa de um commit real no histórico para medir o parser antes de implementar.
- **Nenhum tipo pode ser escondido por configuração.** Depende do arquivo de configuração da **ADR-004**, que ainda não existe.
- **Sem noção de release ou tag.** Tudo cai em `[Unreleased]`; o repositório não tem nenhuma tag para medir a divisão por versão.
