---
titulo: SRS — ignore
data: 2026-08-08
status: entregue
comando: gtr ignore list • gtr ignore add
pacotes:
  - cmd/ignore.go
  - internal/ignore
  - internal/format
commits:
  - 2504eaf
  - 292bd11
  - 3d0f6dc
  - cbbd1c8
  - f1f74a3
  - 7688f5b
  - 55a95f7
  - 8507e63
  - da58e83
  - cd91500
  - fe7bfaf
  - 717e10e
  - 54650a2
  - d65e42e
  - 10145fd
  - 184edca
  - ec0b353
  - 810d594
  - e5682f0
  - 78ab5c5
  - c17a6c7
fonte_externa: nenhuma
---

# SRS — ignore

- **Data:** 2026-08-08
- **Feature:** `cmd/ignore` + `internal/ignore` + `internal/format`
- **Status:** **entregue** — `list`, `--expand-dir` com completion por tab, e `add`
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar o comando `gtr ignore`, que responde **o que está sendo ignorado neste repositório e por qual regra**, e acrescenta padrões ao lugar certo. O git responde mal à primeira pergunta: exige compor dois comandos de plumbing, e o resultado não diz de onde veio cada regra.

Este documento cobre o comando inteiro: `list` (§3.1) e `add` (§3.2).

### 1.2 Escopo

- `gtr ignore list` — listagem com a regra que ignorou cada caminho.
- `--expand` — arquivo a arquivo em vez de diretório colapsado; `--expand-dir`, repetível, para expandir só o que interessa, com completion de shell dinâmica.
- `--format`, `--output`, `--separator` e `--no-header` — **ADR-004**.
- `gtr ignore add` — a primeira feature do `gtr` que escreve arquivo no repositório, com as seis armadilhas de gravação e a ordem de verificações da **ADR-005**.

**Fora de escopo:** `gtr ignore why <caminho>` — o `check-ignore -v` de um caminho só.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Ignorado** | Caminho **não rastreado** que casa com alguma regra de exclusão. Arquivo já rastreado que casa com uma regra **não aparece** — o git continua rastreando quem já estava dentro. |
| **Origem da regra** | O arquivo que contém o padrão. São três, e a distinção importa: `.gitignore` é convenção do time, `.git/info/exclude` vale só neste clone, `core.excludesFile` vale na máquina inteira. |
| **`--exclude-standard`** | Flag do `ls-files` que considera as três origens acima. **Não é só o `.gitignore`**. |
| **Colapso** (`--directory`) | Diretório inteiramente ignorado sai como **uma linha** em vez de um por arquivo. Sem isso, um repositório com `node_modules` cospe centenas de milhares de linhas. |
| **BOM** | Os três bytes `EF BB BF` no início de um arquivo UTF-8. Sem eles o Excel em português lê `relatório` como `relatÃ³rio`. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/git/                execução de git — compartilhado com os demais domínios

internal/format/             formatos de saída — só stdlib, não conhece ninguém
├── format.go                Format: Parse, Extension, Path, Delimited
├── separator.go             Separator: conjunto fechado e ParseSeparator
├── csv.go                   WriteByteOrderMark, WriteCSV
└── json.go                  WriteJSON

internal/ignore/             domínio — não imprime nada
├── entry.go                 Entry
├── destination.go           Destination: raiz, local ou global
├── add.go                   Add, e as seis armadilhas da gravação
├── add_result.go            AddResult: o que aconteceu, sem texto de tela
├── runner.go                a interface maior que este pacote precisa
└── repo.go                  Repo: Ensure, List, candidates, parse

cmd/ignore.go                front de texto
cmd/ignored_record.go        o registro com as tags json
cmd/output.go                writeAndClose
```

**O `git.Runner` não mudou.** A segunda metade da pipeline lê do **stdin**, e o port era `Run(args ...string)`. Havia a escolha entre método novo na interface e interface separada; a resposta foi nenhuma das duas. A capacidade foi para as implementações (`RunWithInput` no `CommandRunner` e no `gittest.Runner`) e **cada consumidor declara a interface que precisa** — `cmd` e `internal/ignore` têm a sua. Quem só usa `Run` continua com um fake de um método.

### 2.2 A pipeline da listagem

```console
$ git ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z \
    | git check-ignore -z --stdin -v
```

Quatro campos NUL-terminados por registro: **origem, linha, padrão, caminho**. Com `--expand`, o `--directory --no-empty-directory` sai.

Conferido contra o `git` 2.43.0 antes de qualquer linha de código ser escrita, incluindo o código de saída e o comportamento do `.git/info/exclude`.

### 2.3 Fluxo

1. Valida `--format`, `--no-header` e `--separator`. **Antes de tocar no git** — formato desconhecido não deve exigir um repositório para falhar.
2. Verifica que o diretório atual é um repositório git.
3. Lista os candidatos com `ls-files`. Nenhum candidato encerra aqui, sem perguntar as regras.
4. Alimenta o `check-ignore` pelo stdin e faz o parse dos registros.
5. Renderiza no formato pedido, para o `stdout` ou para o arquivo do `--output`.

---

## 3. Requisitos funcionais

### 3.1 `gtr ignore list`

#### RF-01 — Listar o que está sendo ignorado, com a regra que ignorou

`gtr ignore list` imprime cada caminho ignorado com **onde a regra mora** (arquivo e linha) e **qual é o padrão**, em colunas alinhadas sob os rótulos `CAMINHO`, `ORIGEM` e `PADRÃO`.

Sem nada ignorado: `Nada está sendo ignorado aqui.` — mensagem específica, não lista vazia.

**O rótulo não é enfeite.** Caminho e padrão podem carregar o mesmo texto — `node_modules/` aparece nos dois — e sem cabeçalho o leitor não tem como saber qual coluna é qual. `ORIGEM` e não `REGRA` porque o padrão *é* a regra; o que a coluna do meio carrega é de onde ela veio.

#### RF-02 — Colapsar o diretório ignorado, e expandir sob demanda

Diretório inteiramente ignorado sai como uma linha. `--expand` lista arquivo a arquivo.

A dica de rodapé que anuncia a flag **só aparece quando há diretório colapsado na lista**. Sem essa condição ela afirmava na tela algo que não estava lá — num repositório em que só um arquivo é ignorado, os dois modos produzem saída idêntica, e sugerir a flag é falso além de inútil.

#### RF-03 — Expandir diretórios colapsados um a um

`--expand-dir <caminho>`, repetível, expande só o diretório informado sem tocar nos demais. `--expand` é tudo ou nada: num repositório com `node_modules/`, `dist/` e `.venv/` ignorados, quem só quer conferir o `dist/` não precisa engolir os três.

**É flag repetível, não lista separada por vírgula.** Nenhuma flag do `gtr` faz split por vírgula, e caminho de arquivo pode legitimamente conter uma, o que forçaria escape numa convenção nova só para esta flag. `pflag.StringArray` resolve com o mecanismo que o Cobra já tem: cada ocorrência é um valor literal.

**O mecanismo são duas passadas de git**, porque `ls-files --directory` colapsa tudo ou nada e não existe flag para colapsar *menos um diretório*:

1. A listagem colapsada de sempre dá os candidatos-base.
2. Para cada caminho pedido que aparece **literalmente** entre os candidatos colapsados, uma segunda chamada — `ls-files --others --ignored --exclude-standard -z -- <caminho>` — troca aquela linha pelos arquivos individuais.
3. O `check-ignore` roda sobre a lista combinada, sem saber que veio de duas chamadas.

O custo é uma invocação de git a mais por diretório pedido, e é aceitável porque é opt-in: o caso comum não muda de custo.

**A fronteira que fica de fora, de propósito:** `--expand-dir` só aceita o caminho **exatamente como o git relatou o colapso**. Pedir um subdiretório de um diretório já colapsado — `--expand-dir vendor/lib` quando o git colapsou `vendor/` — não tem candidato correspondente, porque expandir metade de um colapso exigiria recompor o resto. A recusa nomeia os diretórios que **de fato** estão colapsados na listagem atual.

#### RF-04 — Completion de shell para `--expand-dir`

`cmd.RegisterFlagCompletionFunc` no valor da flag, dinâmica: roda a mesma consulta colapsada do passo 1 e sugere só os diretórios com `Entry.Directory == true`, filtrados pelo prefixo já digitado.

Duas decisões amarradas à mesma consulta:

- **Não é completion de filesystem.** `cobra.ShellCompDirectiveFilterDirs` sugeriria qualquer diretório do repositório, ignorado ou não — errado para uma flag que só aceita diretório colapsado. A dinâmica paga uma chamada de git a cada tab, e é o preço certo: sugerir o que a flag não vai aceitar é pior que a espera.
- **`ShellCompDirectiveNoFileComp`, sempre.** Sem isso, um prefixo sem diretório ignorado correspondente cairia para completion de arquivo do shell — voltando à mesma sugestão errada que a decisão anterior evita.

Fora de um repositório git, ou com o `check-ignore` falhando, a função devolve lista vazia com `ShellCompDirectiveError` — o erro nunca vaza como texto para dentro do protocolo de completion, que o interpretaria como sugestão.

**Só funciona para quem carregou o script de completion do `gtr`** (`source <(gtr completion bash)` ou equivalente). O binário ganha o comando `completion` de graça, do Cobra, mas ativá-lo no shell é passo de quem instala.

#### RF-05 — Emitir a listagem em formato estruturado

`--format text|csv|tsv|json`, padrão `text` — **ADR-004**.

- **csv/tsv:** as quatro colunas na ordem da pipeline, com linha de cabeçalho `origem,linha,padrão,caminho`. `--no-header` a remove; `--separator` aceita `,` `;` `|` e tabulação, escrita como caractere ou como `\t`.
- **json:** envelope sob a chave `ignored`, com chaves em inglês.
- O `tsv` é açúcar para csv com tabulação. Pedir `--format csv --separator '\t'` **sugere** o `tsv`, no `stderr`.

#### RF-06 — Gravar a listagem num caminho

`--output <caminho>` tira a listagem do `stdout`. **É caminho, não nome de arquivo:** relativo, absoluto, para dentro de subdiretório ou para fora do repositório.

Sem extensão no caminho, a do formato é acrescentada (`--format csv --output ignorados` → `ignorados.csv`). Com extensão, o caminho é respeitado como veio, **mesmo trocada**.

**Caminho terminado em separador é recusado.** Ele nomeia um diretório, e como não tem extensão a do formato seria colada nele: `--output saida/` gravaria um arquivo oculto `saida/.csv` e sairia com zero. A recusa nomeia um caminho pronto para usar no lugar.

O diretório de destino precisa existir — o `gtr` não cria árvore de diretórios por conta própria. O arquivo só é criado **depois** de o git responder, então listagem que falha não deixa arquivo vazio para trás.

### 3.2 `gtr ignore add`

#### RF-07 — Acrescentar um padrão ao destino certo

`gtr ignore add "*.log"`, sem flag de destino, grava em `<raiz>/.gitignore` — o padrão do comando é a convenção de time. `--local` grava em `.git/info/exclude`; `--global`, no `core.excludesFile`.

`--local` e `--global` **são exclusivos**: as duas nomeiam destino, e setar as duas é erro explícito, não a última vencendo calada.

#### RF-08 — `--global` recusa o `core.excludesFile` ausente, salvo com `--force`

Sem `--force`, `--global` recusa quando `core.excludesFile` não está configurado: escrever um arquivo que o git não vai ler é pior que não escrever nada. Com `--force`, configura o `core.excludesFile` **e** cria o arquivo, atendendo os dois passos na mesma chamada.

**O caminho padrão do `core.excludesFile` é resolvido como o git resolve, não como a stdlib.** `os.UserHomeDir()` no Windows **ignora** a variável `HOME` e só olha `%USERPROFILE%`; o Git for Windows, medido com `git check-ignore` e `HOME` sobrescrito, **honra** `$HOME` quando ela existe, e só cai para o equivalente de `%USERPROFILE%` quando não há. A implementação lê `HOME` direto antes de cair para a stdlib.

#### RF-09 — A ordem das verificações, antes e depois de gravar

São seis armadilhas, e a ordem entre elas é parte do requisito:

1. **Resolve o destino** (`RF-07`, `RF-08`).
2. **O padrão já está coberto** por uma regra existente? Então não grava, e diz qual regra já cobre.
3. **A linha já está lá**, literal? Então não grava de novo.
4. **Escapa o padrão** antes de gravar: `#` e `!` no início viram comentário e negação se não forem escapados, e espaço no fim é descartado pelo git se não for escapado.
5. **Garante a quebra de linha** antes de anexar — sem isso o append cola na última linha e corrompe a regra anterior. O arquivo é aberto uma vez só: a detecção lê do próprio handle em vez de reabrir o caminho.
6. **Depois de gravar**, confere se a regra pegou algum caminho de verdade — não pegar nada é **aviso, não erro**, porque regra preventiva é legítima — e se algum caminho pego já está rastreado.

**O comando nunca roda `git rm --cached` sozinho.** Avisar que um caminho já rastreado passou a casar com a regra é informação; tirar do índice por conta própria seria decidir por quem roda.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Separador que pode aparecer no dado não é separador.** As duas metades da pipeline rodam com `-z`. O git **cita** caminho com acento ou caractere especial — `relatório 2026.csv` sairia como `"relat\303\263rio 2026.csv"`. Mesma família de não tratar dado do usuário como inerte. |
| **RN-02** | **Código de saída 1 do `check-ignore` é resposta, não falha.** Ele sai com 1 quando nada casa. O `CommandRunner` usava `Output()`, que trata qualquer status não-zero como erro e descarta o código — "nada ignorado" seria indistinguível de repositório quebrado. Daí o `git.ExitError`, alcançável por `errors.As`. Código 128 continua sendo falha de verdade. |
| **RN-03** | **Os caminhos vão pelo `stdin`, nunca por `argv`.** Passá-los como argumento estoura o limite de argv exatamente no repositório grande, que é onde a feature importa. |
| **RN-04** | **Flag setada de propósito e descartada em silêncio é proibida.** `--separator` fora de `--format csv` e `--no-header` fora de `csv`/`tsv` são **erro**, não omissão. Quem setou jura que configurou, a ferramenta jura que não — é o pior modo de falha que existe. |
| **RN-05** | **O BOM é do destino, não do formato.** Sai só com `--output`. No `stdout` os três bytes grudam no primeiro campo, e quem parseia recebe `﻿.gitignore` onde pediu `.gitignore`. Escrever o BOM ficou fora do `WriteCSV`, que não tem como saber para onde seus bytes vão. **Revê a ADR-004**, que o tratava como propriedade do formato. |
| **RN-06** | **A ferramenta não renomeia o que quem roda nomeou.** A extensão é acrescentada só quando falta. Caminho com extensão trocada fica como veio. |
| **RN-07** | **Caminho inválido é recusado antes de qualquer arquivo existir.** A validação do `--output` acontece antes do `os.Create`, então uma recusa nunca deixa rastro no disco — nem arquivo vazio, nem arquivo com nome errado. É a postura da `RN-04` aplicada ao destino em vez da flag. |
| **RN-08** | **`--expand` e `--expand-dir` juntos são erro, não redundância silenciosa.** `--expand` já cobre `--expand-dir`; combiná-los aciona a mesma postura da `RN-04`. |
| **RN-09** | **Gravar não é decidir pelo usuário.** O `add` avisa quando a regra não pegou nada e quando pegou caminho já rastreado, e para por aí: nunca roda `git rm --cached`, nunca reescreve linha existente, nunca reordena o arquivo. |

### 4.1 Regras herdadas

- **O erro que o usuário lê é o da ferramenta, não o do sistema** — `RN-07` da [SRS — branches](branches.md), que nasceu para o `stderr` do git e vale igual para o filesystem. `os.Create` devolve `open relatorios/x.csv: no such file or directory` — mensagem do sistema, em inglês, numa ferramenta cujo resto está em português. Diretório ausente passa a dizer **qual** falta; falta de permissão diz isso. O que não se reconhece continua carregando o erro de origem, porque adivinhar seria pior do que citar.
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `ls-files` e `check-ignore` são plumbing, e `-z` com `-v` é saída feita para script.
- **Lista no `stdout`, erro no `stderr`** — `RN-09` da [SRS — branches](branches.md). Vale também para a sugestão do `tsv`: no `stdout` ela entraria no meio do CSV e corromperia o arquivo.
- **Nenhuma linha de saída termina em espaço** — `RN-10` da [SRS — branches](branches.md).
- **O domínio não escreve na tela** — `RN-11` da [SRS — branches](branches.md). `internal/ignore` devolve `[]Entry` e `AddResult`. Inclusive `Entry.Directory`: a barra final é o git reportando o que o `--directory` colapsou, então quem a interpreta é o domínio, e a apresentação lê um booleano em vez de conhecer a convenção de saída do `ls-files`.

---

## 5. Interface

```console
$ gtr ignore list
Ignorados (4):
  CAMINHO             ORIGEM               PADRÃO
  app.log             .gitignore:2         *.log
  local-only/         .git/info/exclude:1  local-only/
  node_modules/       .gitignore:1         node_modules/
  relatório 2026.csv  .gitignore:4         relat*.csv

Diretório ignorado conta como uma linha só. Use --expand para listar arquivo a arquivo.
```

```console
$ gtr ignore list --format csv
origem,linha,padrão,caminho
.gitignore,2,*.log,app.log
.git/info/exclude,1,local-only/,local-only/
```

| Flag de `list` | Padrão | Efeito |
|---|---|---|
| `--expand` | `false` | Lista arquivo a arquivo em vez de colapsar o diretório |
| `--expand-dir <caminho>` | vazio | Expande só o diretório informado; repetível |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar; sem extensão, a do formato é acrescentada |
| `--separator <s>` | `,` | Só com `--format csv`. Conjunto fechado: `,` `;` `\|` `\t` |
| `--no-header` | `false` | Só com `csv` ou `tsv`. Omite a linha de nomes das colunas |

| Flag de `add` | Padrão | Efeito |
|---|---|---|
| `--local` | `false` | Grava em `.git/info/exclude`, que vale só neste clone |
| `--global` | `false` | Grava no `core.excludesFile`, que vale na máquina inteira |
| `--force` | `false` | Com `--global`, configura o `core.excludesFile` ausente em vez de recusar |

---

## 6. Testes

**100% em `cmd`, `internal/ignore` e `internal/format`.**

**A suíte inteira roda contra o fake.** A pipeline foi conferida contra o `git` 2.43.0 antes da implementação, e o binário foi exercitado em repositório real a cada entrega — é a defesa contra o buraco que o CONTRIBUTING nomeia: nada no portão do CI depende do git de verdade.

### 6.1 Mutações, e os quatro buracos que elas acharam

Cerca de 60 mutações deliberadas. Quatro sobreviveram e viraram teste:

| Mutação sobrevivente | O que faltava |
|---|---|
| `index+3 < len(fields)` → `<=` no parse | Nenhum caso produzia **registro truncado**, então a guarda contra estourar o slice passava de graça. Fechado com `TestListDropsATruncatedRecord`, que monta uma cauda cortada de verdade |
| `ParseSeparator` devolvendo `Comma` em vez de zero no erro | Invisível pelo comportamento, mas é contrato de API compartilhada: devolver separador utilizável faria quem ignorasse o erro gravar um CSV que ninguém pediu |
| `file.Close()` com o erro engolido | Flush que falha perde a cauda do arquivo em silêncio. Inalcançável com arquivo real — resolvido extraindo `writeAndClose`, testável com um `WriteCloser` falso. Três casos: fecha sempre, reporta a falha do close, mantém a falha da escrita na frente |
| Ignorar o erro do `Path` no comando | O caminho recusado sai vazio, então o `os.Create("")` falhava logo depois e o comando errava de qualquer jeito — só que com a mensagem errada. O teste checava que houve erro, não **qual**. Fechado exigindo a mensagem da recusa |

**Duas asperezas só apareceram por explorar as bordas do `--output` à mão**, depois da suíte verde e das mutações mortas: a barra final e a mensagem crua do `os.Create`. Nenhum teste as teria pego, porque nenhum as previa — registro de que cobertura e mutação medem o que já foi imaginado.

### 6.2 Cenários que precisam provar que existiram

Dois testes verificam **ausência** e passariam de graça com montagem vazia, então checam a pré-condição antes:

- "a dica de `--expand` some" confirma primeiro que a entrada está listada;
- "o BOM não sai no `stdout`" confirma primeiro que a listagem saiu.

### 6.3 O README é verificado executando

Os blocos de saída do README são comparados byte a byte contra o binário, e **cada linha de exemplo do `--output` é executada de verdade**. Isso pegou um exemplo que falhava pela regra descrita duas linhas abaixo dele — o `mkdir -p` passou a fazer parte do exemplo.

### 6.4 Refactors que afirmam preservação

Dois commits `refactor:` afirmam preservar comportamento, e a afirmação foi verificada contra o binário anterior, **incluindo os caminhos de erro**: 11 invocações com `stdout`, `stderr` e código de saída idênticos. O mesmo método foi aplicado ao dividir a entrega em commits menores.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `2504eaf` | O port ganha `RunWithInput` e o `git.ExitError`, sem que a interface `Runner` mude — `RN-02`, `RN-03` |
| `292bd11` | `NewRootCommand` passa a receber a interface que o ponto de injeção precisa. `git.Runner` continua com um método |
| `3d0f6dc` | `gtr ignore list` — `RF-01`, `RF-02`, `RN-01` a `RN-03` |
| `cbbd1c8` | A dica de `--expand` só aparece quando algo colapsou. Nasce `Entry.Directory` — `RF-02` |
| `f1f74a3` | Cabeçalho da listagem de texto: `CAMINHO`, `ORIGEM`, `PADRÃO` — `RF-01` |
| `7688f5b` | Nasce `internal/format`. `--format csv` e `--separator` de conjunto fechado — `RF-05`, `RN-04` |
| `55a95f7` | `--format tsv` e a sugestão no `stderr` — `RF-05` |
| `8507e63` | `--format json` — `RF-05` |
| `da58e83` | `--output` e o `writeAndClose` — `RF-06`, `RN-06` |
| `cd91500` | O BOM sai do `stdout` e passa a valer só para arquivo — `RN-05` |
| `fe7bfaf` | Cabeçalho do CSV — `RF-05` |
| `717e10e` | As escolhas de renderização viram um valor só, antes que `render` chegasse a seis parâmetros com dois `bool` colados |
| `54650a2` | `--no-header` — `RF-05`, `RN-04` |
| `d65e42e` | O README ganha o comando. Os dois blocos de saída conferidos byte a byte contra o binário |
| `10145fd` | Caminho terminado em separador é recusado — `RF-06`, `RN-07` |
| `184edca` | A falha de criação passa a sair em português e acionável |
| `ec0b353` | O help e o README passam a dizer que o `--output` recebe caminho — `RF-06` |
| `810d594` | `--expand-dir` e a completion dinâmica — `RF-03`, `RF-04`, `RN-08` |
| `e5682f0` | `internal/ignore.Add`: destino, cobertura, duplicata, escape, quebra de linha e os dois avisos — `RF-07` a `RF-09`, `RN-09` |
| `78ab5c5` | `gtr ignore add` no `cmd` — `RF-07`, `RF-08` |
| `c17a6c7` | O caminho padrão do `core.excludesFile` passa a seguir o git, não a stdlib — `RF-08` |

---

## 8. Follow-ups conhecidos

- **`gtr ignore why <caminho>`** — não especificado.
- **Limitação conhecida: caminho começando em espaço.** O `CommandRunner.run` faz `strings.TrimSpace` na saída crua. Com `-z` o NUL final protege a cauda, mas um caminho cujo **primeiro** caractere seja espaço perderia esse espaço. Mesma família da `RN-01`. Consertar exige um caminho de saída crua no port.
- **O JSON deste comando é envelope, não array puro.** Array e envelope quebram um ao outro nos dois sentidos, e a migração foi feita enquanto o comando tinha um dia de vida, junto com a adoção do `--format` pelos outros. Ver o envelope como porta de mão única nos itens em aberto do projeto.
- **O chrome do cobra está em inglês** (`unknown command`, `Usage:`, `Flags:`), enquanto tudo que o `gtr` escreve está em português. Precede esta feature.
