---
titulo: "SRS — a conexão: doctor --online"
data: 2026-08-17
status: entregue
comando: gtr doctor --online
apelido: --plugged
pacotes:
  - internal/forge
  - internal/doctor/connected.go
  - cmd/network.go
commits:
  - 06f20fa
  - 80869ff
  - c18c27e
  - ae18bfa
  - 96fa236
fonte_externa: o GitHub, através do gh
---

# SRS — a conexão: doctor --online

- **Data:** 2026-08-17
- **Feature:** `internal/forge` + `internal/doctor/connected.go` + `gtr doctor --online`
- **Status:** **entregue** — a conexão e o escopo do token
- **Fonte externa:** **o GitHub** — e este é o primeiro documento do projeto em que essa linha não diz "nenhuma"

---

## 1. Introdução

### 1.1 Propósito

Especificar **a entrada da rede no `gtr`**: o port `internal/forge`, a checagem de conexão, e a flag `gtr doctor --online` que a liga. A primeira aplicação dessa entrada — o `gtr pr list` — tem documento próprio.

### 1.2 A promessa mudou de forma, e isso é o mais importante aqui

A promessa era *"não pede token, não faz requisição de rede e não escreve em repositório remoto"*. Passa a ser *"nenhum comando sai da máquina sem dizer que sai"*, e os que saem declaram no próprio `--help`.

**Mas o que fica no lugar é mais forte que o que saiu:** nenhum deles pede token. **Nenhuma credencial passa pelo `gtr`** — nem por variável de ambiente, nem por `argv`, nem em memória. "Não vê a tua credencial" é uma propriedade mais útil que "não faz requisição" jamais foi, porque é a que importa para quem decide instalar.

### 1.3 Por que embrulhar o `gh`, e não falar HTTP

Decidido com medição. Existem três fontes de token no mundo real:

| Fonte | O que se mediu |
|---|---|
| config do `gh` (keyring próprio) | o `gh` cuida |
| `git credential fill` | sem helper configurado ele **fica esperando entrada** |
| `GH_TOKEN` / `GITHUB_TOKEN` | lidas por quem souber lê-las |

**O ponto que decidiu: o `gh` já é a união das três.** Ele lê as variáveis, cai no login guardado, e resolve host de Enterprise e SSO. Autenticar "pelo git" não seria fonte alternativa — seria reimplementar o `gh`, pior.

E há o perigo concreto: **uma CLI que trava não é opção**, e é isso que o `credential fill` faz sem helper.

O custo evitado: se o `gtr` obtivesse o token, ele **passaria a segurar segredo** — não pode logar, não pode aparecer em mensagem de erro, não pode ir em `argv`, que é visível no `ps`. Hoje não tem essa responsabilidade, e não vale adquiri-la de graça.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/forge/              o domínio de PR — não sabe de onde vieram
├── pull_request.go          o tipo
├── source.go                Source: PullRequests e Viewer
└── cli.go                   embrulha o gh, via internal/exec

internal/doctor/connected.go a checagem de conexão, fora do Diagnose
cmd/network.go               o prazo de 30s, num lugar só
```

**O contrato `Source` nasceu com dois implementadores em mente** — embrulhar o `gh`, e falar HTTP com o token do ambiente para máquinas sem `gh`. Só o primeiro existe. É a mesma forma dos dois `Extractor` do `internal/commits`.

**Nenhum port novo.** A rede entra pelo `internal/exec`, que já existia para rodar o que não é git. Foi o argumento decisivo a favor do `gh`.

### 2.2 Fora do `Diagnose`, de propósito

A `Connected` não entra na lista das sete checagens do [`doctor`](doctor.md). O resto do comando é local e termina sozinho, **e é isso que o torna barato de rodar por curiosidade** — que é o uso principal dele. Uma checagem de rede dentro do `Diagnose` cobraria o pedágio de todo mundo, sempre.

### 2.3 O prazo

```go
const networkDeadline = 30 * time.Second
```

**É a primeira vez que o projeto precisa de um.** Toda operação local termina sozinha; uma chamada de rede não. Sem prazo, um servidor calado penduraria o comando para sempre. O `context` já atravessava os dois ports, então foi só usar.

---

## 3. Requisitos funcionais

### RF-01 — Perguntar ao GitHub quem somos

`gtr doctor --online`, apelido `--plugged`:

```text
ok     conexão      LHPalma

falta  conexão      sem credencial para o GitHub
                    entre com gh auth login, ou exporte GH_TOKEN com um token de acesso
```

| Situação | Estado |
|---|---|
| sem credencial (`gh` sai **4**) | **falha** — nenhum comando de PR funciona |
| servidor recusou (sai **1**) | **aviso** — há credencial, e o problema pode ser do outro lado |
| `gh` ausente | **pulada** — a checagem do `gh` já disse isso |
| aceito (sai **0**) | `ok`, com o login |

### RF-02 — Declarar a rede

Todo comando que sai da máquina diz no `--help` que sai, e há teste afirmando isso em cada um.

### RF-03 — Perguntar ao GitHub quais permissões o token carrega

`gtr doctor --online` também roda a checagem `escopo`, perguntando via `gh api user -i` — o cabeçalho `X-Oauth-Scopes` da resposta — se o token carrega `read:user`, hoje o único escopo que algum uso real exige além do que os outros comandos já pedem (ver [SRS — profile](profile.md)).

```text
ok     escopo       read:user presente

aviso  escopo       falta read:user
                    rode gh auth refresh -h github.com -s read:user para acrescentá-lo ao token
```

Vira **aviso**, nunca falha: nenhum comando de hoje quebra sem esse escopo — mesmo critério de dependência opcional do [`doctor`](doctor.md). Pulada nos mesmos três casos que a `conexão` já cobre (`gh` ausente, sem credencial, conexão recusada), porque repetir o mesmo aviso duas vezes não ajudaria.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O `gtr` nunca vê a credencial.** Quem fala com o GitHub é o `gh`. Nenhum token passa pelo `gtr` — nem por variável, nem por `argv`, nem em memória. Se um dia o caminho HTTP existir, ele adquire essa responsabilidade, e a especificação dele terá de dizer como a carrega. |
| **RN-02** | **Nenhum comando sai da máquina sem dizer.** A declaração vai no `--help`, e há teste. O `doctor` sem `--online` segue local — ver §2.2. |
| **RN-03** | **Toda chamada de rede tem prazo.** 30 segundos, no `cmd/network.go`. Operação local termina sozinha; servidor calado não. |
| **RN-04** | **A checagem de escopo afirma só o que o token declara, nunca o que a conta consegue acessar com ele.** Ter `read:user` não prova que a leitura vai funcionar, só que a permissão está pedida. |

### 4.1 Regras herdadas

- **O `doctor` não afirma o que não mediu** — `RN-04` da [SRS — doctor](doctor.md), e aqui isso tem consequência direta: a checagem afirma *"há credencial e o servidor a aceitou"*, **nunca** *"teu token é válido"*. Há teste proibindo a palavra na saída. Ver §6.
- **Dependência opcional ausente é aviso, nunca falha** — `RN-02` da [SRS — doctor](doctor.md), que é o que põe o `escopo` em aviso.
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `gh api user`, mais o **código de saída** — nunca a prosa do `gh auth status`.

---

## 5. Achados de implementação

### 5.1 A medição que sustentava o adiamento cobria um comando só

A checagem de autenticação esteve adiada por uma medição incompleta: ela olhou o `gh auth status`, que sai **0** mesmo declarando o token inválido — inútil como sinal — e concluiu que não dava.

Remedindo, com o outro comando:

```console
sem token nenhum      gh api user  → exit 4   "please run: gh auth login"
credencial aceita     gh api user  → exit 0   {"login":"LHPalma",...}
403 / 404             gh api user  → exit 1

gh auth status        → exit 0 mesmo imprimindo "The token is invalid"
```

**O `exit 4` é o achado.** O `gh api user` **distingue** os três casos por código de saída, e devolve JSON contratual. O `gh auth status` não distingue nada.

**A lição de método:** uma decisão de "não dá" vale o que a medição que a sustentou cobriu. Aquela cobria um comando; havia outro.

### 5.2 A divisão em commits falhou, e o próprio portão pegou

A flag foi entregue antes da aplicação, e o primeiro commit não compilava sozinho: o `networkDeadline` morava no `cmd/pr.go`, que era o segundo.

Movido para `cmd/network.go`, que é onde a constante pertence de qualquer forma, e reconferido rodando a suíte com o segundo commit em `stash`. É mais uma vez em que o `gtr commits check` reprova a divisão de quem o escreveu.

### 5.3 O apelido da flag exigiu outro mecanismo

O `pflag` **não tem alias de flag**. Tem `SetNormalizeFunc`, que traduz o nome antes da busca — e é onde o `--plugged` cabe, em vez de registrar uma segunda flag escondida.

O risco desse mecanismo é específico: **normalizador mal-feito quebra as outras flags do mesmo comando.** Há teste afirmando que o `--strict` continua funcionando.

O trocadilho: **guitarra desplugada toca sozinha; plugada precisa do cabo.** É a diferença que a flag desenha.

---

## 6. O terceiro caminho só é observável fora de um proxy que reautentica

Num container cujo proxy de saída reautentica as chamadas ao GitHub, só dois dos três caminhos aparecem: `ok conexão LHPalma` e `falta conexão sem credencial`. O terceiro — token sintaticamente inválido — some, porque a API responde o usuário certo de qualquer jeito.

Numa máquina comum os três aparecem. Com `GH_TOKEN` inválido: `erro: o gh não conseguiu contar as contribuições: HTTP 401: Bad credentials` — o `gh` de verdade recusa, sem proxy nenhum no meio.

A `RN-04` da [SRS — doctor](doctor.md) continua valendo do mesmo jeito — a checagem nunca afirmou mais do que "há credencial e o servidor a aceitou" —, mas é possível testar contra rede de verdade os quatro estados: `ok`, sem credencial (código 4), servidor recusou (código 1) e `gh` ausente.

**E é a mesma medição que justifica a checagem de escopo:** uma conta sem `read:user` devolvia uma soma de contribuições 50 vezes menor que a real, **sem erro nenhum** — ver [SRS — profile](profile.md).

---

## 7. Testes

**100% de statements** em `internal/forge`, `internal/doctor` e `cmd`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Os quatro estados da conexão | `ok`, falha, aviso, pulada — `RF-01` |
| 2 | **A palavra "válido" na saída** | Proibida |
| 3 | `doctor` sem a flag | **Sem** a checagem de conexão — `RN-02` |
| 4 | `--help` do `doctor` | Declara a rede — `RF-02` |
| 5 | `--plugged`, e o `--strict` ao lado dele | Mesma saída; e o normalizador não quebrou a outra flag — §5.3 |
| 6 | Resposta ilegível do `gh api user` | Erro, sem fingir login |
| 7 | Escopo com `read:user`, sem ele, e sem escopo nenhum | `ok`, aviso nomeando o que falta, aviso — `RF-03` |
| 8 | Escopo pulado quando `gh` ausente, sem credencial, ou conexão recusada | Pulada nos três casos — não duplica o aviso de `conexão` |

---

## 8. Rastreabilidade

| Commit | Entrega |
|---|---|
| `06f20fa` | O `internal/forge`, a checagem de conexão e o `--online` — `RF-01`, `RF-02`, `RN-01` a `RN-03` |
| `80869ff` | O apelido `--plugged` — §5.3 |
| `c18c27e` | O README, com a promessa reescrita — §1.2 |
| `ae18bfa` | A checagem `escopo` — `RF-03`, `RN-04` |
| `96fa236` | O README, documentando `escopo` |

---

## 9. Follow-ups conhecidos

- **O caminho HTTP, para máquinas sem `gh`.** O contrato `Source` já o prevê. O ganho real é CI: no GitHub Actions o `GITHUB_TOKEN` sempre existe, o `gh` nem sempre. O custo é a `RN-01` deixar de valer para esse caminho, mais parsear o remote em quatro formas e lidar com paginação e rate limit.
- **O prazo é fixo.** 30 segundos, sem flag para mudar. Se algum dia uma rede lenta reclamar, vira opção — não antes.
- **Escrita continua fora**, sob as três guardas da **ADR-008**. Ler o GitHub foi por onde a rede entrou; abrir e fechar PR é outra conversa.
- **O escopo checado é só `read:user`, hoje.** Se um comando futuro precisar de outro — `repo` de escrita, por exemplo, quando a **ADR-008** abrir —, a checagem `escopo` é o lugar natural para crescer: uma checagem por escopo, mesmo desenho que "uma flag por métrica" no `profile`.
