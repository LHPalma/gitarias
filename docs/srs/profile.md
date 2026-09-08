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
fonte_externa: o GitHub, pelo gh — só com --account
---

# SRS — profile

- **Data:** 2026-08-18
- **Feature:** `cmd/profile` + `internal/profile` + `internal/forge`
- **Status:** **entregue** — `--commit-count` local, `--account` e `--by-repo`
- **Fonte externa:** **o GitHub, pelo `gh`** — só para `--account`; o `--commit-count` sozinho continua 100% local

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr profile`, que responde métricas sobre **a sua identidade de git** — hoje uma métrica só, `--commit-count`, com três fontes possíveis: este repositório (padrão), a conta GitHub inteira (`--account`), ou a conta quebrada por repositório (`--account --by-repo`).

### 1.2 Escopo

**Entregue:** `--commit-count`; `--since`/`--until` independentes, cada um com "hoje" como padrão próprio; timestamps explícitos nos limites do período. **`--account`**, contando pela conta GitHub inteira via `contributionsCollection`, com aviso de commits locais ainda não enviados. **`--by-repo`**, quebrando `--account` por repositório — cobrindo qualquer período (bissecciona janela cortada em vez de recusar) e os três formatos de saída.

**Fora de escopo:** contagem de outro autor — isso é o [`gtr stats --author`](stats.md). Mais de uma métrica — o desenho de "uma flag por métrica" segue reservado para quando houver mais. Filtrar `--account`/`--by-repo` a um único repositório — se a pergunta é "quantos commits *eu* fiz *aqui*", isso já é o `--commit-count` sem `--account`, sem rede.

### 1.3 O nome

`profile`, porque é sobre **o seu perfil** — quem você é e quanto fez —, diferente do `stats`, que é sobre todo mundo. Com `--account`, "aqui" vira "na conta inteira", mas o sujeito continua sendo você: nunca há flag para escolher outra pessoa (`RN-01`).

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/profile/                domínio local — não imprime nada
└── repo.go                      Identity, CommitCount, Unpushed, Ensure

internal/forge/                  domínio de rede, reaproveitado do pr list e do doctor
├── cli.go                       AccountCommitCount, AccountCommitCountByRepository
└── repository_commit_count.go   RepositoryCommitCount: Repository, Private, Count

cmd/profile.go                   o comando e as cinco flags
cmd/repository_commit_counts_*   record, document e table do --by-repo
```

O `internal/profile` continua só falando com `internal/git` — a rede não entrou nele. Quem fala com o GitHub é o `internal/forge` que o [`pr list`](pr-list.md) e o [`doctor --online`](doctor-online.md) já tinham; o `--account` é o terceiro consumidor do mesmo `Source`, não um port novo.

### 2.2 `--account` não é uma segunda métrica

É a mesma métrica, `--commit-count`, com a fonte trocada: local (`git log`) vira remota (`gh api graphql`). Por isso `--account` exige `--commit-count`, e `--by-repo` exige `--account` — nenhum dos dois é métrica própria.

---

## 3. Requisitos funcionais

### RF-01 — Resolver a identidade em vigor

`user.email`, com fallback para `user.name` quando o e-mail está vazio ou ausente. Nenhum dos dois configurado é erro, pedindo para configurar um dos dois. **Só vale no modo local** — o `--account` usa a identidade que o `gh` autentica, nunca esta.

### RF-02 — Contar commits da identidade num período

Conta commits de autoria da identidade resolvida entre `since` e `until`, **os dois dias inclusos**, no histórico do `HEAD` atual.

### RF-03 — `--since`/`--until` independentes

Cada flag tem "hoje" como padrão **próprio**: nenhuma das duas dadas é hoje..hoje; só `--since` é `since`..hoje; só `--until` é hoje..`until`. Vale para os três modos.

### RF-04 — `--commit-count` é obrigatória

É a única métrica hoje, e por isso obrigatória: ausência é erro nomeando a flag esperada, antes de tocar no git ou na rede.

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

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **A identidade é sempre a de quem está autenticado, nunca um autor arbitrário.** Ao contrário do `stats`, não há flag para escolher outra pessoa. No modo local é `user.email`/`user.name`; com `--account`, é quem o `gh` diz que somos. |
| **RN-02** | **`since` e `until` viram meia-noite e `23:59:59` explícitas antes de chegar ao git ou à API, nunca a data nua.** Medido contra o git de verdade: `--since=<data>` sozinho vale a hora corrente de agora, não meia-noite. A mesma disciplina vale para `--account`, no fuso local. |
| **RN-03** | **Chave de config setada como string vazia conta como ausente** — mesmo raciocínio do [`doctor`](doctor.md) para a identidade. |
| **RN-04** | **Contagem ilegível é erro, não zero silencioso** — vale para o `rev-list` local e para a resposta do `gh`. |
| **RN-05** | **Validação de flag e de formato de data vem sempre antes de qualquer chamada** — ao git ou à rede. |
| **RN-06** | **A soma de `--account` vale o que o token do `gh` consegue ler.** Sem o escopo `read:user`, o GitHub omite contribuições de repositório privado da soma — **caladas, sem erro** que distinga "poucos commits mesmo" de "faltou permissão". O [`gtr doctor --online`](doctor-online.md) tem checagem própria para isso. |
| **RN-07** | **Resposta cortada nunca vira lista incompleta silenciosa.** O `commitContributionsByRepository` ordena por contribuição e trunca no teto; comparar `len(listado)` com `totalRepositoriesWithContributedCommits` detecta o corte, e ele vira bisecção (`RF-12`), nunca dado perdido calado. |

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

---

## 6. Testes

**100% de statements** em `internal/profile`, `internal/forge` e `cmd/profile.go`, mais os arquivos de tabela do `--by-repo`.

| Área | Cobertura |
|---|---|
| `--commit-count` local | 15 cenários — datas, identidade, contagem, erros |
| `--account` | Total; janela de rede; ausência do `gh`; recusa do `gh`; aviso de commits locais, com e sem upstream; janelas somadas quando o período passa de um ano; falha propagada em cada janela |
| `--by-repo` | Listagem; ordenação por contagem; empate por nome; os três formatos; flags de formato recusadas sem `--by-repo`; resposta cortada virando erro em vez de lista incompleta; merge entre janelas; bisecção com sucesso, com falha na primeira metade, com falha na segunda, sem sobreposição entre metades; desistência só depois de bisseccionar até uma hora |

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

---

## 8. Follow-ups conhecidos

- **Só uma métrica hoje.** O desenho de "uma flag por métrica" segue pronto para crescer; `--account` e `--by-repo` são modo de fonte da métrica existente, não uma nova.
- **`--account` e `--by-repo` não filtram por repositório.** Cogitado e descartado: quem quer só o repositório atual já tem `--commit-count` sem `--account`, sem rede — acrescentar um filtro duplicaria esse caminho.
- **Sem paginação de verdade para `--by-repo`.** A bisecção existe porque o `commitContributionsByRepository` não oferece cursor — não é escolha entre duas formas de paginar, é a única saída disponível. Registrado para quem for mexer de novo não achar que faltou implementar o cursor.
