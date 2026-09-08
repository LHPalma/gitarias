---
titulo: ADR-001 — Go e binário único estaticamente ligado
data: 2026-08-04
status: entregue e verificada por CI
escopo: build e distribuição
supersede: o `go build -o gtr .` sem CGO_ENABLED=0
---

# ADR-001 — Go e binário único estaticamente ligado

- **Data:** 2026-08-04
- **Status:** **entregue** e verificada por CI
- **Escopo:** build e distribuição
- **Supersede:** o `go build -o gtr .` que a especificação original prescrevia

---

## Contexto

A ferramenta é uma CLI de uso pessoal e de time, distribuída por cópia — sem gerenciador de pacotes, sem instalador. A promessa é "copie um arquivo e rode".

Go foi escolhido por dois motivos: o binário único sem runtime instalado na máquina de quem usa, e o ecossistema de CLI e TUI mais maduro que existe hoje — `cobra` é o que `gh`, `kubectl` e `docker` usam, e a família Bubble Tea cobre o front interativo planejado.

**O problema aparece na primeira execução completa da bateria.** O comando documentado não entregava o que o requisito prometia:

```console
$ go build -o gtr .
   ELF 64-bit, dynamically linked
   libc.so.6 => /lib/x86_64-linux-gnu/libc.so.6
   simbolos: GLIBC_2.3.4  GLIBC_2.32  GLIBC_2.34
```

Em Linux com toolchain C presente, o Go liga dinamicamente por padrão — `os/user` e `net` têm caminho cgo. Copiar esse binário para uma máquina de glibc mais antiga (Ubuntu 20.04, Debian 11, RHEL 8) falha na execução com `version GLIBC_2.34 not found`.

**Não é defeito do código.** É o comportamento padrão do linker, e o requisito prometia o que o comando documentado não cumpria.

## Decisão

**`CGO_ENABLED=0` é obrigatório, não otimização**, e a afirmação vira **falha de CI** em vez de disciplina.

```bash
CGO_ENABLED=0 go build -o gtr .
```

```console
$ CGO_ENABLED=0 go build -o gtr .
   ELF 64-bit, statically linked
   ldd: not a dynamic executable
```

Os dois binários produzem saída idêntica; a diferença é só de empacotamento.

**O pipeline afirma a propriedade**, não só compila: um passo do CI verifica que o binário sai `statically linked` e falha se não sair. Outro cruza para `darwin/arm64` e `windows/amd64` — plataformas onde o problema não existe, mas onde a quebra de compilação existiria.

A versão do Go vem do `go.mod` via `go-version-file`, para pipeline e módulo não divergirem.

## Alternativas consideradas

**Documentar a limitação e seguir com `go build` simples.** Rejeitada: o requisito não-funcional passaria a ser "binário único, exceto em Linux, exceto se a glibc de destino for antiga" — que é o mesmo que não ter o requisito.

**Compilar com `-tags netgo osusergo` em vez de desabilitar cgo.** Chega ao mesmo lugar para estes dois pacotes, mas exige saber de antemão quais pacotes têm caminho cgo. `CGO_ENABLED=0` é a garantia categórica; as tags são a garantia caso a caso.

**Distribuir por gerenciador de pacotes** (Homebrew, apt). Rejeitada por enquanto: resolve o problema criando outro maior — empacotamento por plataforma, versionamento, publicação. O projeto não tem usuário externo ainda.

**Outra linguagem.** Rust daria o mesmo binário estático com menos pegadinha de linker, mas o ecossistema de CLI e TUI é menos maduro, e a curva era desnecessária para o objetivo do projeto.

## Consequências

**Positivas**

- A promessa de distribuição por cópia passa a ser verdadeira em qualquer glibc.
- O CI carrega a afirmação: quem quebrar o build estático descobre no PR, não no usuário.
- Cross-compile verificado para as três plataformas sem alteração de código.

**Negativas**

- `CGO_ENABLED=0` precisa aparecer em toda instrução de build — README, CONTRIBUTING, CI. Esquecer num lugar produz um binário que funciona na máquina de quem compilou.
- Se algum dia uma dependência exigir cgo de verdade, a decisão precisa ser revisitada por inteiro.

**Neutras**

- Nenhuma diferença de comportamento entre os dois binários — só de ligação.
- Em macOS e Windows a variável não muda nada, mas também não atrapalha.

## Relacionadas

- **[ADR-003](003-injecao-de-dependencia.md)** — mesma natureza: o requisito existia, faltava o mecanismo cumpri-lo.
- O pipeline de CI é o que transforma esta decisão em portão.
