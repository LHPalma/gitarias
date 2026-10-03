---
titulo: SRS — .gtr.yaml e gtr config
data: 2026-09-27
status: v1 (branches.protected/base + gtr config) implementado • translate e demais consumidores fora
comando: gtr config
pacotes:
  - cmd/config.go
  - cmd/config_table.go
  - cmd/config_record.go
  - internal/config
  - internal/branch
  - internal/ui
commits:
  - 67334bb
  - d24c2ff
fonte_externa: nenhuma
---

# SRS — .gtr.yaml e gtr config

- **Data:** 2026-09-27
- **Feature:** `internal/config` + `gtr config` + o primeiro consumidor real, `gtr branches`
- **Status:** **v1 implementado** — arquivo opcional em duas camadas, `branches.protected` e `branches.base` lidos de lá, `gtr config` mostra o efetivo e de onde veio. **`gtr config translate` e o vocabulário multilíngue seguem fora**, e **`changelog.types`/`aliases`** e **`ignored.separator`** continuam sem ler o arquivo — são consumidores futuros, não parte desta entrega
- **Fonte externa:** nenhuma — dois arquivos locais, lidos com a stdlib

---

## 1. Introdução

### 1.1 Propósito

A **ADR-004** já decidiu o desenho inteiro: um arquivo opcional, `.gtr.yaml`, em duas camadas (pessoal e do repositório), merge por chave, precedência `flag > repo > pessoal > padrão embutido`. Esta SRS especifica **a primeira fatia que vira código**: a leitura do arquivo, o primeiro consumidor real (`branches.protected` e `branches.base`, a pressão nº 1 do contexto da ADR) e o comando que audita o que foi lido.

**Por que só esta fatia.** A ADR também descreve `changelog.types`/`aliases`, `ignored.separator`, `gtr config translate` e o vocabulário multilíngue por `lang`. Nenhum desses tem consumidor nesta entrega — implementá-los agora seria escrever caminho que nenhum teste de uso real alcança, o mesmo defeito que o projeto já evita em outros lugares (ver ADR-005, "nada de entregar capacidade sem consumidor"). A própria ADR-004 já autoriza o corte: **"Começar só com `en` implementado, com o vocabulário já estruturado como mapa por língua. O `pt-BR` entra depois sem refatoração."** Aqui o corte é mais estrito ainda: nem o vocabulário por caminho entra — só os dois campos que `gtr branches` já promete usar.

### 1.2 Escopo

- **Descoberta e leitura das duas camadas** — `~/.config/gtr/config.yaml` (pessoal) e `<raiz-do-repo>/.gtr.yaml` (repo), merge por chave.
- **`branches.protected`** (lista, aceita `*` como coringa de branch, ex. `release/*`) e **`branches.base`** (string).
- **`gtr config`** — imprime a configuração efetiva e a origem de cada chave, nos quatro formatos da ADR-004.

**Fora de escopo (registrado para não ser redescoberto como esquecimento):**

- `gtr config translate` e qualquer vocabulário de chave que não seja inglês. Campo `lang:` diferente de `en` (ou ausente) é **erro claro**, não ignorado em silêncio — a mesma régua do `--separator` com `--format json` na ADR-004.
- `changelog.types`, `changelog.aliases`, `ignored.separator` — nenhum comando além do `branches` lê o arquivo nesta entrega.
- Criar o arquivo. `gtr config` só lê; não há `gtr config init`.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Camada** | Um dos dois arquivos que podem existir: pessoal (`~/.config/gtr/config.yaml`) ou do repositório (`.gtr.yaml` na raiz). Nenhuma delas é obrigatória. |
| **Chave efetiva** | O valor de uma configuração depois do merge: a do repo se ela a define, senão a pessoal, senão o padrão embutido. Merge é **por chave**, não por arquivo — um repo que só define `branches.base` não apaga o `branches.protected` do arquivo pessoal. |
| **Origem** | Qual camada produziu o valor efetivo de uma chave: `repo`, `personal` ou `default`. É o token que aparece no JSON; o rótulo em português (`arquivo do repo`, `arquivo pessoal`, `padrão embutido`) é só de apresentação. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/config/      lê e resolve o arquivo — não imprime nada, não conhece cmd
├── source.go          Source: repo, personal, default — token em internal/config, rótulo em internal/ui
├── file.go             o shape do YAML (só inglês) e o parse com a guarda de `lang`
├── paths.go            descoberta das duas camadas
├── branches.go          Branches: Protected/ProtectedSource, Base/BaseSource
└── config.go           Config, Entry, Entries(), Load(ctx, runner)

internal/branch/       primeiro consumidor
├── base.go              BaseFromConfig, ao lado de BaseFromFlag/OriginHead/Local
└── repo.go              ResolveBase e protected() passam a aceitar o que veio do arquivo

internal/ui/
└── config.go            DescribeConfigSource — rótulo em português, ao lado de DescribeSource

cmd/
├── config.go             o comando gtr config
├── config_table.go        header/rows/document/text
├── config_record.go       o registro do JSON
└── branches.go            carrega o config uma vez e repassa para ResolveBase/Merged/Tree
```

`internal/config` não conhece `internal/branch` nem `cmd` — é o mesmo tipo de porta que `internal/format`: só stdlib (mais o parser YAML) e nenhum domínio o importa de volta. Quem consome (`internal/branch`) recebe valores já resolvidos, nunca o `Config` inteiro — o domínio continua sem saber que um arquivo existe, só que alguém pode ter pedido para proteger `release/*`.

### 2.2 Descoberta das duas camadas

- **Pessoal:** `os.UserConfigDir()` (stdlib; respeita `XDG_CONFIG_HOME` e equivalentes por SO) + `gtr/config.yaml`. Falhar em descobrir o diretório (SO não suportado) não é erro — só significa que não há camada pessoal, mesma postura de arquivo ausente.
- **Repo:** `git rev-parse --show-toplevel` pelo mesmo mecanismo que o `worktrees` já usa. Falhar (fora de um repositório) também não é erro aqui — `internal/config` é usável fora de repositório, e só a camada de repo fica ausente.
- **Arquivo ausente em qualquer camada não é erro.** É o estado padrão e suportado (ADR-004: "a ausência total de configuração é estado válido"). YAML malformado, ou `lang` diferente de `en`/vazio, **são** erro — configuração que existe e não pôde ser lida não deve ser confundida com configuração ausente.

### 2.3 Merge por chave

Cada campo é resolvido independente dos outros:

```text
branches.protected:  repo define?  → usa o do repo, origem repo
                     senão pessoal define?  → usa o do pessoal, origem personal
                     senão → lista vazia (nenhuma adição), origem default

branches.base:       mesma cadeia, valor "" no lugar de lista vazia
```

Não há concatenação entre camadas para a mesma chave — só entre chaves diferentes de arquivos diferentes.

### 2.4 `branches.protected` é aditivo, nunca substitui a RN-03

A [SRS — branches](branches.md) já fixa: **"`main` e `master` são sempre protegidas, mesmo quando não são a base"** (RN-03). Um arquivo de configuração poderia, por omissão ou engano, listar só `develop` e `release/*` sem repetir `main`/`master` — e um mecanismo de *substituição* desprotegeria as duas em silêncio, exatamente o modo de falha que a RN-03 existe para impedir.

**Decisão desta entrega:** `branches.protected` no arquivo **acrescenta** nomes e padrões à proteção embutida; nunca a remove. O exemplo da própria ADR-004 (`[main, master, develop, "release/*"]`) continua válido — repetir `main`/`master` é redundante, não incorreto — mas deixar de repetir também não abre brecha.

---

## 3. Requisitos funcionais

### RF-01 — Ler e resolver as duas camadas

`internal/config.Load(ctx, runner)` devolve um `Config` com `branches.protected` e `branches.base` já resolvidos pela cadeia da §2.3, mais a origem de cada um.

### RF-02 — Recusar `lang` não suportado

Um arquivo com `lang:` diferente de `en` ou de vazio é erro explícito, nomeando o arquivo e a língua pedida. Nenhuma tradução é tentada nesta versão.

### RF-03 — `branches.protected` aceita coringa de branch

Um item do array pode conter `*` (ex. `release/*`), casado contra o nome de cada branch local com a mesma semântica de `path.Match`. Item sem `*` é comparado por igualdade exata.

### RF-04 — `gtr branches` consome o arquivo

- **Base:** com `--base` ausente, uma `branches.base` configurada é tentada **antes** da detecção por `origin/HEAD` e antes do par embutido `main`/`master` — mas **depois** da flag, que sempre vence. Branch configurada que não existe localmente é erro, mesma mensagem de quando é a flag que erra.
- **Protegidas:** a lista e os padrões configurados entram na proteção, por cima de `main`, `master`, a base e a branch atual — nunca no lugar delas.

### RF-05 — `gtr config` mostra o efetivo e a origem

Imprime, para cada chave conhecida (`branches.protected`, `branches.base`), o valor resolvido e a origem (`repo`, `personal` ou `default`, com rótulo em português no texto e no csv/tsv, token em inglês no json). Chave sem valor configurado mostra vazio, origem `default` — não é erro, nem lacuna a preencher.

### RF-06 — `gtr config` não exige repositório

Roda fora de um repositório git: a camada pessoal continua sendo lida; a de repo fica ausente, e as chaves caem no padrão embutido ou no que a pessoal definir.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Arquivo ausente, em qualquer camada, não é erro.** Erro é reservado para arquivo que existe e não pôde ser entendido — YAML malformado ou `lang` não suportado. |
| **RN-02** | **Merge é por chave, nunca por arquivo inteiro.** Um repo que define só `branches.base` não invalida um `branches.protected` do arquivo pessoal. |
| **RN-03** | **Configuração nunca enfraquece a proteção embutida de `main`/`master`** (herdada da RN-03 de [SRS — branches](branches.md)). `branches.protected` só acrescenta. |
| **RN-04** | **Origem é enum com token em inglês para máquina, rótulo em português para gente** — mesma régua que `BaseSource`/`ui.DescribeSource` já aplicam (`internal/ui` não é importado por domínio nenhum). |
| **RN-05** | **`internal/config` não conhece `internal/branch` nem `cmd`.** É porta, não domínio: devolve dados resolvidos: quem decide o que fazer com eles é quem chama. |

### 4.1 Regras herdadas

- **Nenhuma credencial no arquivo** — ADR-004, consequência direta de nunca ler nada além de nomes de branch nesta entrega.
- **`internal/config` não escreve na tela nem lê do teclado** — RN-11 de [SRS — branches](branches.md), aplicada à nova porta: só devolve `Config`.

---

## 5. Interface

```console
$ cat .gtr.yaml
branches:
  protected: [develop, "release/*"]
  base: develop

$ gtr config
CHAVE                 VALOR                   ORIGEM
branches.protected     develop, release/*      arquivo do repo
branches.base          develop                 arquivo do repo

$ gtr branches
Base: develop (informada via configuração)

Branches locais já mergeadas (0):
```

| Flag de `config` | Padrão | Efeito |
|---|---|---|
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` — ADR-004, máquina compartilhada |
| `--output <caminho>` | vazio | Grava em arquivo em vez do stdout |

---

## 6. Testes

Cobertura de 100% em `internal/config`, nos campos novos de `internal/branch` e em `cmd/config*.go`, seguindo a régua do resto do projeto.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Nenhum arquivo em nenhuma camada | Tudo no padrão embutido, `gtr config` mostra origem `default` nas duas chaves |
| 2 | Só arquivo pessoal define `branches.base` | Origem `personal`; `branches.protected` continua `default` |
| 3 | Repo e pessoal definem `branches.base` diferente | Vence o do repo — RN-02 |
| 4 | Repo define só `branches.protected`, pessoal só `branches.base` | As duas chaves aplicam, cada uma da sua camada — RN-02 |
| 5 | `lang: pt-BR` no arquivo | Erro nomeando o arquivo, nenhuma tentativa de tradução — RF-02 |
| 6 | YAML malformado | Erro propagado, não confundido com arquivo ausente — RN-01 |
| 7 | `branches.protected: [develop]`, sem repetir `main`/`master` | As duas continuam protegidas — RN-03 |
| 8 | `branches.protected: ["release/*"]`, branch `release/1.0` | Protegida; `release-old` (sem a barra) não é — RF-03 |
| 9 | `branches.base: nao-existe` | Erro claro, mesma mensagem de `--base` inexistente — RF-04 |
| 10 | `--base` e `branches.base` configurados ao mesmo tempo | Vence a flag, nenhuma leitura de config chega a importar — RF-04 |
| 11 | `gtr config` fora de repositório git | Não falha; camada de repo ausente, resto normal — RF-06 |
| 12 | `gtr config --format json` | Chaves em inglês (`key`, `value`, `source`), origem como token (`repo`/`personal`/`default`) |

---

## 7. Follow-ups conhecidos

- **`gtr config translate` e o vocabulário multilíngue** — fora desta entrega inteira, conforme §1.2. A ADR-004 já autoriza começar só em inglês.
- **`changelog.types`/`aliases` e `ignored.separator`** — mesmo arquivo, consumidores futuros, cada um sua própria mudança em `internal/changelog`/`internal/ignore` quando chegar a vez.
- **`gtr config` não tem `--init` nem `--write`.** Só leitura e diagnóstico nesta versão.
