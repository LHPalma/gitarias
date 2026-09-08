---
titulo: SRS — setup
data: 2026-08-15
status: entregue
comando: gtr setup
apelido: luthier
pacotes:
  - cmd/setup.go
  - internal/platform
commits:
  - 138bce3
fonte_externa: nenhuma
---

# SRS — setup

- **Data:** 2026-08-15
- **Feature:** `cmd/setup` + `internal/platform`
- **Status:** **entregue** — commit `138bce3` na `main`. O `--run` foi **descartado**, e a §6 diz por quê
- **Fonte externa:** nenhuma. Lê o `PATH` e o `/etc/os-release`, e não executa nada

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr setup`, que responde **o que falta e qual comando resolve nesta máquina**.

O [`doctor`](doctor.md) diz que o `gh` não está no `PATH` e aponta para `cli.github.com`. **É a mesma frase em toda máquina**, e não é a frase que quem acabou de instalar o `gtr` quer. Essa pessoa quer a linha para colar.

### 1.2 Escopo

**Entregue:** detecção do gerenciador de pacotes entre nove; comando de instalação com `sudo` só quando é preciso; queda para a página oficial sem gerenciador reconhecido; dica do próprio diagnóstico para o que não se resolve instalando.

**Fora de escopo, e é o coração do desenho: executar.** Ver `RN-01`.

### 1.3 O nome

**`setup`**, porque é o que quem tem essa pergunta digita.

**`luthier` é apelido**, e diferente dos outros do elenco não é analogia: **"setup" já é termo de luthieria** — levar a guitarra para um *setup* é o serviço de regular ação, tensor, rastilho e entonação. A mesma palavra nos dois mundos.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/platform/           a máquina de quem roda — só stdlib
├── finder.go                interface Finder, e o System de verdade
├── manager.go               os nove gerenciadores e o pacote de cada um
├── platform.go              Detect, Install, Describe
└── platformtest/            fake que descreve uma máquina que não é esta

cmd/setup.go                 o comando, o cruzamento e a receita
```

**O `internal/platform` nunca vê um `Check`.** Ele sabe de máquinas e pacotes. O cruzamento *"a checagem do gh falhou" → "instalar gh"* mora no `cmd`, pelo mesmo motivo que o cruzamento entre `branch` e `worktree`: nenhum dos dois lados precisa conhecer o outro.

**A detecção entra por interface.** Sem o `Finder`, só daria para testar o Linux em que a suíte roda; com ele, a suíte exercita macOS, Windows e cinco distribuições a partir de uma só.

### 2.2 O que é detectável sem shell

```go
exec.LookPath("apt-get")   // o gerenciador
/etc/os-release            // ID e VERSION_ID
os.Geteuid() == 0          // precisa de sudo?
runtime.GOOS               // qual tabela consultar
```

Nada disso passa por shell.

---

## 3. Requisitos funcionais

### RF-01 — Imprimir o comando desta máquina

```console
$ gtr setup
Detectei ubuntu 24.04, com apt-get.

git — não encontrado no PATH:
    sudo apt-get update
    sudo apt-get install -y git
```

O `sudo` aparece **só quando o gerenciador precisa e quem roda não é root** — conferido rodando o binário como `nobody` e como root.

**Dizer o que detectou não é enfeite:** sem isso, quem lê não tem como saber se o comando serve para a máquina dele.

### RF-02 — Conhecer nove gerenciadores, e o nome do pacote em cada um

| Sistema | Ordem de procura |
|---|---|
| linux | `apt-get`, `dnf`, `pacman`, `zypper`, `apk` |
| darwin | `brew` |
| windows | `winget`, `choco`, `scoop` |

**O nome do pacote nem sempre é o nome do comando**, e chutar daria uma linha que falha: o `gh` é `github-cli` no Arch e no Alpine, e o git é `Git.Git` no winget. Há teste para os dois.

Havendo mais de um gerenciador no `PATH`, **a ordem da tabela decide**, não a do `PATH`.

### RF-03 — Cair na página oficial em vez de inventar

Sem gerenciador reconhecido, ou para ferramenta que a tabela não conhece, o `Install` devolve **nada** e o front imprime `instale a partir de <página>`.

### RF-04 — Só ferramenta ausente vira instalação

Git abaixo da mínima, ou binário que está no `PATH` e não roda, **não** recebem linha de `install`: o comando não resolveria. Esses caem na dica do próprio diagnóstico.

O mesmo vale para o que não se instala — base adivinhada vira `git fetch origin`, identidade vira `git config`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O `setup` imprime e não executa**, e há teste afirmando que as únicas chamadas externas são os `--version` do diagnóstico. Instalar pacote pede privilégio, e **binário que pede senha de root para agir sozinho é o hábito que não vale ensinar**. Quem executa é quem lê, vendo o quê. |
| **RN-02** | **Comando inventado é pior que nenhum.** Sem gerenciador reconhecido, a ferramenta não improvisa: manda para a página oficial. Uma linha que falha custa mais confiança que uma ausência admitida. |
| **RN-03** | **Privilégio só quando é preciso.** `sudo` sai no `apt`, `dnf`, `pacman`, `zypper` e `apk` — e só se quem roda não for root. `brew`, `winget`, `choco` e `scoop` nunca o pedem. |

### 4.1 Regras herdadas

- **O `doctor` diagnostica e nunca conserta** — `RN-01` da [SRS — doctor](doctor.md), que a `RN-01` daqui preserva: o `setup` é o passo seguinte do diagnóstico e continua sem agir.
- **Sem shell** — `RN-06` da [SRS — branches](branches.md). `exec.LookPath` e leitura de arquivo, nunca `sh -c`.
- **O erro que se lê é acionável** — `RN-07` da [SRS — branches](branches.md). Aqui, é o comando a colar, não um endereço genérico.
- **Nenhuma linha termina em espaço** — `RN-10` da [SRS — branches](branches.md).

---

## 5. Achados de implementação

### 5.1 A detecção é fácil; a decisão é que era o problema

Medido antes de escrever — `LookPath`, `/etc/os-release` e `Geteuid` resolvem tudo, sem shell. **O "como" nunca foi o obstáculo.** O obstáculo era o "o quê": instalar exige `sudo`, e a regra de o diagnóstico nunca consertar existe para que rodá-lo seja seguro.

### 5.2 Um teste achou código morto no leitor de distribuição

O `Release()` tinha `if err != nil { return "" }`, e esse ramo era inalcançável a partir de um Linux com o arquivo presente. Como o `os.ReadFile` **já devolve nada em caso de falha**, o `if` era cerimônia. Saiu, e a cobertura fechou em 100% sem abrir buraco documentado.

---

## 6. O `--run` foi descartado, e a razão é que ele não tinha o que rodar

A ideia seguinte era `setup --run`, executando **só o que fosse local e sem privilégio**. Percorrendo o que o diagnóstico reporta, não sobra carga:

| Conserto | Por que o `--run` não serve |
|---|---|
| instalar `git` ou `gh` | privilégio — `RN-01` |
| `git config user.name` | só o autor sabe o valor |
| `rebase --continue` / `--abort` | destrutivo, e a escolha é do autor |
| `TMPDIR` | variável do shell do pai; processo filho não a muda |
| base adivinhada | **único candidato** — e as duas saídas são ruins |

Na base, as duas opções foram medidas:

- **`git fetch origin`** — seria chamada de rede, e o `setup` é um comando local.
- **`git remote set-head origin main`** — confirmado que é **puramente local**, só escreve uma ref simbólica apontando para uma ref de rastreamento que já está no disco. Mas exige **saber qual é a padrão do remoto, e só o remoto sabe**. Sem rede, o `gtr` estaria **gravando o próprio chute** no repositório.

**E gravar o chute é estritamente pior que o aviso.** Hoje o `doctor` diz "adivinhei"; ali passaria a fingir que sabe, e um chute errado viraria permanente e invisível — contra a regra de mostrar em vez de julgar.

**A flag existiria com um botão só, e esse botão piora a informação.**

---

## 7. Testes

**100% de statements** em `internal/platform`, `platformtest` e `cmd/setup.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Onze combinações de sistema e gerenciador, em tabela | O gerenciador certo, ou nenhum — `RF-02` |
| 2 | Três gerenciadores no `PATH` ao mesmo tempo | A ordem da tabela decide, não a do `PATH` |
| 3 | `apt` como root e sem ser root; `brew` e `winget` | `sudo` só onde precisa — `RN-03` |
| 4 | `gh` no Arch; git no winget | `github-cli` e `Git.Git` — `RF-02` |
| 5 | Sem gerenciador; ferramenta desconhecida | Nada, e o front cai na página — `RF-03` |
| 6 | Git abaixo da mínima | **Sem** linha de instalação — `RF-04` |
| 7 | Tudo no lugar | "Nada a fazer" |
| 8 | Fora de um repositório | Não sugere instalar um repositório |
| 9 | **As chamadas externas do comando** | Só os `--version` — `RN-01` |
| 10 | `luthier` e a ajuda da raiz | Mesma saída; apelido não anunciado |

**Conferido por mutação:** prometendo instalação para ferramenta que existe, um teste cai; pedindo `sudo` sempre, dois caem.

---

## 8. Rastreabilidade

| Commit | Entrega |
|---|---|
| `138bce3` | O `internal/platform`, o comando, o `luthier` — `RF-01` a `RF-04`, `RN-01` a `RN-03` |

---

## 9. Follow-ups conhecidos

- **O `gh` não está no repositório padrão de todo Debian.** No Ubuntu 24.04 o `apt-get install gh` funciona — conferido —, mas em versões mais antigas exige adicionar chave e repositório. O comando não distingue, e a linha pode falhar lá.
- **Não confere se o gerenciador de fato tem o pacote.** Um `apt-cache policy` diria antes, ao custo de uma invocação por ferramenta.
- **Nix, Homebrew no Linux e o `pkg` dos BSD ficaram de fora.**
