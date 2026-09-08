---
titulo: SRS — doctor
data: 2026-08-14
status: entregue
comando: gtr doctor
apelido: soundcheck
pacotes:
  - cmd/doctor.go
  - cmd/diagnosis_table.go
  - cmd/trimming_writer.go
  - internal/doctor
  - internal/ui
commits:
  - ca21cd9
  - 696c9b3
  - 020fdc9
  - 0a1b761
  - a51ccd9
  - bd6eaea
  - 426c05d
  - b2b7e6d
  - 3ef8a01
  - 5f502d6
  - 22ffd25
fonte_externa: nenhuma
---

# SRS — doctor

- **Data:** 2026-08-14
- **Feature:** `cmd/doctor` + `internal/doctor`
- **Status:** **entregue** — sete checagens: `git`, temporário, repositório, árvore, identidade, base e `gh`
- **Fonte externa:** nenhuma nas sete checagens deste documento. A de conexão e a de escopo do token, sim — ver [SRS — a conexão: doctor --online](doctor-online.md)

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr doctor`, que responde **se o `gtr` funciona aqui, agora**.

O projeto assume `git` no `PATH`. Sem essa checagem, cada comando falha na primeira chamada de git que alcançar, com a mensagem que aquela chamada produzir — diagnóstico por acidente, diferente conforme o comando.

**A pergunta é sobre o diretório, não sobre a máquina.** "A máquina tem o que o `gtr` precisa" deixaria de fora repositório e base, que são do lugar onde se está. É a leitura que o `flutter doctor` usa, e é mais útil: a pergunta de quem roda não é "meu computador está bem", é "por que isto não funciona".

### 1.2 Escopo

`git` no `PATH` **e em versão suficiente**, **diretório de temporários gravável**, estar num repositório, **nenhuma operação de git no meio do caminho**, **identidade de git configurada**, base detectável **e com a procedência dela declarada**, `gh` no `PATH`. Flag `--strict`. Os quatro formatos da **ADR-004**.

A ordem em que saem conta uma história: **máquina** (`git`, temporário) → **contexto** (repositório) → **estado do repositório** (árvore) → **configuração** (identidade, base) → **ferramenta opcional** (`gh`).

**Especificado à parte:** autenticação e escopo do token, em [SRS — a conexão: doctor --online](doctor-online.md). O §6 aqui registra a medição que tornou aquela entrega possível.

**Fora de escopo: consertar qualquer coisa.** O `doctor` diagnostica e diz como resolver; quem resolve é quem lê. É o mesmo limite que o `branches` põe ao listar as três formas de soltar uma branch presa sem executar nenhuma.

### 1.3 O nome

**`doctor`**, porque é a convenção que `brew`, `flutter` e o próprio `gh` usam — quem chega de outra ferramenta digita sem pensar.

**`soundcheck` é apelido**, e um easter egg: passagem de som é conferir que o equipamento funciona **antes** de tocar, que é exatamente o comando. Fica fora da ajuda da raiz para que o nome anunciado seja o óbvio; aparece em `gtr doctor --help`. Combina com o `gtr` ser abreviação de guitarra.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/doctor/             domínio do diagnóstico — não imprime nada
├── state.go                 State: Ok, Warning, Failure, Skipped
├── check.go                 Check: nome, estado, detalhe e dica
├── version.go               a mínima do git, e como ler uma saída de --version
├── operation.go             as operações de git detectáveis por ref
└── doctor.go                Diagnose, e uma função por checagem

internal/ui/doctor.go        DescribeCheck: o rótulo de tela
cmd/diagnosis_table.go       a tabela, nos quatro formatos
cmd/trimming_writer.go       alinha e apara — §5.2
cmd/doctor.go                o comando e o --strict
```

**O `internal/doctor` importa `internal/branch`, e isso é exceção consciente à regra de que os domínios não se conhecem.** Ele não é domínio par: é **agregador**. Checar a base exige a cadeia de resolução da [SRS — branches](branches.md), e reimplementá-la seria duplicar conhecimento — a duplicação que o CONTRIBUTING nomeia como a única de verdade.

A regra continua válida no que importa: **`branch`, `worktree`, `ignore` e `commits` seguem sem se conhecer**, ninguém importa o `doctor` a não ser o `cmd`, e o grafo continua acíclico.

### 2.2 Os quatro estados

| Estado | Significa | Afeta a saída? |
|---|---|---|
| `ok` | está no lugar | não |
| `falta` | o `gtr` não funciona sem isso | **sai 1** |
| `aviso` | quebra **um** comando, não a ferramenta | só com `--strict` |
| `--` | não se aplica aqui | nunca, nem com `--strict` |

**A distinção entre `falta` e `aviso` é o que o comando tem de útil.** Um diagnóstico que trata tudo como falha grita sobre `gh` ausente para quem nunca vai abrir um PR, e aí ninguém mais lê o que ele diz.

**`--` não é aviso disfarçado.** Fora de um repositório, nada está errado — falta contexto. Reclamar seria pedir para o usuário consertar o próprio `cd`.

---

## 3. Requisitos funcionais

### RF-01 — Conferir o que o `gtr` precisa, e dizer como resolver

`gtr doctor` roda as checagens em ordem e imprime, por linha, o estado, o nome e o detalhe. **Só o que não está `ok` carrega dica**, e a dica é acionável — um comando a rodar ou um endereço de onde instalar.

Quando a ferramenta consultada responde algo útil na falha, **a saída dela vira a dica** em vez de um texto próprio: `git` que está no `PATH` mas não roda diz mais sobre si do que qualquer frase genérica.

### RF-02 — Rodar fora de um repositório

É o **único comando que não exige um repositório git**. Fora de um, as checagens que dependem dele se declaram inaplicáveis e a saída é 0.

A base depende do repositório: sem ele, é pulada por dependência e não por falha própria, e diz isso.

### RF-03 — Distinguir o que quebra a ferramenta do que quebra um comando

| Checagem | Ausente vira | Por quê |
|---|---|---|
| `git` | **falha** | nenhum comando funciona sem ele |
| repositório | pulado | falta contexto, não falta peça |
| base | **aviso** | quebra só o `branches`, e mesmo lá há saída pelo `--base` |
| base **adivinhada** | **aviso** | o nome existe, mas veio de chute entre `main` e `master` — `RF-08` |
| identidade | **aviso** | quebra a sondagem de squash do `branches`, e só ela — `RF-07` |
| `git` abaixo da mínima | **falha** | o binário existe, mas não tem o comando que a resolução de base chama — `RF-06` |
| temporário | **aviso** | só o `commits check` escreve lá — `RF-09` |
| operação em curso | **aviso** | o `gtr` funciona, mas o `HEAD` não significa o de sempre — `RF-10` |
| `gh` | **aviso** | só os comandos de PR precisam |

### RF-04 — `--strict`

Promove aviso a falha para efeito de código de saída. Existe para portão de CI, que quer tudo redondo e não só tudo funcionando.

**Pulado nunca vira falha**, nem com `--strict`.

### RF-05 — Emitir o diagnóstico em formato estruturado

`--format text|csv|tsv|json`, com `--output`, `--separator` e `--no-header` — **ADR-004**.

No `json` o `state` é token — `ok`, `warning`, `failure`, `skipped`. No `csv` é o rótulo de tela.

### RF-06 — Julgar a versão do git, e não só exibi-la

Uma checagem que imprime a versão e **passa em qualquer coisa que responda** não diz nada: um git velho demais para rodar o que o `gtr` chama sairia igual a um saudável.

**A mínima é 2.22**, e o critério não foi escolhido, foi levantado: é o `branch --show-current`, o comando **mais novo** entre todos os que o `gtr` invoca, e do qual a resolução de base depende. Os outros pisos são mais baixos — `worktree list --porcelain` pede 2.7, e `commit-tree`, `cherry` e `archive` são antigos.

Abaixo da mínima é **falha**, e a dica **nomeia o comando que falta**, que é mais útil do que o número sozinho.

**Versão ilegível é aviso.** Se a versão é o critério, critério que não dá para avaliar não é aprovação. O `--strict` o promove como a qualquer outro aviso.

A comparação usa **só maior e menor**. É o que a mínima exige, e o patch é onde as distribuições sujam: o git do macOS anexa a build da Apple (`2.39.5 (Apple Git-154)`) e o do Windows anexa o sufixo dela (`2.44.0.windows.1`).

### RF-07 — Relatar a identidade de git em vigor

Mostra `Nome <email>` quando as duas chaves existem; **aviso** nomeando a que falta quando não. Quando faltam as duas, nomeia as duas — consertar uma e descobrir a outra depois é uma viagem a mais. Chave setada como **string vazia conta como ausente**: `config --get` sai com código 0 nesse caso.

**A checagem mostra, não julga** — `RN-06`.

### RF-08 — Dizer de onde a base veio, e não só qual é

A cadeia de resolução da [SRS — branches](branches.md) tem três níveis: o `--base`, o `refs/remotes/origin/HEAD` e, por último, o chute entre `main` e `master`. **Os três devolvem um nome**, e sem procedência os três sairiam como `ok base main`, indistintos.

Só o terceiro **pode estar errado sem que nada pareça errado**: com uma `main` local esquecida e a padrão de verdade sendo `develop`, o `--clean` responde *"mergeada na main"* à pergunta *"mergeada na develop"* — e deleta com base nisso.

| Origem | Estado | Detalhe |
|---|---|---|
| `origin/HEAD` | `ok` | `main, declarada pelo remoto` |
| chute por nome | **aviso** | `main, adivinhada pelo nome` • `git fetch origin` |
| nada resolve | **aviso** | `não determinável aqui` • `--base` |

A causa comum do chute é banal — clone sem refs de rastreamento remoto — e o conserto é um comando só, o que faz da dica algo acionável. **Ter chutado um nome e não ter nome nenhum são situações diferentes**, e leem-se como tais.

**O aviso soa mesmo quando o chute está certo**, e ele quase sempre está. A alternativa seria manter `ok` e só mostrar a procedência, que é a leitura da `RN-06`. Prevaleceu que aqui, ao contrário da identidade, há **deficiência conhecida com conserto conhecido**, e é disso que aviso trata.

### RF-09 — Sondar o diretório de temporários

O `commits check` materializa a árvore de **cada** commit sob um temporário. Um `TMPDIR` que recusa escrita o derruba **no meio de um range longo**, e não na porta. A checagem descobre isso na porta.

**A sondagem escreve.** Cria um diretório lá e o apaga em seguida — há teste afirmando que **não sobra nada**. Inspecionar permissão responderia a pergunta errada: montagem somente leitura, disco cheio e ACL negam a escrita **sem que o modo do arquivo mude**.

O motivo vem do sistema operacional, desembrulhado do `fs.PathError` para tirar o caminho que a stdlib prefixa — o caminho já é dito à parte, e o sufixo aleatório do nome só atrapalha:

```text
aviso  temporário   /nao/existe: no such file or directory
```

**Quando passa, não diz nada** — como a checagem de repositório. Nomear o diretório na linha saudável deixaria a saída **dependente da máquina** (`/tmp` no Linux, `/var/folders/…` no macOS) e os testes de saída exata reféns disso.

### RF-10 — Reconhecer operação de git no meio do caminho

O `branches` decide o que deletar a partir do `HEAD`, e **no meio de um rebase o `HEAD` não é o que parece**. A checagem diz isso antes que importe.

**Qual marcador cada operação deixa foi medido**, não lembrado — repositório montado e cada operação interrompida de propósito:

| Operação | Marcador |
|---|---|
| `merge` | ref `MERGE_HEAD` |
| `cherry-pick` | ref `CHERRY_PICK_HEAD` |
| `revert` | ref `REVERT_HEAD` |
| `rebase` | ref `REBASE_HEAD` |
| `am` | **ref nenhuma** — §5.8 |

Ref é formato contratual e passa pelo `git.Runner`. A alternativa seria parsear o `git status`, que é prosa, muda entre versões e pode ser localizado.

**Quando não há operação, a checagem diz `sem operação em curso` em vez de ficar muda**, e aí ela diverge do temporário de propósito. O nome "árvore" sozinho prometeria mais do que ela olha: quem lesse um `ok árvore` pelado poderia entender "nenhum arquivo modificado", e isso ela não verifica.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O `doctor` diagnostica e nunca conserta.** Não instala, não autentica, não cria branch. Um diagnóstico que age deixa de ser confiável para rodar por curiosidade, e rodar por curiosidade é o uso principal dele. |
| **RN-02** | **Dependência opcional ausente é aviso, nunca falha.** O critério é objetivo: se algum comando existente para de funcionar, é falha; se só um para, é aviso; se nenhum comando a usa ainda, é aviso de qualquer jeito. **Quem nunca vai usar o `gh` não pode ser avisado de que falta algo.** |
| **RN-03** | **Falta de contexto não é falha nem aviso.** Fora de um repositório o `gtr` está inteiro. O estado pulado existe para dizer isso sem alarmar, e por isso `--strict` não o toca. |
| **RN-04** | **O `doctor` não afirma o que não mediu.** Ver §6: `gh auth status` sai 0 mesmo declarando o token inválido, então o código de saída dele só prova "há token configurado". |
| **RN-05** | **Versão exibida é versão julgada.** Se um requisito de versão existe, ele é o critério; e se o número não pode ser lido, o critério não foi avaliado — o que é aviso, não aprovação. **A mínima sai do código, não do gosto:** é o comando mais novo que o `gtr` de fato invoca. |
| **RN-06** | **A identidade é relatada, nunca julgada.** Só o git resolve a precedência entre o config local e o global, e o `gtr` não tem como saber qual das duas o autor queria — inventar uma regra sobre qual e-mail é "certo" seria o `doctor` opinando sobre o repositório de quem o roda. **Quem lê a linha reconhece na hora**, que é todo o valor. |
| **RN-07** | **Mensagem de tela não usa `(s)`.** O parênteses é o que se escreve para não decidir a flexão, mas ele chega na tela como se fosse a decisão — e o singular é o caso comum. A flexão mora no `internal/ui`, ao lado do resto do vocabulário de exibição. Ver §5.6. |
| **RN-08** | **O doc comment declara o contrato, não a história.** O que o identificador faz, o que devolve e quais os casos de borda — e nada que exija conhecer roadmap, sessão ou intenção do autor para ser entendido. **A razão de decisão mora na especificação**, que é onde se revisita. Ver §5.5. |
| **RN-09** | **Capacidade se sonda usando-a, não inspecionando-a.** Para saber se dá para escrever, escreve-se — e apaga-se em seguida. Montagem somente leitura, disco cheio e ACL negam a escrita **sem que o modo do arquivo mude**, então ler permissão responde outra pergunta. A contrapartida é obrigatória: **a sondagem não deixa nada para trás**, e há teste afirmando isso. Ver §5.7. |
| **RN-10** | **Checagem diz algo quando o nome não basta, e cala quando basta.** O `temporário` passa mudo; a `árvore` diz `sem operação em curso`, porque um `ok árvore` pelado seria lido como "nenhum arquivo modificado" — que ela não verifica. **O detalhe existe para impedir que o nome prometa mais do que a checagem olha.** |

### 4.1 Regras herdadas

- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `--version` e código de saída, nunca o texto humano de `gh auth status`, que é prosa, muda entre versões e pode ser localizado.
- **Sem shell** — `RN-06` da [SRS — branches](branches.md). `git` e `gh` são invocados direto, pelo `internal/exec`.
- **O erro que se lê é acionável** — `RN-07` da [SRS — branches](branches.md). Toda checagem que não passa diz o próximo passo.
- **Nenhuma linha termina em espaço** — `RN-10` da [SRS — branches](branches.md), e essa foi a que quase escapou. Ver §5.2.
- **O domínio não escreve na tela** — `RN-11` da [SRS — branches](branches.md). `internal/doctor` devolve `[]Check` com o estado como enum; o rótulo mora no `internal/ui`.

---

## 5. Achados de implementação

### 5.1 O parser de versão precisou mudar por causa do `gh`

O `git` imprime **uma linha terminando na versão**, então pegar o último campo funcionaria. O `gh` imprime versão, data **e uma segunda linha com a URL do release** — o último campo dali é um link.

O parser lê o campo **depois da palavra `version`, na primeira linha**, com o comportamento antigo como reserva. Conferido contra os dois binários de verdade:

```text
git version 2.43.0                                       → 2.43.0
gh version 2.45.0 (2025-07-18 Ubuntu 2.45.0-1ubuntu0.3)  → 2.45.0
https://github.com/cli/cli/releases/tag/v2.45.0
```

O pacote do Ubuntu põe mais coisa entre parênteses do que a distribuição oficial, e o parser aguenta.

### 5.2 Espaço à direita, alinhamento, e o teste que não pegou

Checagem sem detalhe deixa a última célula vazia, e o `tabwriter` a preenche até a largura da coluna: a linha do `repositório` saía **terminando em espaços**.

**O teste da regra existia e não pegou**, porque só exercitava o caminho de falha — onde toda checagem tem detalhe. Roda contra os dois agora.

A primeira correção foi omitir a célula vazia. Resolveu o espaço e **quebrou o alinhamento**: célula que não termina em tab não entra na coluna, então `repositório` deixou de alargar a coluna e as outras linhas desalinharam.

A solução foi um writer que **apara cada linha ao passar**, entre o `tabwriter` e o destino. Mantém as duas propriedades **e** torna o erro do `Flush` real: antes ele ia para um buffer em memória, que nunca falha, e era statement inalcançável.

**Candidato a generalizar:** o `columns()` do projeto inteiro poderia aparar. Não foi feito porque mudaria a saída dos outros comandos, e essa é uma afirmação que precisa de verificação contra binário anterior.

### 5.3 O `--strict` esperou ter consumidor

Ficou de fora do primeiro incremento de propósito: ele promove aviso a falha, e nada avisava até a base indeterminável existir. Sem consumidor seria flag que nenhum teste alcança — o mesmo critério que a **ADR-005** aplica a capacidade de port sem consumidor.

### 5.4 A premissa da identidade foi medida, não suposta

A tentação era justificar a checagem com "você vai querer commitar depois". Isso seria o `doctor` opinando sobre intenção. A justificativa real é mais estreita e já existe: **a sondagem de equivalência por conteúdo monta um objeto de commit com `commit-tree`, e o git recusa montá-lo sem autor.** Rodado contra o git de verdade, com `HOME` e os configs neutralizados:

```console
$ git commit-tree <tree> -p HEAD -m sonda
Author identity unknown

*** Please tell me who you are.
```

Daí **aviso e não falha**: quebra a detecção de squash do `branches`, e só ela — mesmo critério da base.

**A medição cobrou o preço dela.** Para ver a saída do lado do aviso, o `user.name` local foi removido — e o backup do `.git/config` saiu **depois** do primeiro `--unset`, então a restauração devolveu um config já sem o nome. Quem denunciou foi a própria checagem recém-escrita, na invocação seguinte. Refeito na hora, sem commit no meio. É *nunca destruir na dúvida* aplicado à sessão e não ao código: **o backup vem antes do primeiro comando destrutivo, não depois.**

### 5.5 Comentário de documentação entrou no repositório, e levou dois commits para achar a forma

Go não tem vocabulário de tags — nada de `@param`, `@return` ou `<summary>`. O *doc comment* é prosa colada à declaração, **aberta pelo nome do identificador**, em frases completas, e o `gofmt` a canoniza desde o Go 1.19.

**`bd6eaea` consertou a forma.** O comentário do `minimumGit` abria com "O comando mais novo", que o `go doc` imprime como se descrevesse a variável em vez de nomeá-la.

**`3ef8a01` consertou o conteúdo, e essa foi a correção que importava.** Os comentários explicavam **como o código veio a ser**, e para isso puxavam contexto que quem lê o pacote não tem como sustentar: um item de roadmap, a intenção do autor, um episódio da sessão que os escreveu. **Comentário que precisa da história do projeto para ser entendido não é documentação do pacote.**

A regra que ficou: **o doc comment declara o contrato — o que o identificador faz, o que devolve e em que casos de borda** — e nada além. Quais campos o `parseRelease` ignora, para o que o `version` recua e quando devolve vazio, que estado o `identity` e o `base` devolvem e sob que condição.

Duas razões mudaram de casa nessa passagem, e ficam registradas aqui:

- **A procedência da base tem texto próprio no domínio do diagnóstico.** Reusar o `ui.DescribeSource` esbarraria na regra de que nenhum domínio importa o `internal/ui`. E as duas frases não dizem a mesma coisa: o `branches` nomeia o **nível da cadeia** ("detectada via origin/HEAD") e o `doctor` declara a **confiança** ("declarada pelo remoto" contra "adivinhada pelo nome"). Não é duplicação, então não há drift a temer.
- **A flexão mora no `internal/ui`** porque nenhuma tabela de traduções resolve singular e plural sozinha — ver §5.6.

### 5.6 O `(s)` era do projeto inteiro, não do `doctor`

O `doctor` fechava com `%d checagem(ns) falharam`, que para uma falha imprime `1 checagem(ns) falharam`. Procurando por `(s)`, `(es)` e `(ns)` no código de produção apareceram **nove** ocorrências, em quatro arquivos e três comandos — `branches`, `commits check` e `doctor`.

O parênteses é o que se escreve para não decidir, mas ele chega na tela **como se fosse a decisão**: `1 commit(s)` não é abreviação, é frase errada. E o singular é o caso comum de uma ferramenta que verifica um commit ou deleta uma branch.

Corrigido com o `ui.Plural`, ao lado do resto do vocabulário de exibição — flexão é apresentação, e os dois fronts vão precisar dela. **Onde varia uma palavra, a chamada é inline; onde varia a oração inteira** (`O commit se sustenta sozinho.` contra `Os 3 se sustentam sozinhos.`) **são dois ramos explícitos**, porque forçar oração por dentro do `Plural` sairia pior que o parênteses que ele substitui.

Dois testes existentes afirmavam o texto antigo e quebraram — **expectativa obsoleta, não regressão**. Conferido por mutação: forçando o `Plural` a devolver sempre o plural, cinco testes caem.

### 5.7 O caminho de permissão negada não é observável em container privilegiado

Da `RF-09`, só metade se verifica num container que roda como root: um `chmod 500` num diretório **não impede a escrita**, porque o kernel ignora o modo do arquivo para uid 0.

```console
$ id -u
0
$ chmod 500 travado && touch travado/sonda
(escreveu)
```

Então o único caminho de falha observável dali é **"o diretório não existe"**. "Permissão negada" precisa de máquina com usuário comum.

É o **segundo** caminho de falha que esse ambiente apaga, depois do proxy que reautentica a API do GitHub (§6.2). O padrão vale mais que os dois casos: **container privilegiado e proxy transparente são ambientes onde teste de caminho de falha passa de graça.** Medir ali responde "funciona", nunca "falha como deveria".

### 5.8 O `git am` é a operação sem ref, e o diretório dela tem dois donos

O `am` (*apply mailbox*) aplica patches em formato de e-mail — a ponta receptora do `format-patch` / `send-email`, o fluxo de contribuição por lista de discussão. **Ele não deixa ref, e a razão é boa:** num `rebase`, o `REBASE_HEAD` aponta para o *commit* sendo aplicado; no `am` não há commit, há **texto de patch**. Não existe objeto para uma ref apontar.

O que fica é o diretório `rebase-apply`. **Mas ele sozinho não prova nada**, e isso também foi medido:

| Situação | `REBASE_HEAD` | `rebase-apply/` | Marcador dentro |
|---|---|---|---|
| `git am` | ausente | existe | `applying` |
| `git rebase --apply` | **presente** | existe | `onto` |

O backend `--apply` do rebase usa o **mesmo diretório**. Quem separa é o arquivo `applying` — e o rebase, de todo modo, já foi reconhecido pela ref antes de chegar lá. **As refs vêm primeiro porque são inequívocas**, e há teste afirmando isso.

**O caminho vem do git**, e não de um `.git/rebase-apply` escrito à mão. Medido:

```text
da raiz:        .git/rebase-apply
de um subdir:   ../.git/rebase-apply
de um worktree: /tmp/…/.git/worktrees/wt/rebase-apply
```

Sempre relativo ao diretório corrente — que é de onde o `os.Stat` também parte, então os dois concordam de graça. Cravar o caminho erraria **dentro de worktree e sob `GIT_DIR`**, e as três formas foram conferidas rodando o binário contra um `am` conflitado de verdade, inclusive de um subdiretório.

É a **única checagem que toca o sistema de arquivos por conta de um estado de git** — as demais passam inteiras pelo port. O custo foi aceito: a curiosidade histórica do fluxo por e-mail vale o desvio.

---

## 6. A checagem de autenticação, e por que ela não está aqui

Ela é especificada em [SRS — a conexão: doctor --online](doctor-online.md). O que segue é a medição feita aqui, e que tornou aquela entrega possível.

O `gh` foi instalado para medir em vez de supor. Três achados verificados e um limite.

### 6.1 O que foi medido

| Situação | `gh auth status` | `gh auth token` |
|---|---|---|
| Sem token nenhum | sai **1** | sai **1** |
| Token presente, **declarado inválido pelo próprio comando** | sai **0** | sai **0** |

**Duas suposições caíram:** o código de saída não serve como sinal de "autenticado", e a saída não vai para o `stderr` — ele escreve no `stdout`. O código de saída **só distingue "não há token configurado"**.

Pior: num container com proxy de saída, o `gh auth status` imprime `X Failed to log in to github.com using token (GH_TOKEN)` enquanto `gh api user` responde o usuário correto. O veredito dele é **falso-negativo ali**.

### 6.2 O limite do ambiente

O candidato óbvio seria `gh api user`: JSON contratual, sai não-zero em 401. **Mas o caminho de falha dele é inverificável sob um proxy que reautentica:** testado com um token sintaticamente inválido, a API respondeu com o usuário certo.

É exatamente o tipo de ambiente em que um teste de caminho de falha **passa de graça** — a armadilha que o CONTRIBUTING nomeia.

### 6.3 O que fica decidido

- **A checagem só pode afirmar o que mede.** Com `gh auth status`, isso é "há token configurado", não "autenticado" — `RN-04`.
- **Token inválido, expirado ou sem escopo aparece quando o comando de PR falhar**, com a mensagem do próprio `gh`.
- **Se for pelo `gh api user`**, ela vira a primeira checagem com requisição de rede, e a promessa do README muda de "o `gtr` não faz requisição" para "nenhum comando faz sem declarar" — a mesma cirurgia que a **ADR-006** prevê.

---

## 7. Testes

**100% de statements** em `cmd`, `internal/doctor` e `internal/ui`.

O caminho de **`gh` ausente foi exercitado de verdade**, não simulado: a máquina não tinha `gh` quando a checagem foi escrita.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Tudo no lugar | Sete `ok`, saída 0, alinhado — `RF-01` |
| 2 | Sem `git` | Falha, saída 1, com o endereço de onde instalar — `RF-03` |
| 3 | `git` no `PATH` que não roda | Falha, e a **saída do próprio git** vira a dica — `RF-01` |
| 4 | Fora de um repositório | Dois pulados, saída 0, dizendo por que — `RF-02`, `RN-03` |
| 5 | Repositório sem `main`, `master` nem `origin/HEAD` | Aviso com a saída pelo `--base`, saída 0 — `RF-03` |
| 6 | O mesmo com `--strict` | Saída 1 — `RF-04` |
| 7 | Pulado com `--strict` | Saída 0: contexto ausente nunca vira falha — `RN-03` |
| 8 | `gh` ausente, com e sem `--strict` | 0 e 1 — `RN-02` |
| 9 | Sem espaço à direita, **nos dois caminhos** | E o teste confere antes que houve saída |
| 10 | `soundcheck` roda o mesmo, e não aparece na ajuda da raiz | §1.3 |
| 11 | `git` abaixo da mínima, e exatamente **na** mínima | Falha nomeando o comando que falta, e `ok` — a mínima é mínima, não exclusiva — `RF-06` |
| 12 | Versão com sufixo da Apple e do Windows, e versão ilegível | Lidas certo; ilegível vira aviso — `RF-06`, `RN-05` |
| 13 | Sem `user.name`, sem `user.email`, sem os dois, e `user.email` vazio | Aviso nomeando cada chave que falta — `RF-07`, `RN-06` |
| 14 | Base declarada pelo remoto, base chutada, e base inexistente | `ok`, aviso com `git fetch origin`, aviso com `--base` — e os três detalhes diferentes entre si — `RF-08` |
| 15 | Uma checagem falhando, e mais de uma | `1 checagem falhou` e `2 checagens falharam` — `RN-07` |
| 16 | Temporário gravável, inexistente, e **sem sobrar lixo** na sondagem | `ok` mudo, aviso com o motivo do SO e `TMPDIR`, e diretório vazio depois — `RF-09`, `RN-09` |
| 17 | Cada uma das quatro operações com ref, em tabela | Aviso nomeando a operação e as duas saídas dela — `RF-10` |
| 18 | `am` em curso; `rebase --apply` no mesmo diretório; ref e diretório ao mesmo tempo | `am`; **`ok`**; e a ref ganha — §5.8 |
| 19 | Árvore fora de um repositório | Pulada — `RN-03` |

**Cada regra nova foi conferida por mutação**, e não só por suíte verde — suíte que passa com a regra desligada não estava testando a regra. Mínima baixada para `0.0`: três testes caem. `Plural` devolvendo sempre o plural: cinco caem. Temporário sempre gravável: dois caem. Detecção de operação desligada: cinco caem.

**E o que o fake não prova foi rodado contra o git de verdade:** as cinco operações interrompidas de propósito, o `am` também **de dentro de um subdiretório**, e o `rebase --apply` para confirmar que o mesmo diretório não o confunde.

---

## 8. Rastreabilidade

| Commit | Entrega |
|---|---|
| `ca21cd9` | Nasce o `internal/doctor` e o comando, com a checagem do `git`. O apelido `soundcheck` — `RF-01`, `RF-05` |
| `696c9b3` | Repositório e base. O escopo passa a ser "funciona aqui". Nasce o estado pulado, o `--strict` e o writer que apara — `RF-02` a `RF-04`, `RN-03`, §5.2 |
| `020fdc9` | O `gh`, e o parser de versão que ele obrigou a mudar — `RN-02`, §5.1 |
| `0a1b761` | A mínima do git. A versão deixa de ser decoração e vira critério; ilegível passa a ser aviso — `RF-06`, `RN-05` |
| `a51ccd9` | A identidade, com a premissa medida contra o `commit-tree` — `RF-07`, `RN-06`, §5.4 |
| `bd6eaea` | Os comentários reescritos na forma de *doc comment* — §5.5 |
| `426c05d` | A procedência da base. Chute vira aviso — `RF-08` |
| `b2b7e6d` | O `ui.Plural`, e as nove mensagens que usavam `(s)` — `RN-07`, §5.6 |
| `3ef8a01` | Os comentários passam a declarar contrato em vez de história — `RN-08`, §5.5 |
| `5f502d6` | A sondagem do diretório de temporários — `RF-09`, `RN-09`, §5.7 |
| `22ffd25` | Operação de git no meio do caminho, as quatro por ref e o `am` pelo diretório — `RF-10`, §5.8 |

**Todos foram verificados pelo `gtr commits check` antes de ir para a `main`**, cada um sozinho, e todos verdes.

---

## 9. Follow-ups conhecidos

- **`gtr base` como comando dedicado** — considerado e **segurado de propósito**. A **ADR-004** desenha o `gtr config` para responder exatamente "por que resolveu assim, e de qual camada veio", e prevê `branches.base` como chave de configuração. Um `gtr base` hoje seria comando que o `config` engole depois. **Mas o argumento a favor não é sobre a pergunta, é sobre o custo:** para ler a base, roda-se `gtr branches` e paga-se a detecção de equivalência em todas as branches. O `doctor` já alivia isso de graça.
- **Aparar linha no `columns()` do projeto inteiro** — §5.2.
- **A checagem de repositório não distingue árvore suja, worktree, nem bare.** Se algum dia algum comando depender disso, entra aqui.
- **O temporário não mede espaço livre.** Sondar a escrita pega montagem somente leitura e permissão, mas um disco com poucos megabytes passa — e o `commits check` extrai a árvore inteira de cada commit. Medir espaço exigiria `statfs`, que é específico de sistema operacional, e um limiar que ninguém sabe fixar sem conhecer o tamanho do repositório.
- **A árvore não olha arquivo modificado, nem `git stash`, nem `bisect`.** Só operações que param no meio. **O `bisect` é o candidato mais sério**, e foi medido: durante um `git bisect start` **nenhuma das quatro refs aparece**, mas o arquivo `BISECT_START` existe — detectável, portanto, pelo mesmo par `rev-parse --git-path` mais `os.Stat` que o `am` usa. Ficou de fora porque nenhum comando do `gtr` depende disso hoje.
- **Nenhuma checagem tem tempo limite.** Todas são locais e rápidas. A de autenticação, por rede, precisa de um — e o `context` já atravessa os dois ports.
