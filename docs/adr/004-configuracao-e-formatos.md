---
titulo: ADR-004 — Configuração opcional e formatos de saída
data: 2026-08-04
status: --format entregue • .gtr.yaml e gtr config propostos
escopo: .gtr.yaml, gtr config, flag --format compartilhada
supersede: a formulação "nenhum arquivo de configuração", e o item --json que estava aberto
---

# ADR-004 — Configuração opcional e formatos de saída

- **Data:** 2026-08-04
- **Status:** **`--format` entregue** e adotado por todos os comandos que produzem tabela. O **`.gtr.yaml`** e o **`gtr config`** seguem propostos, nada implementado
- **Escopo:** `.gtr.yaml`, `gtr config`, flag `--format` compartilhada
- **Supersede:** o requisito não-funcional na formulação "nenhum arquivo de configuração"

---

## Contexto

Três pressões separadas apontam para o mesmo desenho:

1. **O `branches` tem `main` e `master` cravadas no código.** Um time que usa `develop` ou `release/*` não tem como dizer isso à ferramenta.
2. **Saída para consumo por script** — hoje só texto. Uma flag `--csv` num comando e `--json` noutro multiplicaria superfície.
3. **O `changelog` planejado** vai precisar de vocabulário de seções e de tipos de commit, que é conteúdo do time, não da ferramenta.

### O requisito precisou ser reescrito antes

"Nenhum arquivo de configuração" misturava **duas garantias diferentes numa frase**: segurança (sem token) e fricção zero (funciona recém-baixado). A segunda se mantém inteira com arquivo opcional; a primeira nunca esteve em jogo.

Formulação corrigida: **configuração é opcional em qualquer nível** — o `gtr` nunca cria arquivo sozinho, nunca exige um, e a ausência total de configuração é estado válido e suportado.

## Decisão

### O arquivo

Um nome só: **`.gtr.yaml`**, na raiz do repositório, descoberta com `git rev-parse --show-toplevel` — mecanismo que o `worktrees` já usa, sem código novo de busca.

**Quem decide se a configuração é do time ou sua é o git, não o `gtr`:**

- **Commitado** → convenção do repositório, revisável em PR, vale para todos.
- **No `.gitignore`** → preferência pessoal naquele repositório, ninguém mais vê.

Não há dois formatos nem dois caminhos: a decisão é uma linha no `.gitignore`.

**Duas camadas, merge por chave:**

```text
~/.config/gtr/config.yaml     preferências pessoais, todos os repositórios
<raiz>/.gtr.yaml              este repositório
```

Precedência: **flag > arquivo do repo > arquivo pessoal > padrão embutido.** Merge por chave e não por arquivo — se o repo define só `changelog`, o `branches` do arquivo pessoal sobrevive.

### Formato e vocabulário

**YAML.** `go.yaml.in/yaml/v3` já está no grafo como indireta do cobra; promovê-la a direta não baixa nada novo. TOML custaria dependência nova; JSON não aceita comentário, o que é eliminatório num arquivo editado à mão.

**Chave em inglês, valor livre.** Chave é vocabulário da ferramenta — mesma categoria que nome de flag, e as flags já são `--clean` e `--base`, não `--limpar`. Valor é conteúdo do usuário e vai na língua dele.

```yaml
lang: pt-BR

branches:
  protected: [main, master, develop, "release/*"]
  base: develop

changelog:
  types:
    feat:     { section: "Novidades",  show: true }
    fix:      { section: "Correções",  show: true }
    refactor: { show: false }
  aliases:
    refact: refactor

ignored:
  separator: ";"
```

`section: "Novidades"` é texto que sai no `CHANGELOG.md` — conteúdo, não vocabulário. Um time espanhol escreve `"Novedades"` sem que o `gtr` precise falar espanhol.

**`lang` declara a língua das chaves.** Ausente significa `en`. É o que impede o arquivo meio a meio: declarou `pt-BR` e escreveu `protected`, o `gtr` recusa e sugere `protegidas`.

**Ressalva:** YAML não tem ordem semântica. "Primeira linha" é convenção para humano — o `gtr` tem de ler o `lang` em qualquer posição. Lintar a posição é possível; exigir não é.

**Vocabulário indexado por caminho**, porque nem toda chave é vocabulário:

```text
"branches.protected"         {en: protected, pt-BR: protegidas}
"changelog.types"            {en: types,     pt-BR: tipos}
"changelog.types.*.section"  {en: section,   pt-BR: secao}
"changelog.aliases"          {en: aliases,   pt-BR: apelidos}
```

O `*` marca posição onde a chave é **dado do usuário** — `feat`, `fix`, `refact` nunca são traduzidos. Ausência do caminho no vocabulário significa duas coisas que o parser precisa separar: chave de usuário, que passa intacta, e chave errada, que é erro.

**Parse em duas passadas:** decodifica num `yaml.Node` genérico só para ler o `lang`; monta o mapa reverso; caminha a árvore traduzindo para chaves canônicas; decodifica na struct, que só conhece inglês. **A tradução morre na fronteira do parse** — mesma disciplina do domínio que não conhece texto de tela.

**Começar só com `en` implementado**, com o vocabulário já estruturado como mapa por língua. O `pt-BR` entra depois sem refatoração.

### `gtr config` e `gtr config translate`

**`gtr config`** imprime a configuração efetiva já resolvida, dizendo de qual camada veio cada valor. É o que responde "por que minha configuração não pegou" sem adivinhação.

**`gtr config translate --to <lang>`** traduz o arquivo entre línguas. O `--write` **não é conveniência, é correção** — o caminho óbvio do shell é armadilha:

```bash
gtr config translate --to pt-BR > .gtr.yaml   # ERRADO: o shell trunca antes de ler
gtr config translate --to pt-BR --write       # certo
```

**A tradução trabalha em `yaml.Node`, nunca na struct.** Decodificar e re-serializar apaga comentários, ordem das chaves e linhas em branco — traduziria o arquivo jogando fora a documentação que a pessoa escreveu nele. *Ressalva honesta:* o encoder do `yaml.v3` preserva comentário, mas não é fiel byte a byte — a indentação normaliza. Dizer isso no help do comando.

Os testes que a feature destrava são todos de **propriedade**, não de exemplo: ida e volta devolve o original; traduzir para a mesma língua não muda nada; comentário sobrevive; chave de usuário não é traduzida.

### `--format`

Compartilhada entre comandos, em vez de `--csv` num e `--json` noutro:

```text
--format text|csv|tsv|json     padrão text
--output <arquivo>             padrão stdout, preservando a saída pipeável
--separator <sep>              só com csv
```

`tsv` é açúcar para csv com tabulação, incluindo a extensão padrão.

**Separadores aceitos:** `,` `;` `|` e `\t`. **Conjunto fechado** nesta primeira entrega — separador arbitrário convida a escolher um caractere presente no dado e corromper o arquivo. Abrir depois é compatível; fechar depois quebraria quem já usa.

**Padrão `,`**, conforme a RFC 4180. Seguir o locale da máquina foi considerado e descartado: o mesmo comando geraria arquivos diferentes em máquinas diferentes, e o CSV deixaria de ser reproduzível.

**`--format csv` com `--separator '\t'` emite sugestão de usar `--format tsv` no `stderr`**, nunca no `stdout` — lá ela entraria no meio do CSV e corromperia o arquivo.

**`--separator` com `--format json` é erro, não é ignorado.** Flag que o usuário setou de propósito e a ferramenta descartou calada é o pior modo de falha: ele jura que configurou, ela jura que não.

**Dois detalhes decidem se o arquivo abre no Excel em português:** o separador `;` e o **BOM UTF-8**. Sem os três bytes iniciais, `relatório` é lido como `relatÃ³rio`.

`encoding/csv` é stdlib e já escapa vírgula e aspas dentro do campo. Com tabulação o arquivo é tecnicamente um TSV, e o Go continua **citando** campos que contenham o separador — correto e seguro, mas diferente do TSV clássico, que proíbe tabulação no campo em vez de escapar.

---

## O que a implementação assentou

A maquinaria vive em **`internal/format`**, que não conhece ninguém — só stdlib, como o `internal/git`. No `cmd`, o switch sobre os formatos é escrito uma vez só, atrás da interface `table` (`header`, `rows`, `document`, `text`).

### Uma decisão revista

**O BOM sai só com `--output`, e não sempre.** Esta ADR o tratava como propriedade do formato. Ele é do **destino**: no `stdout` os três bytes grudam no primeiro campo, e quem parseia recebe `﻿.gitignore` onde pediu `.gitignore`. Arquivo continua abrindo certo no Excel, pipe continua parseável. Escrever o BOM saiu do `WriteCSV`, que não tinha como saber para onde seus bytes iam.

### O que a ADR não previa

- **`--separator` aceita as duas grafias da tabulação.** O shell entrega `'\t'` como barra invertida mais `t`, não como tabulação. Aceitar só o caractere real faria falhar a forma escrita nesta própria ADR.
- **Cabeçalho no CSV, e `--no-header` para tirá-lo.** Nomes em português — planilha é lida por gente; as chaves do JSON ficam em inglês, porque aquela saída é lida por código. Fora de `csv` e `tsv` a flag é erro, mesma postura do `--separator`.
- **`--output` acrescenta a extensão quando falta** e respeita a que veio, mesmo trocada. A ferramenta não renomeia o que quem roda nomeou. O arquivo só é criado depois de o git responder, e o erro do `Close` é reportado — flush que falha perde a cauda do arquivo em silêncio.
- **`SetEscapeHTML(false)` no JSON.** Padrão de gitignore é dado, não marcação: `<dist>` sairia escapado à toa.

### O envelope do JSON é porta de mão única

**O JSON de todo comando é um objeto, e a lista mora sob a chave do comando** — `ignored`, `branches`, `worktrees`. Array puro não tem onde pôr o que não é linha, e o `branches` tem exatamente isso: a base que ele resolveu é a premissa da resposta inteira.

Array e envelope **quebram um ao outro nos dois sentidos** — `.branches` devolve `undefined` no array, e o array deixa de ser iterável no envelope. Por isso o primeiro comando a emitir array puro migrou no mesmo lote em que os outros adotaram o formato: ele tinha um dia de vida, o custo de quebrar estava no mínimo e só cresceria.

### Três decisões que a adoção pelo `branches` forçou

- **Metadado que não cabe em coluna vai para onde cada formato sabe receber.** A base vai para o `stderr` no csv e no tsv — mesmo tratamento que o aviso do `--separator '\t'` já recebia — e viaja **dentro do envelope** no json, sem se repetir. Uma coluna `base` repetida em toda linha foi considerada e descartada: no caso vazio ela some, e caso vazio é justamente quando se quer saber qual base foi usada.
- **Valor de enum no JSON é token, não rótulo.** `ancestry`, `squash` e `rebase` vivem em `internal/branch`; `mergeada`, `squashada` e `rebaseada` continuam em `internal/ui` e só saem no csv. **É a internacionalização futura que obriga:** um script casando com `"squashada"` quebraria calado no dia da tradução. Mesma linha que esta ADR já traçava entre cabeçalho em português e chave em inglês, agora valendo também para o **valor**.
- **`--format` com `--clean` é erro.** O `--clean` pergunta, deleta e imprime o que apagou; cruzá-lo com uma tabela ou joga o prompt no meio do csv, ou descarta a flag calada. Mesma postura de `--separator` com json.

E uma consequência menor: o exemplo que o `Path` sugere ao recusar um diretório deixou de ter um nome cravado e virou parâmetro — cada comando sugere o seu.

---

## Alternativas consideradas

**TOML em vez de YAML.** Menos armadilha de indentação e tipagem mais previsível. Rejeitada: custaria dependência direta nova, e o YAML já está no grafo de graça.

**JSON.** Rejeitada por um motivo só, mas eliminatório: **não aceita comentário**, e este é um arquivo editado à mão.

**Chaves em português por padrão.** Rejeitada: chave é vocabulário da ferramenta, mesma categoria de nome de flag. O que se quer em português é o **erro** quando a configuração está errada — e isso é a internacionalização, não esta decisão. Invertido de propósito em relação ao instinto.

**Uma flag por formato (`--json`, `--csv`).** Rejeitada: multiplica superfície por comando e diverge com o tempo. Uma flag compartilhada cobre os quatro de uma vez.

**Separador arbitrário desde já.** Rejeitada por ora: convida a corromper o arquivo. Planejada para quando o formato estiver estável.

**Chave desconhecida ignorada em silêncio.** Rejeitada com veemência — configuração ignorada por typo é o pior modo de falha.

## Consequências

**Positivas**

- O `branches` deixa de ter `main` e `master` cravadas.
- Um só mecanismo de saída estruturada serve todos os comandos, presentes e futuros.
- `gtr config` torna a precedência auditável em vez de adivinhada.

**Negativas**

- Uma dependência indireta vira direta (`yaml.v3`).
- O parse em duas passadas é mais complexo que decodificar direto na struct — preço do multilíngue nas chaves.
- Documentação dobrada quando o `pt-BR` entrar; mitigado começando só com `en`.

**Neutras**

- Nada muda para quem não criar arquivo nenhum — e esse caminho continua sendo testado.
- **Nenhuma credencial no arquivo, jamais.** É o que o mantém seguro de commitar sem pensar.

## Relacionadas

- **[ADR-005](005-gtr-ignore.md)** — consome `ignored.separator` e a mesma flag `--format`.
- **[ADR-006](006-api-do-github.md)** — reforça que credencial nunca entra em arquivo de configuração.
- **[SRS — branches](../srs/branches.md)** — a lista de protegidas cravada é o que esta decisão paga.
