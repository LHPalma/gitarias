---
titulo: SRS — quebrar um diff em commits
data: 2026-08-09
status: commits check entregue • split não implementado
comando: gtr commits check
pacotes:
  - cmd/commits.go
  - cmd/checked_table.go
  - internal/commits
  - internal/exec
commits:
  - a8e9b08
  - 4489a20
  - 2dc274e
  - d5eb734
  - 2765de1
  - 8db82c7
  - cb98614
fonte_externa: nenhuma
---

# SRS — quebrar um diff em commits

- **Data:** 2026-08-09
- **Feature:** `cmd/commits` + `internal/commits` + `internal/exec` • `cmd/split` + `internal/hunk`
- **Status:** **§3 (`commits check`) entregue e completo** — na `main`, incluindo a cláusula de sinal da `RN-06`, no §10. **§4 (`split`) não implementado**, e depende do contrato `Selector` pelo mesmo motivo que a [SRS — export de diff](diff.md): a seleção nasce junto
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar duas ferramentas para transformar uma árvore suja grande em **vários commits pequenos que se sustentam sozinhos**, e para provar que eles se sustentam.

**O git já sabe fatiar.** O `git add -p` mostra o diff hunk a hunk e pergunta o que entra no índice; o `git add -N` faz o arquivo novo aparecer nessa lista; o `git stash push --keep-index` esconde o que ficou de fora para você testar só o que vai commitar. As peças existem e funcionam.

**O que não existe é resposta para duas perguntas**, e são elas que fazem a diferença entre um histórico fatiado e um histórico útil:

1. **"Este commit compila sozinho?"** O `add -p` deixa você montar felizmente um commit que não compila, e nada avisa.
2. **"Todos os commits deste range passam?"** O mais perto que o git chega é `git rebase --exec 'go test ./...' main`, que **reescreve o histórico para checar** e **para no primeiro vermelho**. Quem quer só *saber* não quer reescrever nada, e quer a lista inteira.

É o mesmo formato de defeito que a [SRS — export de diff](diff.md) registra: *o caminho óbvio corrompe em silêncio e o caminho certo é uma incantação que ninguém decora.* Aqui o silêncio é mais longe: um commit intermediário quebrado não incomoda ninguém no dia, e cobra a conta meses depois, quando o `git bisect` para nele, ou quando alguém tenta reverter só aquele commit.

### 1.2 O caso concreto que definiu o escopo

A adoção do `--format` pelo `branches` e pelo `worktrees` foi feita num turno só e depois quebrada em seis commits, à mão. O trabalho real não foi escolher os hunks — foi **descobrir a ordem**: o commit 1 só podia ser refatoração pura se o envelope do JSON saísse dele, o que exigiu reverter dois arquivos, commitar e restaurar depois. O `add -p` não ajuda em nada nisso.

E a verificação de que os seis compilavam sozinhos foi feita extraindo cada um com `git archive` para um diretório temporário e rodando os testes lá. Funciona, é meia dúzia de linhas de shell, e **é exatamente o §3 deste documento.**

### 1.3 Escopo

- **`gtr commits check`** — roda um comando em cada commit de um range e reporta todos.
- **`gtr split`** — quebra a árvore suja em vários commits, com seleção de hunks por grupo e verificação entre um e outro.

**Fora de escopo:**

- **Reescrever histórico existente.** `git rebase -i` faz, e embrulhar não se paga. O `split` age sobre a **árvore suja**, antes de existir commit.
- **Análise de dependência entre arquivos.** Saber que um arquivo não compila sem o outro exige entender a linguagem, e isso quebraria a propriedade que define o projeto: orquestrar o `git` que já está na máquina. O §3 ataca o mesmo problema por fora — não previne o commit quebrado, **mostra qual é**, e custa uma fração.
- **Adivinhar o agrupamento.** Nada de heurística que decide sozinha o que vai junto. Ver `RN-05`.

### 1.4 Definições

| Termo | Significado |
|---|---|
| **Hunk** | Bloco contíguo de linhas alteradas dentro de um arquivo, com contexto em volta. É a menor unidade que o `git add -p` sabe encenar, e a unidade de seleção do `split`. |
| **Range** | Intervalo de commits, na notação do git: `main..HEAD` são os commits que estão em `HEAD` e não em `main`. Aqui o argumento é só a base, e o topo é sempre o `HEAD`. |
| **Commit que se sustenta** | Commit cuja árvore, isolada de tudo que veio depois, satisfaz o comando de verificação. Não é o mesmo que "o commit final passa" — é a propriedade que o `git bisect` e o revert por commit consomem. |
| **Extração** | Materializar a árvore de um commit num diretório fora do repositório. `git archive --output=<tar>` seguido de untar pela stdlib — **não escreve nada** no repositório, ao contrário de `git worktree add`, que registra a árvore em `.git/worktrees/`. |
| **Índice** | A área de stage. O `split` a usa como instrumento e tem de devolvê-la como encontrou. Ver `RN-04`. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/git/                execução de git — Run e RunWithInput, ambos com context

internal/exec/               executa o que não é git
├── runner.go                interface Runner: diretório, nome e argv
├── result.go                Result: código e saída combinada
├── command_runner.go        exit != 0 é Result; não começar é erro
└── exectest/                fake roteirizado por ordem de chamada

internal/commits/            domínio da verificação
├── commit.go                Commit: sha e assunto
├── result.go                Result: commit, código de saída e saída capturada
├── extractor.go             o contrato: Extract e Release
├── archive_extractor.go     git archive; não escreve no repositório
├── worktree_extractor.go    git worktree; para quem precisa do .git
├── untar.go                 só stdlib, com guarda de travessia
└── repo.go                  Repo: Ensure, Range, Check

internal/hunk/               do split — não implementado
├── hunk.go                  Hunk: arquivo, cabeçalho, linhas
├── group.go                 Group: nome do commit e os hunks dele
└── repo.go                  Repo: Ensure, Hunks, Stage, Commit, Rollback

cmd/commits.go               front de texto
cmd/checked_table.go         a tabela do relatório, nos quatro formatos
cmd/split.go                 não implementado
```

### 2.2 O port de git não serve, e é a quarta pressão sobre ele

O `git.Runner` roda **git**. O `commits check` precisa rodar `go test`, `make`, `pytest` — comando que não é git. Não é caso de estender o `Runner`: é outro executor, e forçá-lo no mesmo port faria "runner de git" deixar de significar alguma coisa.

Daí o `internal/exec` separado. Ele não conhece git, e o `internal/commits` conhece os dois — mesma direção de dependência que o `cmd` já tem com `ui` e os domínios.

**As quatro pressões sobre o port, vistas juntas:** `stdin` (da [SRS — equivalência](equivalencia.md), entregue em `2504eaf`), variável de ambiente (da [SRS — export de diff](diff.md)), executar não-git (esta, entregue em `a8e9b08`) e `context` para cancelamento (entregue em `2765de1`, §10). As duas primeiras são capacidade *do* git; a terceira não é, e por isso saiu do pacote em vez de entrar nele; a quarta atravessa os dois.

**O ponto de injeção ganhou o segundo runner:** `NewRootCommand(git, exec)`. Construí-lo por dentro tornaria a verificação irroteirizável nos testes, e relatório que não dá para testar não é relatório.

### 2.3 Fluxo do `commits check`

1. Valida as flags. **Antes de tocar no git.**
2. Verifica que o diretório atual é um repositório git.
3. Lista os commits do range com `log --reverse --format=%H%x00%s`, do mais antigo para o mais novo. O `rev-list` foi o esboço; ele emite duas linhas por commit quando se pede formato, e o `log` entrega sha e assunto num registro só.
4. Para cada commit: extrai a árvore num diretório temporário, roda o comando ali, captura código de saída e saída.
5. Reporta **todos**, e sai com 1 se algum falhou.
6. Apaga os diretórios temporários, **inclusive sob interrupção** — o sinal é interceptado e a varredura retorna pelo caminho normal, que é o único em que `defer` roda.

### 2.4 Fluxo do `split`

1. Verifica repositório e que a árvore está suja.
2. Levanta os hunks de tudo que mudou, incluindo untracked (via `add -N` em índice temporário, como a [SRS — export de diff](diff.md) já resolve).
3. Apresenta os hunks ao `Selector`, uma rodada por grupo, até o usuário fechar os grupos.
4. Para cada grupo, na ordem: encena os hunks, commita, e **roda a verificação** se houver.
5. Verificação vermelha **para tudo** e oferece o desfazer.

---

## 3. Requisitos funcionais — `gtr commits check`

### RF-01 — Rodar um comando em cada commit de um range

`gtr commits check <base> -- <comando>` roda o comando na árvore de cada commit de `<base>..HEAD`, do mais antigo para o mais novo, e reporta o resultado de cada um.

Range vazio não é erro: informa que não há commit no intervalo e sai com 0.

### RF-02 — Reportar todos, nunca parar no primeiro vermelho

Um commit vermelho **não interrompe** a varredura. O valor da ferramenta é a lista inteira: saber que os commits 1 e 4 quebraram e os outros não é diagnóstico; saber que "o 1 quebrou" é o que o `rebase --exec` já dá.

A saída do comando que falhou é mostrada, sob o commit dela. A dos que passaram não — salvo `--verbose`.

Saída 1 se **qualquer** commit falhou, 0 se todos passaram.

### RF-03 — Não escrever no repositório

A verificação usa `git archive` para um diretório temporário. Nenhuma ref é criada ou movida, nada entra no banco de objetos, o índice e a árvore de trabalho não são tocados, e o `HEAD` fica onde estava.

Consequência observável: rodar o `check` com a árvore suja **não conflita** com o trabalho em andamento, e `git status` antes e depois é idêntico. É isso que mantém a feature fora da **ADR-008** e que a separa do `rebase --exec`.

**Limitação assumida:** a árvore extraída não tem `.git`. Comando que precise de metadado de git — contar commits, ler tags, gerar versão a partir do `describe` — falha ali. O `--worktree` troca a extração por `git worktree add --detach`, que resolve isso ao custo de escrever em `.git/worktrees/`. **O padrão é o que não escreve.**

### RF-04 — Emitir o relatório em formato estruturado

`--format text|csv|tsv|json`, com `--output`, `--separator` e `--no-header` — **ADR-004**, sem código novo de formatação.

- **csv/tsv:** `sha`, `assunto`, `código`, `estado`.
- **json:** envelope com a chave `commits`, e o `estado` como token — `passed` / `failed`.

O comando verificado e o range vão para o `stderr` nos formatos delimitados, e para dentro do envelope no json — mesmo tratamento que a base do `branches` recebe.

---

## 4. Requisitos funcionais — `gtr split`

### RF-05 — Agrupar hunks e commitar na ordem

`gtr split` apresenta os hunks da árvore suja e monta um commit por grupo, na ordem em que os grupos foram definidos. Cada grupo recebe uma mensagem.

Hunk não atribuído a nenhum grupo **fica na árvore**, não commitado. Sair do `split` com trabalho de fora é caso legítimo e não é aviso de erro.

### RF-06 — Verificar entre um commit e o seguinte

Com `--check -- <comando>`, o comando roda depois de cada commit, na árvore **do commit recém-criado** — sem o que ainda não foi encenado, que é a única forma de a verificação significar alguma coisa.

Vermelho **para na hora**, antes de criar o commit seguinte. O que já foi commitado permanece, e o comando diz exatamente onde parou e como desfazer.

### RF-07 — Desfazer o que a sessão criou

`gtr split --abort` devolve o repositório ao estado anterior: os commits criados pela sessão são desfeitos com `reset --mixed` até o `HEAD` original, e todas as alterações voltam para a árvore.

**Nenhuma alteração é perdida** — o `--mixed` preserva o conteúdo na árvore de trabalho. O SHA do `HEAD` de origem é registrado no início e mostrado no fim, para que o desfazer manual seja sempre possível mesmo que o comando não esteja disponível.

### RF-08 — Falhas com código de saída

Fora de repositório git, com árvore limpa, ou com falha ao encenar, o comando escreve no `stderr` e sai com 1.

---

## 5. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O comando de verificação chega como `argv`, depois de `--`, nunca como linha de comando a ser parseada.** `--run "go test ./..."` obrigaria a ferramenta a decidir onde a string se divide, e fazer isso direito é reimplementar um shell — com aspas, escape e glob. `gtr commits check main -- go test ./...` entrega os argumentos já separados pelo shell de quem chamou. É a extensão direta de rodar sem `sh -c`. |
| **RN-02** | **A verificação não escreve no repositório.** Extração por `git archive` para diretório temporário. Sem ref nova, sem objeto novo, sem tocar índice, árvore ou `HEAD`. É o que permite rodar o `check` com trabalho em andamento e o que mantém a feature fora da **ADR-008**. O `--worktree` é a exceção explícita, e é opt-in justamente por deixar de valer esta regra. |
| **RN-03** | **Todo commit do range é reportado.** Nem vermelho interrompe, nem verde é omitido da lista. Um relatório que para no primeiro erro obriga a rodar de novo depois de cada correção, e é o que já se tem com `rebase --exec`. |
| **RN-04** | **O índice é instrumento, e volta como estava.** O `split` encena hunks para commitar; ao terminar, com sucesso, com falha ou com `--abort`, o índice tem de refletir exatamente o que restou na árvore. Índice sujo deixado para trás é pior do que não ter rodado: quem for commitar depois leva junto o que não escolheu. |
| **RN-05** | **A ferramenta não adivinha o agrupamento.** Nada de agrupar por diretório, por prefixo de nome ou por proximidade no diff. Um agrupamento errado sugerido com confiança é pior do que nenhum: ele parece revisado e não foi. O que se automatiza é a **execução** do agrupamento e a **prova** de que ele se sustenta, não a escolha. |
| **RN-06** | **Diretório temporário é sempre removido**, inclusive com interrupção por sinal. Um `check` sobre trinta commits deixaria trinta cópias da árvore para trás. E o processo filho morre junto: sem isso o `go test` interrompido continuaria rodando órfão, que é pior que o diretório vazado. Ver §10. |

### 5.1 Regras herdadas

- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `log` e `diff` de plumbing, nunca da saída humana.
- **Comandos rodam sem shell** — `RN-06` da [SRS — branches](branches.md), tanto os de git quanto o de verificação. Ver `RN-01`.
- **Relatório no `stdout`, progresso e erro no `stderr`** — `RN-09` da [SRS — branches](branches.md). É o que torna `gtr commits check main -- go test ./... --format json > relatorio.json` possível sem que a linha de progresso contamine o arquivo.
- **Nenhuma linha de saída termina em espaço** — `RN-10` da [SRS — branches](branches.md).
- **O domínio não escreve na tela nem lê do teclado** — `RN-11` da [SRS — branches](branches.md). `internal/commits` e `internal/hunk` devolvem `[]Result` e `[]Hunk`; a seleção é pedida pela apresentação através do `Selector`, e `passed`/`failed` é enum, com o rótulo escolhido em `internal/ui`.

---

## 6. Interface

```console
$ gtr commits check HEAD~2 -- go build ./...
Verificando 2 commits sobre HEAD~2.

  VERMELHO  aa5f8a0  segundo: chama dois() que ainda nao existe
      # quebrado
      ./main.go:3:30: undefined: dois
  verde     794ecd4  terceiro: define dois()
erro: 1 de 2 commits não se sustenta sozinho
```

| Flag de `commits check` | Padrão | Efeito |
|---|---|---|
| `--verbose` | `false` | Mostra também a saída dos commits que passaram |
| `--worktree` | `false` | Extrai com `git worktree add` em vez de `git archive`, para comandos que precisam do `.git`. Deixa de valer a `RN-02` |
| `--format <f>` | `text` | `text`, `csv`, `tsv` ou `json` — **ADR-004** |
| `--output <caminho>` | vazio | Caminho do arquivo a gravar, em vez do `stdout` |

```console
$ gtr split --check -- go build ./...
Hunk 3 de 11 — cmd/branches.go

  @@ -42,6 +45,14 @@ func runBranches(
  +	chosen, err := options.resolve(command)
  +	if err != nil {
  +		return err
  +	}

  [x] grupo 1: refactor: share the output machinery
  [ ] grupo 2: feat: give branches the shared output formats
  [ ] deixar na árvore
```

---

## 7. Testes

**100% de statements** em `internal/commits`, `internal/exec`, `exectest`, em `cmd/commits.go` e `cmd/checked_table.go`, e nos rótulos deste comando no `internal/ui` — com a suíte inteira rodando contra os fakes, e nada no portão do CI dependendo do git de verdade. Os cenários do §7.2 estão implementados; os do §7.3 seguem sendo aceitação do que não existe.

**Uma linha do relatório de cobertura merece explicação, porque parece buraco e não é:** o `ArchiveExtractor.Release` aparece em 0% porque **não tem statement nenhum**. O corpo é vazio de propósito — ele existe só para cumprir o contrato `Extractor`, que o `WorktreeExtractor` precisa de verdade, para rodar `worktree remove --force`. A extração por `archive` escreve num diretório temporário que quem chamou já apaga, então não há o que liberar.

### 7.1 Duas asperezas que a implementação achou

- **O fake de git casa por comando exato, e o caminho do `archive` é temporário e aleatório.** Não dá para roteirizar `archive --output=/tmp/gtr-check-918273/<sha>.tar`. Resolvido no teste, não na produção: um decorador do `gittest.Runner` intercepta `archive` e `worktree` e materializa um tar de verdade. O código de produção não ganhou costura para ser testado.
- **A guarda de travessia nasceu morta, e o teste provou.** A primeira versão normalizava o caminho *antes* de checar, então `../fora.txt` virava `fora.txt` dentro do destino e a verificação nunca disparava — segura por acidente, e inalcançável por cobertura. O teste que exigia **recusa** derrubou a ordem, e a guarda passou a existir de fato.

### 7.2 O cenário que define o `commits check`

Três commits montados de propósito, com o do meio quebrado — um commit que **só** compila junto com o seguinte:

| # | Cenário | Esperado |
|---|---|---|
| 1 | Range de 3 com o do meio quebrado | Os **três** reportados, o do meio vermelho, saída 1. É o teste que separa a ferramenta do `rebase --exec`, que reporta um e para — `RF-02`, `RN-03` |
| 2 | `git status` e `rev-parse HEAD` antes e depois | Idênticos, **com a árvore suja** durante a rodada — `RF-03`, `RN-02`. Conferido também contra o git de verdade, §8.1 |
| 3 | Listagem de `git worktree list` antes e depois | Idêntica sem `--worktree`; é o risco concreto da extração — `RN-02` |
| 4 | Comando com argumento contendo espaço, após `--` | Chega inteiro como um argumento só, sem shell — `RN-01` |
| 5 | Range vazio (`HEAD..HEAD`) | Mensagem específica, saída 0 — `RF-01` |
| 6 | Interrupção no meio da varredura | Nenhum diretório temporário sobra e **nenhum processo filho fica órfão** — `RN-06`. Conferido também com `SIGINT` no binário de verdade |
| 7 | Todos verdes | Saída 0, e saída de comando nenhuma no relatório sem `--verbose` — `RF-02` |
| 8 | Fora de repositório git | Erro claro, saída 1 |

### 7.3 O cenário que define o `split`

| # | Cenário | Esperado |
|---|---|---|
| 9 | Dois grupos, o primeiro não compilando sozinho, com `--check` | Para depois do commit 1, commit 2 não é criado, mensagem diz onde parou — `RF-06` |
| 10 | `--abort` depois do cenário 9 | `HEAD` de volta ao original e **todas** as alterações de volta na árvore, nenhuma perdida — `RF-07` |
| 11 | Índice depois de sucesso, de falha e de `--abort` | Reflete o que restou na árvore nos três casos — `RN-04` |
| 12 | Hunk deixado de fora de todo grupo | Continua na árvore, não commitado, sem aviso de erro — `RF-05` |
| 13 | Árvore com arquivo untracked | Aparece entre os hunks; o índice real termina byte a byte igual — herda a regra de índice temporário da [SRS — export de diff](diff.md) |

### 7.4 Ponto de atenção do método

Os cenários 1, 9 e 10 são do tipo que a [SRS — equivalência](equivalencia.md) registra como armadilha: **teste de cenário precisa provar que o cenário existiu.** Se a montagem do commit quebrado falhar em silêncio e ele compilar, o cenário 1 passa de graça reportando três verdes. A pré-condição — que o commit do meio de fato não compila — precisa ser conferida antes da asserção.

O cenário 10 tem o mesmo risco na direção oposta: se a montagem não criar commit nenhum, o `--abort` "funciona" sem ter feito nada.

---

## 8. Rastreabilidade

| Commit | Entrega |
|---|---|
| `a8e9b08` | Nasce `internal/exec`. Saída diferente de zero é `Result`, comando que não começa é erro — `RN-01` |
| `4489a20` | Nasce `internal/commits`: extração por `archive` e por `worktree`, `untar` com guarda de travessia, `Range` e `Check` — `RF-01` a `RF-03`, `RN-02`, `RN-03` |
| `2dc274e` | `gtr commits check`, o relatório e os quatro formatos. `NewRootCommand` ganha o segundo runner — `RF-01` a `RF-04`, `RF-08` |
| `d5eb734` | O README ganha o comando. Todos os blocos rodados contra o binário, inclusive o que falha |
| `2765de1`, `8db82c7` | O `context` atravessa os ports e os domínios; a cláusula de sinal da `RN-06` — §10 |
| `cb98614` | O furo de host na guarda de travessia — §9 |

### 8.1 A verificação que a entrega permitiu fazer de si mesma

Os commits acima foram verificados **pela ferramenta que eles entregam**, com `gtr commits check f150939 -- go test ./...`: todos verdes, e o repositório conferido antes e depois — `HEAD`, índice, árvore e lista de worktrees idênticos, nenhum temporário sobrando. É a `RN-02` observada em vez de afirmada.

O cenário vermelho foi conferido à parte, num repositório montado com o commit do meio chamando uma função que só existe no seguinte — exatamente a forma de defeito que a feature existe para achar.

---

## 9. Onde a implementação divergiu do desenho

Três desvios, todos decididos escrevendo o código:

- **O pacote chama-se `internal/exec`, não `internal/command`.** `command` colidia com a variável `command` que todo comando cobra usa no projeto inteiro; manter o nome obrigaria a renomear a variável num arquivo só, contra a convenção dos outros. Dentro do pacote o `os/exec` entra com alias.
- **A extração é `git archive` mais `archive/tar` da stdlib**, não um pipe para o `tar` do sistema. O desenho previa `git archive <sha> | tar -x`, o que exigiria shell — contra a regra de rodar sem `sh -c` — ou um `tar` instalado. Untar em Go não depende de nenhum dos dois. **Junto veio o que o desenho não previa:** guarda de path traversal em cada entrada. O `git archive` nunca emite `..`, mas extrator que confia na entrada só está correto até a entrada mudar.

  **A guarda tinha um furo específico de host** (fechado no commit `cb98614`). Ela recusava caminho absoluto checando `filepath.IsAbs` — semântica do host que está **lendo** o arquivo. O formato tar sempre grava caminho com `/`, qualquer que seja o host que escreveu; num host Windows, uma entrada `/etc/passwd` não é "absoluta" para o `filepath.IsAbs` dali, e escapava a checagem. Corrigido recusando qualquer entrada começando em `/`, antes de qualquer normalização dependente de host — achado rodando a suíte pela primeira vez num host Windows de verdade, não em teoria.
- **São dois vocabulários de resultado, não um.** A tela fica com `verde`/`VERMELHO` — a caixa alta é o que faz a falha ser achada rolando —, mas a planilha recebe `passou`/`falhou`, porque gritar numa célula não é o registro que o `branches` usa nas dele. O JSON não recebe nenhum dos dois: `passed`/`failed` são tokens. Moram os dois em `internal/ui`.

---

## 10. A cláusula de sinal da `RN-06`

A limpeza do temporário sob interrupção saiu **depois** de o comando já estar na `main`, nos commits `2765de1` e `8db82c7`.

### A decisão de escopo

A leitura estreita seria: `context` só no `internal/exec`, porque das cinco chamadas do fluxo **quatro são de git e todas limitadas** — `log`, `archive`, `worktree add` e `worktree remove`. A única sem teto é a do comando de verificação. O argumento a favor era o precedente da **ADR-005**: nada de entregar capacidade sem consumidor, porque seria statement que nenhum teste alcança.

**O `context` atravessa os três domínios também**, e a decisão se sustenta por um motivo que a leitura estreita não pesava: o segundo consumidor é previsível. O `branches` com detecção de equivalência gasta **4 a 5 invocações de git por branch** — em 200 branches, cerca de mil chamadas. Cancelar aquilo vai interessar.

O custo temido — ramo de cancelamento sem teste em três domínios — não se materializou: **um teste por domínio fecha os três**.

### Duas sutilezas que só aparecem implementando

- **`ctx.Err()` tem de ser checado *depois* da chamada, não só antes.** O `exec.CommandContext` mata o filho no cancelamento, e o processo morto volta como **saída não-zero comum**. Sem a checagem posterior, o commit seria reportado como **vermelho** em vez de a varredura ser reportada como interrompida — uma ferramenta que acusa o commit errado por causa de um `Ctrl+C`.
- **O teste tem de cancelar no meio, não no começo.** Contexto cancelado antes do primeiro commit passa mesmo que o laço nunca mais consulte `ctx.Err()`. É a armadilha do §7.4 aplicada ao cancelamento: o cenário precisa provar que existiu.

### E os fakes cancelam

`gittest.Runner` e `exectest.Runner` devolvem `ctx.Err()` antes de qualquer coisa. Fake que ignorasse o contexto deixaria o teste de domínio afirmar um cancelamento que o runner real nunca executa — a suíte ficaria verde sobre uma garantia inexistente.

### Verificado com sinal de verdade

`SIGINT` num `gtr commits check ... -- sleep 60` em execução: saída `erro: interrompido` com código 1, **nenhum processo `sleep` vivo** e **nenhum diretório `gtr-check-*` em `/tmp`**. Antes, os dois sobreviviam.

### A ferramenta pegou o próprio erro desta entrega

A primeira divisão em commits separava a mudança de assinatura dos ports dos call sites no `cmd`. O `gtr commits check` rodado sobre os dois **reprovou o primeiro**: ele não compilava sozinho. A divisão foi refeita antes do push.

É o caso de uso que justificou construir o comando, ocorrendo um dia depois — e a evidência de que a `RN-03` (reportar todos, não parar no primeiro) importa: o segundo commit estava verde, e sabê-lo poupou investigar dois.

---

## 11. Follow-ups conhecidos

- **O `Selector` genérico deixa de ser opinião e vira necessidade.** A **ADR-002** esboçou `Select(candidates []branch.Branch)`. A [SRS — export de diff](diff.md) já mostrou que arquivo não é branch. O `split` é o **terceiro** consumidor e seleciona hunk, que não é nem um nem outro. Três tipos distintos é onde a própria ADR-002 diz que a divergência começa — o contrato nasce `Selector[T any]` ou nascem três seletores quase iguais.
- **O `split` precisa de seleção com mais de dois estados.** Os outros dois consumidores marcam ou desmarcam; aqui cada hunk vai para **um grupo entre N**, ou para nenhum. Isso não é checkbox, e o contrato genérico precisa comportar os dois modos ou admitir que são coisas diferentes. **Decidir antes de implementar**, não depois.
- **Ordem de dependência continua sem ferramenta.** O `check` mostra o commit quebrado, não sugere como consertar a ordem. Sugerir exigiria entender a linguagem — ver §1.3. Se algum dia valer a pena, o caminho honesto é um plugin por linguagem, não código no `gtr`.
- **Plano declarado em arquivo**, como alternativa à seleção interativa: um `.gtr-split.yaml` com os grupos e os caminhos, para quem prefere editar a escolher hunk a hunk, e para tornar o split reproduzível. Depende do formato de configuração da **ADR-004**.
- **`commits check` sobre range que não termina em `HEAD`.** Hoje o argumento é só a base. `--head <ref>` é extensão compatível.
- **O nome não está fechado.** `gtr commits check` tem subcomando como o `gtr ignore list`, mas `commits` é substantivo emprestado do git e `check` é vago. `gtr verify` foi considerado e é curto demais para dizer o que verifica. Dado o histórico do projeto com nomes, decidir antes de mexer.
