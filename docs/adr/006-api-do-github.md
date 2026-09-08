---
titulo: "ADR-006 — API do GitHub: desenho preliminar"
data: 2026-08-05
status: proposta
escopo: o caminho HTTP direto com a API do GitHub
supersede: nada
---

# ADR-006 — API do GitHub: desenho preliminar

- **Data:** 2026-08-05
- **Status:** **proposta** — nada decidido, nada implementado por este caminho. A rede entrou no projeto **pelo `gh`**, e é o que a [SRS — a conexão](../srs/doctor-online.md) especifica; o que segue vale para o dia em que o caminho HTTP direto existir
- **Escopo:** falar com a API do GitHub sem intermediário
- **Supersede:** nada

---

## Contexto

### A pergunta certa

Não é "que comando faço com a API?", é **"que problema só a API resolve?"**.

Listar PR não qualifica sozinho: o `gh pr list` faz melhor, e um clone do `gh` não tem razão de existir. O que só o `gtr` pode fazer é **cruzar estado local com estado do PR**.

Consequência de desenho: a primeira feature de API provavelmente **não é comando novo**, e sim modo opcional de um comando existente. Menos superfície, mais valor, e o token vira detalhe de implementação em vez de virar assunto.

### O candidato mais forte caiu

O caso que motivava esta decisão — a branch squashada, que parecia exigir perguntar ao GitHub se existiu PR mergeado com aquela `head` — **foi resolvido sem API**, por comparação de patch-id local. Ver [SRS — equivalência por conteúdo](../srs/equivalencia.md).

Isso **não invalida o raciocínio acima — invalida o exemplo.** A pergunta "que problema só a API resolve?" continua aberta e agora está **mais difícil de responder**, porque o candidato mais forte deixou de precisar dela.

**Lição de método:** antes de pagar token, rede e reescrita de requisito não-funcional, vale procurar se o git local já sabe a resposta. Neste caso sabia, e o `git cherry` existe desde sempre para exatamente isso.

### E quando a rede entrou, entrou por outro caminho

A leitura do GitHub que o projeto de fato tem hoje passa pelo `gh`, via `internal/exec`, com o `internal/forge` como port — sem token nenhum passando pelo `gtr`. O desenho abaixo é o do caminho **direto**, que continua sem existir, e cujo maior custo é justamente perder essa propriedade.

## Decisão

**Nenhuma ainda.** O que segue é desenho preliminar, registrado para não ser redescoberto.

### Token — sem fluxo de autenticação próprio

O `gtr` faria com token o que já faz com git: **orquestra o que está na máquina.** Cadeia na mesma forma da resolução de base, informando qual caminho foi usado:

| Ordem | Fonte | Por quê |
|---|---|---|
| 1 | `GH_TOKEN` / `GITHUB_TOKEN` | É o que CI já injeta; funciona sem nada instalado |
| 2 | `gh auth token` | Quem tem o `gh` já tem token autenticado e renovado, e o `gtr` não guarda nada |
| 3 | `git credential fill` | O token que o próprio git já usa para push HTTPS, vindo do keychain ou do libsecret |

Nada encontrado é erro acionável, igual à base indeterminável.

**`gtr auth login` está descartado.** Construir fluxo de OAuth é exatamente o peso que o projeto quis adiar. E a **[ADR-004](004-configuracao-e-formatos.md)** já cravou que credencial nunca entra em arquivo de configuração.

### Duas promessas precisam ser reescritas antes

- **Sem rede** hoje é formulação amarrada ao momento — "nenhuma das três primeiras features faz requisição HTTP". Vira regra por comando: **nenhum comando faz requisição de rede sem declarar**, e os locais continuam offline para sempre.
- **Funciona sem configuração** continua íntegra para tudo que existe, mas passa a ter uma exceção nomeada.

Sem essa cirurgia, a feature entra contradizendo o documento.

### Descobrir dono e repositório

Vem de `git remote get-url origin`, com formas variadas — `git@github.com:owner/repo.git`, `https://github.com/owner/repo`, `ssh://git@github.com/owner/repo`, com e sem `.git`. Três armadilhas:

- **`url.<base>.insteadOf`** no gitconfig reescreve URL: o que o `get-url` devolve pode não ser o que o git de fato usa.
- **Sem `origin`, ou com vários remotes** (fork mais upstream) é caso comum, e qual é o "certo" depende de intenção.
- **GitHub Enterprise** muda o host da API. Cravar `api.github.com` nasce errado.

### Duas regras de segurança que nascem junto

- **Falha de rede aborta, nunca degrada.** Se a resposta alimenta uma deleção, request que falhou não pode virar "nenhum PR mergeado encontrado" — isso apagaria trabalho por causa de rede ruim. É *nunca destruir na dúvida* aplicado à rede.
- **Só leitura.** A API expõe `DELETE /git/refs/heads/...`; a regra que proíbe escrita em ref remota continua valendo, agora também pelo caminho HTTP.

### Dependência e teste

**Sem `go-github`.** São megabytes de dependência para dois endpoints; `net/http` e `encoding/json` resolvem em ordem de 100 linhas e mantêm o grafo do tamanho que ele tem.

**O fake mora no port, não no transporte.** O `gittest` funciona porque git está atrás do `Runner`; o análogo é um port de API devolvendo dado já parseado, com fake no mesmo estilo. Mockar `http.RoundTripper` testaria serialização, que não é onde mora o risco.

## Alternativas consideradas

**Usar `go-github`.** Cobertura completa da API, tipagem pronta. Rejeitada: megabytes para dois endpoints, num projeto cujo grafo de dependências tem uma direta e duas indiretas.

**Fluxo de OAuth próprio (`gtr auth login`).** Rejeitada: é o peso que o projeto adiou de propósito, e há três fontes de token já disponíveis na máquina.

**Mockar `http.RoundTripper` nos testes.** Rejeitada como padrão: testa serialização, não a lógica. O fake vai no port.

**Search API do GitHub** para achar PRs. Desnecessária — `GET /repos/{owner}/{repo}/pulls?state=closed&head=owner:branch` é mais barato e direto.

## Consequências

**Se um dia for implementada:**

- As duas promessas mudam de formulação antes da primeira linha de código.
- **O `gtr` passa a segurar credencial**, que é exatamente a propriedade que o caminho pelo `gh` preserva hoje: nenhum token passa por ele, nem por variável, nem por `argv`, nem em memória.
- Aparece um modo de falha novo — rede — num comando que hoje só falha por estado local.

**Adiado explicitamente:** paginação, rate limit (60/h sem token contra 5000/h com) e cache por `ETag` — resposta `304` não conta no limite. Nada disso é necessário numa primeira entrega que consulta poucas branches por execução.

## Relacionadas

- **[SRS — a conexão: doctor --online](../srs/doctor-online.md)** — o caminho que de fato existe, pelo `gh`, e onde o HTTP direto está registrado como follow-up.
- **[SRS — equivalência por conteúdo](../srs/equivalencia.md)** — o caso que motivava esta decisão, resolvido sem ela.
- **[ADR-004](004-configuracao-e-formatos.md)** — credencial nunca entra em arquivo de configuração.
- **[SRS — branches](../srs/branches.md)** — as regras que continuam valendo pelo caminho HTTP.
