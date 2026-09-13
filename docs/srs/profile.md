---
titulo: SRS — profile
data: 2026-08-18
status: entregue
comando: gtr profile
pacotes:
  - cmd/profile.go
  - cmd/repository_commit_counts_table.go
  - cmd/repository_commit_counts_document.go
  - internal/profile
  - internal/forge
commits:
  - 071cea1
  - f4a80dd
  - 2b7899d
  - 190f7c6
  - 68bcb20
  - 9f3f784
  - 17d07b5
  - a358602
  - d945615
  - f4bd032
  - 9a81ac4
  - 74a94c2
  - b036315
  - d6535cb
  - bcbca4b
  - f076fad
  - 1a51939
  - 447832b
  - 3d8718f
  - cb9bae2
  - 6a6447e
fonte_externa: o GitHub, pelo gh — só com --account
---

# SRS — profile

- **Data:** 2026-08-18
- **Feature:** `cmd/profile` + `internal/profile` + `internal/forge`
- **Status:** **entregue** — `--commit-count` local, `--account`, `--by-repo` e `--streak`
- **Fonte externa:** **o GitHub, pelo `gh`** — só para `--account`; o `--commit-count` sozinho continua 100% local

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr profile`, que responde métricas sobre **a sua identidade de git** — hoje duas, que não se combinam:

- **`--commit-count`**, quantos commits no período, com três fontes possíveis: este repositório (padrão), a conta GitHub inteira (`--account`), ou a conta quebrada por repositório (`--account --by-repo`).
- **`--streak`**, quantos dias seguidos com commit — a sequência em curso e a maior do histórico. Só local, e sem período: a pergunta olha o histórico inteiro.

### 1.2 Escopo

**Entregue:** `--commit-count`; `--since`/`--until` independentes, cada um com "hoje" como padrão próprio; timestamps explícitos nos limites do período. **`--account`**, contando pela conta GitHub inteira via `contributionsCollection`, com aviso de commits locais ainda não enviados. **`--by-repo`**, quebrando `--account` por repositório — cobrindo qualquer período (bissecciona janela cortada em vez de recusar) e os três formatos de saída. **`--streak`**, a sequência de dias seguidos com commit, em curso e a maior do histórico, com **`--author`** para apurar a de outra pessoa. **`--by-hour`** e **`--by-weekday`**, quebrando a contagem local por hora do dia e por dia da semana.

**Fora de escopo:** contagem de outro autor no `--commit-count` — isso é o [`gtr stats --author`](stats.md). Filtrar `--account`/`--by-repo` a um único repositório — se a pergunta é "quantos commits *eu* fiz *aqui*", isso já é o `--commit-count` sem `--account`, sem rede. **A sequência da conta inteira (`--streak --account`)** — o desenho está acordado e registrado no §8, e não saiu porque não houve como medir a consulta por dia contra a API real. **Formatos estruturados para o `--streak`** — a saída é de duas linhas, e não há tabela para `csv` ou `json` formatarem.

### 1.3 O nome

`profile`, porque é sobre **o seu perfil** — quem você é e quanto fez —, diferente do `stats`, que é sobre todo mundo. Com `--account`, "aqui" vira "na conta inteira", e o sujeito continua sendo você.

**A regra de "sempre você" valia inteira até o `--streak`, e agora vale pela metade** — ver `RN-01`. O `--author` escolhe outro sujeito, e só existe na métrica onde a alternativa seria pior: sem ele, ler a sequência de outra pessoa exigiria trocar a identidade configurada do repositório.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/profile/                domínio local — não imprime nada
├── repo.go                      Identity, CommitCount, Unpushed, Streaks, Ensure
├── streak.go                    Streak: dias, primeiro e último
└── streak_report.go             StreakReport: a em curso, a maior e o último dia

internal/forge/                  domínio de rede, reaproveitado do pr list e do doctor
├── cli.go                       AccountCommitCount, AccountCommitCountByRepository
└── repository_commit_count.go   RepositoryCommitCount: Repository, Private, Count

cmd/profile.go                   o comando e as sete flags
cmd/repository_commit_counts_*   record, document e table do --by-repo
```

O `internal/profile` continua só falando com `internal/git` — a rede não entrou nele. Quem fala com o GitHub é o `internal/forge` que o [`pr list`](pr-list.md) e o [`doctor --online`](doctor-online.md) já tinham; o `--account` é o terceiro consumidor do mesmo `Source`, não um port novo.

### 2.2 `--account` não é uma segunda métrica

É a mesma métrica, `--commit-count`, com a fonte trocada: local (`git log`) vira remota (`gh api graphql`). Por isso `--account` exige `--commit-count`, e `--by-repo` exige `--account` — nenhum dos dois é métrica própria.

### 2.3 `--streak` é métrica; `--account` e `--by-repo` continuam sendo fonte

Os dois eixos são independentes e não se misturam: **métrica** é o que se pergunta (`--commit-count`, `--streak`), **fonte** é de onde vem a resposta (aqui, ou a conta), e **recorte** é por qual eixo a resposta é quebrada (`--by-repo`, `--by-hour`, `--by-weekday`).

**Os três recortes são a mesma operação em eixos diferentes**, e é por isso que nenhum deles é métrica própria: quebram a contagem do `--commit-count` por repositório, por hora ou por dia da semana. Um por invocação — cada um devolve **uma** tabela, e é isso que dá a eles os quatro formatos que o `--streak` não pode ter. Uma métrica por invocação, e cada flag de fonte ou de recorte pertence à métrica que a usa — daí `--since`/`--until`/`--account` valerem só com `--commit-count`, e `--author` só com `--streak`.

Nenhuma delas é descartada calada quando vem na métrica errada: é erro, antes de tocar no git (`RF-18`).

---

## 3. Requisitos funcionais

### RF-01 — Resolver a identidade em vigor

`user.email`, com fallback para `user.name` quando o e-mail está vazio ou ausente. Nenhum dos dois configurado é erro, pedindo para configurar um dos dois. **Só vale no modo local** — o `--account` usa a identidade que o `gh` autentica, nunca esta.

### RF-02 — Contar commits da identidade num período

Conta commits de autoria da identidade resolvida entre `since` e `until`, **os dois dias inclusos**, no histórico do `HEAD` atual.

### RF-03 — `--since`/`--until` independentes

Cada flag tem "hoje" como padrão **próprio**: nenhuma das duas dadas é hoje..hoje; só `--since` é `since`..hoje; só `--until` é hoje..`until`. Vale para os três modos.

### RF-04 — Uma métrica é obrigatória

Nenhuma métrica é erro nomeando as esperadas, antes de tocar no git ou na rede. Era `--commit-count` sozinha até a sequência chegar; hoje são duas, e a exigência virou "uma delas" — ver `RF-17`.

### RF-05 — Validar o formato das datas antes do git

`--since`/`--until` fora do formato `AAAA-MM-DD` são recusadas antes de qualquer chamada.

### RF-06 — Repositório sem histórico

Um repositório onde o `HEAD` ainda não aponta para nenhum commit devolve contagem zero, não erro.

### RF-07 — Mensagem varia com o período

Singular e plural via `ui.Plural`; `"N commits em <data>"` quando `since == until`, `"N commits entre <since> e <until>"` quando são diferentes. Vale para `--account` também.

### RF-08 — `--account` conta pela conta GitHub inteira

Substitui o `git log` local por `contributionsCollection.totalCommitContributions`, via `gh`. **Faz chamada de rede.** Período maior que um ano é quebrado em janelas de até 365 dias e somado — a API não aceita janela maior numa consulta só.

### RF-09 — `--account` avisa sobre commits locais não enviados

Depois de imprimir a soma da conta, se o `HEAD` deste repositório estiver à frente do `@{u}` configurado, avisa quantos commits — esses não entraram na soma, que só enxerga o que já chegou ao GitHub. Sem upstream configurado, o aviso fica de fora: não há com o que comparar.

### RF-10 — `--by-repo` quebra `--account` por repositório

Requer `--account`; sozinho é erro nomeando a exigência, antes de tocar em qualquer coisa. Lista cada repositório com contribuição no período, ordenado por contagem decrescente, empate desempatado por nome.

### RF-11 — `--by-repo` cobre qualquer período

Como o `--account` sozinho, quebra período maior que um ano em janelas — mas aqui isso não basta, porque o mesmo repositório pode aparecer em janelas diferentes: as janelas se somam por `nameWithOwner` antes de sair.

### RF-12 — `--by-repo` bissecciona uma janela cortada

`commitContributionsByRepository` tem teto de 100 repositórios por consulta e **não pagina** — sem `after`/`before`, só `maxRepositories`. Quando uma janela vem cortada, é dividida ao meio e cada metade tenta de novo, recursivamente, até caber ou até a janela chegar a uma hora — só aí desiste, com erro nomeando a janela e o total real.

### RF-13 — `--by-repo` aceita os três formatos de saída

`text`, `csv` e `json`, mais `--no-header` e `--output`. Essas flags só valem com `--by-repo`; setadas sem ele, erro — nenhuma tabela existe para formatar.

### RF-14 — Apurar a sequência de dias com commit

`--streak` devolve, no histórico do `HEAD` atual, **a sequência em curso** e **a maior de todo o histórico**, cada uma com o número de dias e as datas de início e fim. Vários commits no mesmo dia contam **um dia só**: a unidade é o dia, não o commit.

### RF-15 — O dia corrente não quebra a sequência

Sem commit hoje, a sequência em curso **termina ontem e continua valendo**, e a saída diz o que falta (`— ainda sem commit hoje`). Ela só quebra quando ontem também não teve — e aí a linha nomeia o último dia com commit, em vez de dizer só "nenhuma".

O dia que ainda está correndo não é um dia perdido: seria punir quem ainda vai commitar à noite.

### RF-16 — `--author` apura a sequência de outra pessoa

Casa por substring, como o `--author` do próprio `git log`. Sem ela, o sujeito é a identidade resolvida pela `RF-01`; **com ela, a identidade local nem chega a ser lida** — não é dela que se está falando, e lê-la seria uma chamada de git sem propósito.

Vale só com `--streak` (`RF-18`).

### RF-17 — Uma métrica por invocação

Nenhuma métrica é erro nomeando as duas esperadas; **as duas juntas também são erro**. `--commit-count` e `--streak` respondem perguntas de forma diferente — uma contagem e duas sequências —, e não há saída que seja as duas coisas.

### RF-18 — Flag de uma métrica é recusada na outra

`--since`, `--until` e `--account` com `--streak`; `--author` com `--commit-count`; as flags de formato fora de `--by-repo`. Todas viram erro **antes de tocar no git ou na rede**, nomeando a métrica a que pertencem.

A sequência não tem período a escolher de propósito: ela olha o histórico inteiro, e um `--since` ali responderia outra pergunta — "a maior sequência dentro da janela" —, que ninguém pediu.

### RF-19 — Sem commit não é erro

Repositório onde o `HEAD` não aponta para nenhum commit, ou autor sem nenhum commit no histórico, devolve `Nenhum commit encontrado.` — a mesma frase do [`stats`](stats.md), e nunca uma sequência de zero dias com datas inventadas. No repositório vazio o `git log` nem chega a ser chamado.

### RF-20 — `--by-hour` e `--by-weekday` quebram a contagem local

Recortes do `--commit-count`, um por invocação, **sempre locais**: `--by-hour` devolve as 24 horas do dia; `--by-weekday`, os sete dias da semana, da segunda ao domingo. Ambos aceitam os quatro formatos, porque ambos são tabela de verdade.

### RF-21 — O recorte herda o período, inclusive o padrão

`--since`/`--until` valem igual, e sem nenhuma das duas o período é **hoje** — o recorte muda o eixo da resposta, nunca a pergunta. Quebrar o dia de hoje por hora é resposta legítima; a pergunta ampla se pede com `--since`.

**Decisão consciente contra o açúcar:** o padrão do período não muda por causa de outra flag estar ligada. Padrão que se desloca conforme a combinação é o tipo de conveniência que custa previsibilidade.

### RF-22 — `--account` não aceita os dois recortes

A contagem da conta vem do `contributionsCollection`, que traz total por repositório e **não traz hora nem dia da semana**. A combinação é recusada com a razão nomeada, em vez de devolver tabela vazia ou zerada.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **A identidade é a de quem está autenticado — e no `--commit-count` isso não se escolhe.** No modo local é `user.email`/`user.name`; com `--account`, é quem o `gh` diz que somos, e ali não há como ser outra pessoa: quem autentica é o `gh`. **A exceção é o `--streak --author`** (`RF-16`), aberta em 2026-09-13 por decisão explícita do autor, e deliberadamente **não** estendida ao `--commit-count` — para contar commits de um autor qualquer, o comando continua sendo o [`gtr stats --author`](stats.md). |
| **RN-02** | **`since` e `until` viram meia-noite e `23:59:59` explícitas antes de chegar ao git ou à API, nunca a data nua.** Medido contra o git de verdade: `--since=<data>` sozinho vale a hora corrente de agora, não meia-noite. A mesma disciplina vale para `--account`, no fuso local. |
| **RN-03** | **Chave de config setada como string vazia conta como ausente** — mesmo raciocínio do [`doctor`](doctor.md) para a identidade. |
| **RN-04** | **Contagem ilegível é erro, não zero silencioso** — vale para o `rev-list` local e para a resposta do `gh`. |
| **RN-05** | **Validação de flag e de formato de data vem sempre antes de qualquer chamada** — ao git ou à rede. |
| **RN-06** | **A soma de `--account` vale o que o token do `gh` consegue ler.** Sem o escopo `read:user`, o GitHub omite contribuições de repositório privado da soma — **caladas, sem erro** que distinga "poucos commits mesmo" de "faltou permissão". O [`gtr doctor --online`](doctor-online.md) tem checagem própria para isso. |
| **RN-07** | **Resposta cortada nunca vira lista incompleta silenciosa.** O `commitContributionsByRepository` ordena por contribuição e trunca no teto; comparar `len(listado)` com `totalRepositoriesWithContributedCommits` detecta o corte, e ele vira bisecção (`RF-12`), nunca dado perdido calado. |
| **RN-08** | **O dia da sequência é o da autoria, no fuso de quem roda** — `%ad` com `--date=short-local`. É o dia que a pessoa viveu, e é o que **sobrevive a um rebase**, que preserva a data de autoria e reescreve a de commit. Ler a data de commit faria uma reescrita de histórico apagar ou inventar sequência. |
| **RN-09** | **A aritmética de dia acontece sobre a data civil à meia-noite UTC, nunca sobre hora local.** Onde o horário de verão entra à meia-noite, o dia anterior de uma meia-noite local **não é meia-noite**, e somar ou subtrair um dia cairia num horário que o fuso não tem. O fuso entra uma vez só, no `--date=short-local` do git; daí para frente são datas civis. |
| **RN-10** | **Commit datado no futuro não abre sequência.** Relógio adiantado de quem commitou não inventa constância: o dia futuro é pulado na apuração da sequência em curso — mas segue contando para a maior e para o último dia, porque lá ele é fato do histórico, não afirmação sobre hoje. |
| **RN-11** | **Empate na maior sequência fica com a mais recente.** Duas sequências do mesmo tamanho desempatam por data, não por ordem de leitura: a recente é a que diz mais sobre o hábito atual. |
| **RN-12** | **O domínio não lê o relógio.** `Streaks` recebe o dia de referência de quem chama, e o `time.Now()` mora no `cmd`, onde o processo toca o mundo — sem isso, a apuração dependeria da data em que o teste roda. O `cmd` lê o relógio **uma vez só** por invocação: duas leituras podem cair em dias diferentes se a chamada atravessar a meia-noite, e a sequência seria apurada contra um dia e descrita contra outro. |
| **RN-13** | **Data ilegível é erro, não sequência inventada** — o par da `RN-04` do lado da sequência. Linha vazia é ignorada; linha que não casa com `AAAA-MM-DD` interrompe com erro nomeando o valor lido. |
| **RN-15** | **A hora e o dia da semana são os do fuso de quem roda, e o fuso entra uma vez só** — no `--date=iso-strict-local`, que o git resolve. O instante chega em RFC 3339 com deslocamento embutido, então ler hora e dia dele já devolve o local, sem segunda conversão. Mesma disciplina da `RN-09`. |
| **RN-16** | **O recorte devolve todos os baldes, inclusive os zerados** — as 24 horas, os sete dias. A forma da distribuição é a resposta, e balde ausente obrigaria quem lê a contar linha para descobrir o que faltou. |
| **RN-17** | **A semana começa na segunda, não no domingo do `time.Weekday`.** A pergunta é sobre hábito de trabalho, e o fim de semana diz mais junto, no fim da tabela, do que partido entre as duas pontas. A ordem é explícita no domínio, nunca a do enum. |
| **RN-18** | **O que vai para a planilha não é o que vai para a tela, e a régua é "grandeza ou rótulo".** A hora sai crua no `csv`/`json` (`22`, não `22h`) porque é número que ordena e soma — mesma decisão do `ui.Bytes`. O dia da semana sai pelo nome em **todos** os formatos porque é rótulo: `3` obrigaria quem abre o arquivo a saber onde a semana começa. |
| **RN-14** | **A ordem de leitura do `git log` não é confiável para data de autoria.** Ele ordena por data de **commit**, e rebase, cherry-pick e amend deslocam uma sem a outra — por isso os dias são ordenados no domínio antes de virar sequência. A ordenação parece redundante e não é. |

---

## 5. Achados de implementação

### 5.1 A semântica de data nua foi medida, não suposta

`--since=2026-08-15` sozinho vale a hora corrente de agora naquele dia, não meia-noite — medido contra o git antes de decidir. Por baixo, `--since` vira `<data> 00:00:00` e `--until` vira `<data> 23:59:59`, sempre.

### 5.2 O teste da função de período caiu na mesma armadilha que ela corrige

`TestProfileUntilAloneStartsAtToday` usava uma data futura cravada; quando o relógio real a alcançou, o teste quebrou pelo mesmo motivo que a `RN-02` já corrigia no produto. Trocado por `profileYesterday()`, que nunca coincide com "hoje" de verdade.

### 5.3 O "5 contra 254" é escopo de token, não bug de consulta

A primeira tentativa de `--account` devolveu **5** commits onde a tela de atividade do GitHub mostrava **254**. A causa não foi a consulta: foi o token do `gh` não carregar o escopo `read:user`.

Sem ele, o GitHub omite contribuições de repositório privado da soma, **silenciosamente**, mesmo com "Include private contributions" ligado no perfil — esse ajuste só afeta o que aparece para *outras pessoas*, não o que a API devolve para o próprio dono via token.

Depois de `gh auth refresh -s read:user`, a soma bateu exatamente com a tela: 254 contra 254, por repositório inclusive. A checagem `escopo` do [`gtr doctor --online`](doctor-online.md) nasce direto dessa descoberta.

### 5.4 `commitContributionsByRepository` não pagina

Descoberto por introspecção do schema GraphQL (`__type(name: "ContributionsCollection")`): o campo só aceita `maxRepositories`, com teto de 100 — confirmado contra a API real, que devolve um erro `ARGUMENT_LIMIT` nomeado ao pedir mais. Não há `after` nem `before`.

Isso descartou de vez um "incremento de paginação" cogitado antes de medir, e obrigou o desenho da bisecção (`RF-12`) como única saída para não recusar todo período ativo.

### 5.5 A bisecção nunca precisou disparar na prática

Testada com sucesso, com falha na primeira metade, com falha na segunda, e com a garantia de que as duas metades nunca se sobrepõem — a segunda começa exatamente um segundo depois do fim da primeira.

Contra uma conta real com 21 a 22 repositórios ativos por ano, bem abaixo do teto de 100, a bisecção nunca chegou a rodar: o caminho comum, uma consulta por janela, continua sendo o único que roda de verdade.

### 5.6 `--account` não é escopado ao repositório atual, de propósito

Uma contagem de 1340 commits rodando dentro de um repositório específico causa estranheza, e não é bug: o `--account` sempre soma a conta inteira, em todos os repositórios. Rodar dentro de um repositório não filtra nada, porque a pergunta que ele responde é sobre a conta, não sobre o diretório atual.

Confirmado quebrando a mesma consulta por `--by-repo`: **380** dos 1340 eram daquele repositório, o resto de outros vinte. Quem quer só o repositório atual usa `--commit-count` sem `--account`.

### 5.7 O exemplo inventado no README mentiu antes de qualquer código errar

A primeira versão da documentação do `--streak` trazia uma saída de exemplo escrita à mão:

```text
Sequência atual: 12 dias, de 2026-09-02 a 2026-09-13.
Maior sequência: 31 dias, de 2026-08-01 a 2026-08-31.
```

Os números são internamente coerentes — descrevem um histórico sem commit no dia 1º de setembro —, mas **desenham exatamente a aparência de um bug**: a sequência morre em 31/08 e recomeça em 02/09, como se a virada do mês zerasse o contador. O autor leu, reconheceu que provavelmente commitara no dia 1º, e reportou como defeito.

Não era. A conta atravessa mês, ano e bissexto, e isso foi confirmado por dois caminhos: um repositório de verdade com commits de 29/08 a 02/09, que devolve os 5 dias inteiros (e cai para 3, quebrando em 31/08, quando só o dia 1º é removido), e a varredura de §6 sobre quatro anos de datas.

**O achado é sobre documentação, não sobre o código:** exemplo inventado é afirmação sobre comportamento, e mente com a mesma força que código errado — com o agravante de que nenhum teste o cobre. Os exemplos do `--commit-count` que já estavam no README são inócuos porque não têm estrutura interna; os da sequência têm, porque as datas se relacionam entre si. **Exemplo com estrutura interna sai de execução real.**

### 5.8 A aritmética de dia foge do fuso em vez de tratá-lo

O primeiro desenho manipulava dias como meia-noite **local**. Isso quebra onde o horário de verão entra à meia-noite — o Brasil fez exatamente isso até 2019 —, porque `AddDate(0, 0, -1)` sobre uma meia-noite que não existe devolve outra hora, e a comparação de igualdade entre dias falha.

A saída foi não tratar o caso: o fuso entra **uma vez só**, no `--date=short-local` que o git resolve, e daí para frente o dia é data civil à meia-noite **UTC**, onde não há horário de verão nenhum para atrapalhar (`RN-09`). Nenhuma tabela de fusos, nenhuma dependência — e o caso de borda deixa de existir em vez de ser coberto.

### 5.9 Em clone raso a sequência subconta calada

Medindo a sequência real do autor neste repositório, a resposta foi **2 dias** de máxima. O clone do container era **raso**: 146 commits, começando em 22/08. Depois de `git fetch --unshallow`, com os 277 commits, a mesma pergunta devolveu **7 dias** — errado por um fator de três e meio, sem nada na saída denunciando.

É o mesmo modo de falha da `RN-06` (a soma que vale o que o token lê), agora do lado local: o `--streak` afirma sobre o histórico inteiro e recebe um histórico truncado. O git responde `rev-parse --is-shallow-repository`, então **dá para avisar** — registrado no §8, não implementado.

### 5.10 O formato de data foi medido antes de escolher, e o escolhido é o que a stdlib já conhece

Três candidatos foram rodados contra o git de verdade antes de decidir:

```text
--date=iso-local           2026-09-13 16:56:38 +0000
--date=iso-strict-local    2026-09-13T16:56:38+00:00
--date=format-local:%H     16
```

O escolhido é o `iso-strict-local`, por dois motivos. É **RFC 3339 exato**, então a leitura usa `time.RFC3339` da stdlib em vez de um layout escrito à mão — uma coisa a menos para digitar errado. E evita o `format-local:`, que passa por `strftime` e é onde diferença entre plataformas costuma aparecer — o projeto já foi mordido por formato de saída em Windows duas vezes (o `THIRD-PARTY-LICENSES` em CRLF, o `status --porcelain` v1).

O `-local` foi conferido honrando o `TZ` do ambiente: o mesmo commit sai `16:56:38 +0000` e `13:56:38 -0300` conforme o fuso, que é o que a `RN-15` promete.

---

## 6. Testes

**100% de statements** em `internal/profile`, `internal/forge` e nos arquivos de tabela do `--by-repo` — a sequência incluída, domínio e comando.

**Em `cmd/profile.go` falta um ponto:** o `runAccountCommitCountByRepository` mede 91,7%. É o caminho mais ramificado do comando — janela, merge entre janelas e bisecção — e o que sobra descoberto ali são ramos de erro da apresentação, não da contagem.

| Área | Cobertura |
|---|---|
| `--commit-count` local | 15 cenários — datas, identidade, contagem, erros |
| `--account` | Total; janela de rede; ausência do `gh`; recusa do `gh`; aviso de commits locais, com e sem upstream; janelas somadas quando o período passa de um ano; falha propagada em cada janela |
| `--by-repo` | Listagem; ordenação por contagem; empate por nome; os três formatos; flags de formato recusadas sem `--by-repo`; resposta cortada virando erro em vez de lista incompleta; merge entre janelas; bisecção com sucesso, com falha na primeira metade, com falha na segunda, sem sobreposição entre metades; desistência só depois de bisseccionar até uma hora |
| `--streak` | 12 cenários de apuração — sequência terminando hoje e terminando ontem; quebra com dois dias sem commit; dias repetidos contando um só; data de autoria fora de ordem; empate resolvido pela mais recente; virada de ano; dia no futuro, sozinho e junto de hoje; um dia só; autor sem commit. Mais: `--author` chegando ao `git log`; hora da referência descartada; repositório vazio sem chamar o log; data ilegível; falhas propagadas; cancelamento. No `cmd`, as duas linhas de saída em cada forma, `--author` não lendo a identidade, e as sete recusas de flag |
| Virada de mês | `TestStreaksAcrossEveryMonthBoundary`: sequências de 1 a 6 dias terminando em **cada dia de quatro anos**, bissexto incluído — 8.760 casos, 0,02s |

| `--by-hour` / `--by-weekday` | Baldes com contagem e os zerados presentes; ordem da tabela; a semana começando na segunda; a hora lida do deslocamento que veio do git; o dia sem commit nenhum; repositório vazio sem chamar o log; instante ilegível; falhas propagadas nos dois recortes; as seis recusas de combinação; `csv` com a hora crua e `json` com o nome do dia; nenhuma linha terminando em espaço; falha de escrita nos dois |
| `ui.DescribeWeekday` | Os sete rótulos, um subteste cada — `switch` de rótulo sem caso a caso deixa passar troca entre dois deles |

**A varredura não passa de graça.** A mutação que a motivou — comparar o dia do mês (`days[next].Day() != start.Day()-1`) em vez da data inteira — reprova nela, que é o modo de falha do §5.7.

**Mais oito mutantes plantados e mortos** na entrega dos recortes: `--date=iso-strict` no lugar de `iso-strict-local`; hora marcando em vez de acumular; sábado e domingo trocados na ordem da semana; a tabela de horas nascendo vazia em vez das 24; o `csv` levando `22h` em vez de `22`; um rótulo de dia trocado por outro; os dois recortes juntos aceitos; recorte local aceito junto de `--account`.

**Oito mutantes plantados e mortos** na entrega da sequência: exigir commit hoje para a sequência valer; empate na maior indo para a mais antiga; dia no futuro zerando em vez de ser pulado; `--date=short` no lugar de `short-local`; remover a ordenação por data de autoria; `1 dia` repetindo a data nos dois extremos; o aviso de "ainda sem commit hoje" sempre ligado; `--author` aceito fora do `--streak`.

Testado contra o GitHub e o `gh` reais nos três modos, não só com `gh` roteirizado — inclusive o período de dois anos que motivou a bisecção, rodado num repositório de terceiros com histórico de fato longo.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `071cea1`, `f4a80dd`, `2b7899d` | `--commit-count` local — domínio, comando e a correção do teste que dependia do relógio — §5.1, §5.2 |
| `190f7c6` | `internal/forge.CLI.AccountCommitCount` e `internal/profile.Repo.Unpushed` — `RF-08`, `RF-09` |
| `68bcb20` | `gtr profile --account` — o comando |
| `9f3f784` | README do `--account` |
| `17d07b5` | `internal/forge.CLI.AccountCommitCountByRepository` — `RF-10` |
| `a358602` | `gtr profile --account --by-repo` — o comando e as tabelas de saída — `RF-13` |
| `d945615` | README do `--by-repo` |
| `f4bd032`, `9a81ac4`, `74a94c2` | Merge entre janelas — domínio, comando, README — `RF-11`, §5.4 |
| `b036315`, `d6535cb`, `bcbca4b` | Bisecção de janela cortada — domínio, comentários do comando, README — `RF-12`, §5.4, §5.5 |
| `f076fad` | `internal/profile.Repo.Streaks`, `Streak` e `StreakReport` — a apuração — `RF-14`, `RF-15`, `RF-19`, `RN-08` a `RN-14` |
| `1a51939` | `gtr profile --streak` e `--author` — a segunda métrica, as recusas entre métricas — `RF-16`, `RF-17`, `RF-18`, `RN-01` revisada |
| `447832b` | README da sequência, com os exemplos vindos de execução real — §5.7 |
| `3d8718f` | `internal/profile.CommitCountByHour` e `CommitCountByWeekday`, mais `ui.DescribeWeekday` — os baldes, a ordem da semana, a data medida — `RF-20`, `RN-15` a `RN-17`, §5.10 |
| `cb9bae2` | `gtr profile --by-hour` e `--by-weekday` — os recortes, as recusas, as duas tabelas nos quatro formatos — `RF-21`, `RF-22`, `RN-18` |
| `6a6447e` | README dos recortes |

---

## 8. Follow-ups conhecidos

- **`--streak --account`: a sequência da conta inteira — desenhada, não medida.** É o número que de fato interessa a quem trabalha em vários repositórios: a sequência *deste* repositório quase sempre subconta a constância real. **O desenho já está acordado** e cabe no vocabulário existente, sem flag mudando de sentido: `--streak` é aqui; `--streak --account` é a conta; `--streak --account --by-repo` é a conta quebrada por repositório, com o `--by-repo` mantendo o sentido de detalhamento que já tem. **Descartada a forma proposta primeiro** — global por padrão, com uma flag para filtrar —, por dois motivos: tornaria a rede obrigatória para a pergunta mais básica, contra o "local primeiro, rede só declarada" do projeto; e daria ao `--by-repo` um segundo sentido, o de filtro, dependendo da métrica ligada.
  A fonte certa é `contributionsCollection.commitContributionsByRepository → contributions.nodes { occurredAt commitCount }`, que é por dia, por repositório e **só commit** — o `contributionCalendar` também tem dado por dia, mas conta issue, PR e review junto, o que seria outra métrica. Três semânticas precisariam virar regra escrita antes de sair: a sequência da conta só enxerga o que **chegou ao GitHub** (dia com commit local sem push não conta, e passa a contar no instante do push — o aviso da `RF-09` importa mais aqui, porque a unidade é o dia); a fronteira do dia passa a ser a do GitHub, não o `--date=short-local`, e perto da meia-noite as duas respostas divergem legitimamente; e `contributions(first:)` tem teto por repositório, um **segundo eixo de corte** além do de cem repositórios que a `RF-12` já trata. **Não saiu porque não houve como medir**: o `gh` não está instalado no container da sessão, e o proxy recusa GraphQL fora do conjunto fixo de operações de revisão de PR. Implementar sem medir contrariaria a regra do projeto.
- **Clone raso subconta a sequência, calado** — §5.9. O git responde `rev-parse --is-shallow-repository`; o `--streak` poderia avisar em vez de afirmar sobre um histórico que não tem, na mesma linha do aviso de commits não enviados da `RF-09`. Vale também para o `--commit-count` local, com efeito menor: ali o período costuma ser curto, e o corte fica longe.
- **`--author` só existe no `--streak`.** Estendê-lo ao `--commit-count` — e, por tabela, aos recortes `--by-hour`/`--by-weekday`, que são dele — é uma linha, e foi deixado de fora de propósito: mudaria o comportamento de uma métrica já entregue, sem pedido. É a extensão mais provável de ser pedida a seguir: "a que horas *fulano* commita" é pergunta natural, e hoje exige trocar a identidade configurada do repositório. Enquanto isso, a `RN-01` vale pela metade, o que é dívida de coerência registrada, não desenho final.
- **A sequência não tem formato estruturado.** Duas linhas de texto, sem `csv`/`json` — não há tabela a formatar. Se um dia houver (uma linha por sequência, ou a lista de dias), o caminho é o mesmo trio `record`/`document`/`table` do resto do projeto.
- **Só uma métrica de contagem hoje.** O desenho de "uma flag por métrica" segue pronto para crescer; `--account` e `--by-repo` são modo de fonte do `--commit-count`, não uma métrica nova.
- **`--account` e `--by-repo` não filtram por repositório.** Cogitado e descartado: quem quer só o repositório atual já tem `--commit-count` sem `--account`, sem rede — acrescentar um filtro duplicaria esse caminho.
- **Sem paginação de verdade para `--by-repo`.** A bisecção existe porque o `commitContributionsByRepository` não oferece cursor — não é escolha entre duas formas de paginar, é a única saída disponível. Registrado para quem for mexer de novo não achar que faltou implementar o cursor.
