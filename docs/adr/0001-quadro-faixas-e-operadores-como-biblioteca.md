# 0001. Quadro com faixas (zero-cópia) e operadores como biblioteca

- **Status:** aceita
- **Data:** 2026-10-08

## Contexto

O KE é a v2.0 do KEngine (spec `KEngine/.planning/specs/kengine-2-0-arquitetura.md`,
ADR 0015 do KEngine). O cerne é o firewall, com o router, e os operadores.
Dois pedidos guiam esta fatia:

1. Processos são **sequências de funções mínimas** (o princípio do ISA, spec
   §1.1), e não funções grandes que refazem o trabalho delas por fora.
2. O router **não move dados pela memória**: o trabalho é feito com endereços
   e posições dentro de buffers.

O spec §7.5 proíbe `unsafe` e assembly.

## Decisão

1. **`quadro`**: um buffer de capacidade fixa (64 KiB) por comando.
   - O transporte lê **direto** para ele (`Reservar` + `io.ReadFull`).
   - `Indexar` marca a `Faixa{Ini, Fim}` de cada campo TLV
     (`[tag u16][tamanho u16][valor]`).
   - Todos os módulos passam o endereço (`*Quadro`) e leem pela faixa.
     `Campo` devolve `buf[ini:fim:fim]`: é uma janela e não copia; a
     capacidade cortada impede um `append` sobre o vizinho.
   - A saída é anexada **no mesmo buffer**. Sem capacidade, a operação recusa
     em vez de realocar.
   - O pool sobrescreve os bytes usados antes de reaproveitar o quadro.
   - Valores intermediários ficam em registradores (`Reg [8]int64`).
2. **`operadores` é biblioteca** e depende só da stdlib e do `quadro`.
   - Qualquer projeto a importa; ela não conhece firewall, núcleo, plataforma
     nem negócio.
   - `Op func(*quadro.Quadro) error` é a instrução; `Seq` compõe instruções.
   - Operador novo só entra quando uma receita real pedir.
3. **Regra ISA imposta por teste**: fora de `operadores/`, nenhum código
   define `func(*quadro.Quadro) error`. Um processo novo é uma sequência de
   operadores; se faltar uma instrução, ela entra na biblioteca.
4. `firewall/router/externo` é o único pacote que importa `net`. O envio usa
   `net.Buffers` (writev): cabeçalho e corpo saem sem serem concatenados.

## Desvios do spec v2

- **D24:** as receitas saíram de `operadores/<plataforma>/pipelines.go`. São
  de quem consome a biblioteca, porque `operadores` não pode ter um espaço por
  plataforma e continuar independente de projeto.
- **§7, tabela:** `operadores` deixa de ser "não importa nada" e passa a
  importar `quadro`, que também é biblioteca e também depende só da stdlib.

## Alternativas consideradas

- **JSON no fio** (como na v1): descartado aqui. `encoding/json` aloca e
  copia cada campo. O JSON fica para o futuro `api/` (Protheus, Pix).
- **`unsafe` para aritmética de ponteiro**: descartado (spec §7.5,
  confiabilidade primeiro). Slices já dão endereço e posição com checagem de
  limite.
- **Buffer que cresce com `append` livre**: descartado. Realocar invalida a
  premissa de "o dado não se move" e esconde o custo.
- **`go.mod` próprio para a biblioteca**: adiado (A5). Com um módulo só, quem
  importa `operadores` já compila só `operadores` e `quadro`. Vale separar
  quando a biblioteca precisar de versão própria.

## Consequências

- Medido em 08/10/2026 (`BenchmarkCampoPor*`, campo de 512 B): faixa 1,3 ns
  e 0 alocações; cópia 102 ns, 512 B e 1 alocação.
- `TestCaminhoNaoAloca` reprova se alguém introduzir cópia ou alocação entre
  a leitura e a saída.
- Mais difícil: o formato TLV é nosso (com fuzz em `FuzzIndexar`), e a posse
  do quadro vai junto com o ponteiro (`router/interno`). Quem encaminha não
  pode tocar mais nele, e o `-race` cobra isso.
- Limites desta fatia: 64 tags por mensagem, 8 registradores e entrada de no
  máximo 32 KiB (metade do quadro, para a resposta sempre caber).
