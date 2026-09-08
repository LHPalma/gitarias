---
titulo: SRS — licenses
data: 2026-08-13
status: entregue
comando: gtr licenses
pacotes:
  - cmd/licenses.go
  - main.go
fonte_externa: nenhuma
---

# SRS — licenses

- **Data:** 2026-08-13
- **Feature:** `cmd/licenses` + o `//go:embed` do `main`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma. O texto viaja dentro do binário

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr licenses`, que mostra a licença do `gtr` e as das dependências **embutidas no binário**.

A obrigação é concreta, não cerimônia: o binário é estaticamente ligado (**[ADR-001](../adr/001-go-e-binario-unico.md)**), então o código das dependências **viaja dentro dele**, e as licenças delas exigem acompanhar a redistribuição. Um binário distribuído por cópia não tem um diretório ao lado para carregar o `THIRD-PARTY-LICENSES` junto — ou o texto está dentro dele, ou a promessa de "copie um arquivo e rode" quebra a licença de terceiros.

### 1.2 Escopo

**Entregue:** o arquivo `THIRD-PARTY-LICENSES` embutido por `//go:embed`; resumo por padrão; `--full` para o texto completo; um teste que compara o `go.mod` com o arquivo e falha se entrar dependência não listada.

**Fora de escopo:** gerar o `THIRD-PARTY-LICENSES` — ele é escrito à mão e conferido por teste, em vez de produzido por ferramenta que precisaria virar dependência. Detectar licença incompatível: o teste afirma **presença**, não compatibilidade.

### 1.3 O nome

`licenses`, no plural e em inglês — é o termo que quem procura essa informação digita, e o mesmo que outras CLIs usam. Sem apelido: é utilitário, não piada.

---

## 2. Descrição geral

```text
main.go                      //go:embed THIRD-PARTY-LICENSES → a string notices
cmd/licenses.go              o comando, a flag --full e o notice()
THIRD-PARTY-LICENSES         o arquivo, escrito à mão e conferido por teste
```

O texto entra pelo **ponto de injeção**, como todo o resto: o `main` embute e passa a string ao `NewRootCommand`. O `cmd` nunca lê arquivo — o que ele recebe é conteúdo, e é isso que torna o comando testável sem tocar o disco.

O separador entre o resumo e o texto completo é uma linha de oitenta `=`, e é ele que o `notice()` corta.

---

## 3. Requisitos funcionais

### RF-01 — Mostrar o resumo por padrão

`gtr licenses` imprime a primeira seção do arquivo — a tabela de dependência, licença e observação —, que é o que responde a pergunta comum: *o que viaja dentro deste binário, e sob qual licença*.

### RF-02 — `--full` imprime tudo

Com `--full`, sai o arquivo inteiro, com o texto integral de cada licença — que é o que a redistribuição exige de fato.

### RF-03 — Não exige repositório git

É o terceiro comando com essa isenção, ao lado do `doctor` e do `riff`, e por uma razão própria: o que ele mostra não vem do repositório, vem do binário. Roda em qualquer diretório.

### RF-04 — Nenhuma flag de formato

O `licenses` fica **fora** da família `--format` da **[ADR-004](../adr/004-configuracao-e-formatos.md)**. Ele não emite tabela de dados: emite texto de licença, que ninguém consome como csv e que perderia sentido reformatado.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **A licença viaja com o binário, não ao lado dele.** Embutir por `//go:embed` é o que mantém a distribuição por cópia compatível com as licenças das dependências. Um arquivo separado se perde na primeira vez que alguém copia só o executável. |
| **RN-02** | **O teste afirma presença, nunca compatibilidade.** Ele compara o `go.mod` com o arquivo e falha se entrar dependência cujas licenças não estejam ali. Dizer se uma licença é aceitável é decisão de quem mantém, não de um teste. |
| **RN-03** | **O teste recusa passar de graça.** Se o `go.mod` não declarar dependência nenhuma, ele falha em vez de aprovar a lista vazia — e também falha se o conteúdo embutido for um marcador em vez do arquivo de verdade. É a mesma disciplina de provar que o cenário existiu. |

### 4.1 Regras herdadas

- **Nenhuma linha de saída termina em espaço** — `RN-10` da [SRS — branches](branches.md), com teste próprio aqui.
- **O ponto de injeção recebe tudo de fora** — **[ADR-003](../adr/003-injecao-de-dependencia.md)**: o conteúdo embutido desce por parâmetro, e o comando nunca abre arquivo.

---

## 5. Achado de implementação: o `--full` virava no-op silencioso no Windows

O `//go:embed` lê o arquivo **como o disco entregou**. Num checkout Windows com `core.autocrlf=true` — que é a configuração recomendada pelo próprio instalador do Git for Windows —, isso significa **CRLF**.

O separador que o `notice()` procura é escrito só com `LF`, então ele **nunca casava** naquele checkout: o `strings.Cut` não achava nada, devolvia a string inteira, e `--full` passava a não fazer diferença nenhuma — sem erro, sem aviso, com exit 0.

A correção é normalizar antes de cortar:

```go
notices = strings.ReplaceAll(notices, "\r\n", "\n")
```

**O que este caso ensina não é sobre licença, é sobre `embed`:** todo conteúdo embutido carrega a convenção de fim de linha do checkout que compilou, e qualquer comparação com literal escrito em LF é sensível a isso. Coberto por `TestNoticeCutsTheSeparatorEvenOnCRLF`.

---

## 6. Testes

**100% de statements** em `cmd/licenses.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Sem flag | Só o resumo, cortado no separador — `RF-01` |
| 2 | `--full` | O arquivo inteiro — `RF-02` |
| 3 | Conteúdo sem separador nenhum | Sai inteiro, sem erro |
| 4 | Conteúdo em CRLF | O corte funciona igual — §5 |
| 5 | Fora de um repositório git | Roda normalmente — `RF-03` |
| 6 | Nenhuma linha termina em espaço | §4.1 |
| 7 | Toda dependência do `go.mod` está no arquivo | Falha nomeando a que faltar — `RN-02` |
| 8 | `go.mod` sem dependência, e conteúdo de marcador | Falha em vez de aprovar — `RN-03` |

---

## 7. Follow-ups conhecidos

- **O `THIRD-PARTY-LICENSES` é escrito à mão.** Gerar por ferramenta (`go-licenses` e afins) traria uma dependência de build para um arquivo que muda quando o `go.mod` muda — hoje, três vezes desde o início. O teste é a rede.
- **A observação de plataforma é texto livre.** `(só no binário Windows)`, ao lado do `mousetrap`, é convenção de quem escreve, não campo estruturado.
- **Nada afirma compatibilidade de licença** — `RN-02`. Se um dia entrar dependência copyleft, quem percebe é quem lê o PR.
