# Guia das funções do KE

O que cada função faz, como uma função "complexa" do negócio vira uma cadeia de funções básicas, e como escrever as suas. Todos os exemplos foram executados contra o código; os números são os reais. A referência rápida (assinaturas e erros) está no [README](../README.md).

> O projeto está em pausa ([ADR 0003](adr/0003-pausa-e-direcao-transpilador-python.md)). O guia descreve o código como ele está.

## 1. A ideia

O KE não tem "funções grandes". Um cálculo de negócio, como o total de uma fatura, **não é uma função Go**: é uma **sequência de instruções mínimas** (operadores) que lê campos de um buffer, guarda valores em registradores e escreve a resposta no mesmo buffer. É o princípio do ISA: poucas instruções, previsíveis, e o que diferencia um processo de outro é só a ordem.

| Camada | Peça | É o quê |
| --- | --- | --- |
| Átomo | `Converter`, `Somar`, `Escrever`... | Uma instrução: `func(*quadro.Quadro) error` |
| Cadeia | `Seq` | Átomos em ordem; para no primeiro erro |
| Função complexa | `Receita` | Uma `Seq` com nome e a lista de entradas que ela exige. É a "mini-thread" |
| Orquestração | `Juncao`, `Transmitir`, Router | Espera as entradas chegarem, passa o quadro entre pipelines |
| Memória | `Quadro` | O buffer do comando, o índice dos campos e os registradores |

## 2. Como a função complexa inicia a cadeia

A receita nunca é chamada "no meio" de nada. Quem a inicia é o Router, quando o dado já está no quadro e passou pelo firewall:

```text
fio
 └─ externo.Receber        lê direto para o buffer do quadro (1 única vez)
     └─ entrada.Verificar  indexa os campos (Faixa) e aplica o contrato por tag
         └─ interno.Encaminhar   passa só o ENDEREÇO do quadro (8 bytes) ao trabalhador
             └─ Receita.Executar       ← a função complexa começa aqui
                 ├─ confere as entradas exigidas (falta alguma? ErrAguardando)
                 └─ Seq: Converter → Multiplicar → Subtrair → ... → Escrever   ← cadeia de átomos
                     └─ externo.Enviar     a resposta sai do MESMO buffer
                        (ou Transmitir: o quadro segue para a próxima pipeline)
```

Três coisas que esse desenho garante:

1. **O dado não se move.** Cada átomo lê pela `Faixa` e escreve registradores. Ninguém copia campo.
2. **A receita só dispara quando pode.** `Executar` confere as entradas antes do primeiro átomo. Faltando uma, devolve `ErrAguardando`, que é espera, não falha.
3. **Erro diz onde.** A `Seq` embrulha o erro com a etapa: `fatura-item: etapa 0: operadores: formato inválido`.

## 3. O que cada função faz

### Átomos (`operadores`)

Cada operador é **montado uma vez** (devolve um `Op`) e **executado por comando**. Registrador inválido dá `panic` na montagem, nunca no meio de um comando.

| Operador | Lê | Escreve | Falha com |
| --- | --- | --- | --- |
| `Converter(dst, tag)` | campo `tag` (dígitos ASCII) | `Reg[dst]` | `ErrAusente`, `ErrFormato` (vazio, não dígito, sinal), `ErrEstouro` |
| `Somar(dst, a, b)` | `Reg[a]`, `Reg[b]` | `Reg[dst] = a + b` | `ErrEstouro` |
| `Subtrair(dst, a, b)` | idem | `Reg[dst] = a - b` | `ErrEstouro` |
| `Multiplicar(dst, a, b)` | idem | `Reg[dst] = a * b` | `ErrEstouro` |
| `Dividir(dst, a, b)` | idem | `Reg[dst] = a / b` | `ErrInexato` (tem resto), `ErrDivisaoPorZero`, `ErrEstouro` |
| `Arredondar(dst, a, b, modo)` | idem | `Reg[dst] = a / b` arredondado | `ErrDivisaoPorZero`, `ErrEstouro` |
| `Escrever(tag, src)` | `Reg[src]` | campo `tag` na saída, em decimal | `ErrCheio`, `ErrSolto` |
| `Transmitir(pipeline, entrega)` | o quadro | solta e entrega o quadro | o erro da `entrega` (o quadro volta à origem) |

Modos de `Arredondar`, para `-7 / 2 = -3,5` e `7 / 2 = 3,5`:

| Modo | `7/2` | `-7/2` | Regra |
| --- | --- | --- | --- |
| `Truncar` | 3 | -3 | em direção a zero |
| `Piso` | 3 | -4 | em direção a -∞ |
| `Teto` | 4 | -3 | em direção a +∞ |
| `MeioAcima` | 4 | -4 | ao mais próximo; a metade vai para longe de zero |
| `MeioPar` | 4 | -4 | ao mais próximo; a metade vai para o par (bancário) |

`Dividir` e `Arredondar` são separados de propósito: o resto nunca some em silêncio. Quem quer o quociente de uma divisão inexata escolhe o modo, às claras.

### Cadeia e função complexa (`operadores.Seq`, `pipeline.Receita`)

| Função | O que faz |
| --- | --- |
| `Seq(ops...)` | Roda os operadores em ordem; no primeiro erro para e diz a etapa. É a única forma de montar um processo |
| `Declarar(nome, entradas, etapas...)` | Monta a `Receita`: o nome, as tags exigidas e a `Seq` das etapas |
| `r.Executar(q)` | Confere as entradas e roda a `Seq`. Se a cadeia terminou em `Transmitir`, devolve `nil` |

### Orquestração (`pipeline.Juncao`, `Transmitir`, router)

| Função | O que faz |
| --- | --- |
| `NovaJuncao(r, n)` / `j.Entregar(chave, o)` | Junta pedaços da mesma entidade vindos de pipelines diferentes e só dispara a receita quando todas as entradas chegaram |
| `Transmitir(pipeline, entrega)` | Passa o quadro para outra pipeline. Sentido único, sem devolução |
| `interno.Rota` (`Encaminhar`, `Recebidos`, `Fechar`) | Leva o endereço do quadro entre goroutines; cheia, recusa na hora |
| `externo.Receber` / `externo.Enviar` | Fio para o quadro, e saída do quadro para o fio, sem cópia |
| `entrada.Verificar` | Indexa o quadro e aplica o contrato (declaração, presença, tamanho, formato) |

### Memória (`quadro`)

O quadro é o único lugar onde o dado mora. As funções se dividem em cinco papéis:

| Papel | Funções |
| --- | --- |
| Ciclo de vida | `Pegar`, `Novo`, `Devolver`, `Zerar` |
| Entrada | `Reservar` (área para o transporte preencher), `Indexar` (marca as `Faixa`s) |
| Leitura | `Presente`, `Faixa`, `Campo` (janela, sem copiar), `Saida` |
| Escrita | `Anexar`, `AnexarInt`, `AnexarTLV` |
| Composição e posse | `Ligar` (fan-in), `Promover`, `Soltar`, `Acoplar`, `Solto`, `Marcas` |

## 4. Exemplo 1: uma fatura de item

**Problema.** Dado preço (centavos), quantidade, desconto (centavos) e uma taxa em milésimos, calcular o valor líquido e o imposto, arredondando o imposto para o par mais próximo.

**Como pensar.** Em vez de uma função `CalcularFatura(...)`, desenhe os passos de um cálculo de caderno, cada um um átomo:

```go
const (
    tPreco quadro.Tag = iota + 1
    tQtd
    tDesconto
    tTaxa
    tMil // divisor, vem na mensagem (ver "Limitações")
    tLiquido
    tImposto
)

var faturaItem = pipeline.Declarar("fatura-item",
    []quadro.Tag{tPreco, tQtd, tDesconto, tTaxa, tMil}, // entradas exigidas
    operadores.Converter(operadores.R0, tPreco),                     // R0 = preço
    operadores.Converter(operadores.R1, tQtd),                       // R1 = quantidade
    operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1), // R2 = bruto
    operadores.Converter(operadores.R3, tDesconto),                  // R3 = desconto
    operadores.Subtrair(operadores.R2, operadores.R2, operadores.R3),    // R2 = líquido
    operadores.Escrever(tLiquido, operadores.R2),                    // saída: líquido
    operadores.Converter(operadores.R4, tTaxa),                      // R4 = taxa
    operadores.Multiplicar(operadores.R5, operadores.R2, operadores.R4), // R5 = líquido × taxa
    operadores.Converter(operadores.R6, tMil),                       // R6 = 1000
    operadores.Arredondar(operadores.R7, operadores.R5, operadores.R6, operadores.MeioPar),
    operadores.Escrever(tImposto, operadores.R7),                    // saída: imposto
)
```

**Execução** com preço 1990, quantidade 3, desconto 500, taxa 75, divisor 1000. Os registradores depois de cada etapa:

| Etapa | Átomo | R0 | R1 | R2 | R3 | R4 | R5 | R6 | R7 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| 0 | `Converter(R0, tPreco)` | 1990 | | | | | | | |
| 1 | `Converter(R1, tQtd)` | 1990 | 3 | | | | | | |
| 2 | `Multiplicar(R2, R0, R1)` | 1990 | 3 | 5970 | | | | | |
| 3 | `Converter(R3, tDesconto)` | 1990 | 3 | 5970 | 500 | | | | |
| 4 | `Subtrair(R2, R2, R3)` | 1990 | 3 | **5470** | 500 | | | | |
| 5 | `Escrever(tLiquido, R2)` | | | 5470 | | | | | |
| 6 | `Converter(R4, tTaxa)` | | | 5470 | 500 | 75 | | | |
| 7 | `Multiplicar(R5, R2, R4)` | | | 5470 | 500 | 75 | 410250 | | |
| 8 | `Converter(R6, tMil)` | | | | | | 410250 | 1000 | |
| 9 | `Arredondar(R7, R5, R6, MeioPar)` | | | | | | 410250 | 1000 | **410** |
| 10 | `Escrever(tImposto, R7)` | | | | | | | | 410 |

`410250 / 1000 = 410,25`, que o `MeioPar` arredonda para **410**. A saída do quadro, em TLV, é `[tag 6][len 4]"5470"` seguido de `[tag 7][len 3]"410"`.

**Chamada:**

```go
q := quadro.Pegar()
defer quadro.Devolver(q)
// ... externo.Receber(conn, q); entrada.Verificar(q, &contrato) ...
if err := faturaItem.Executar(q); err != nil {
    return err
}
externo.Enviar(conn, q) // 5470 e 410, escritos no mesmo buffer da entrada
```

## 5. Exemplo 2: como a cadeia falha

A mesma receita, com entradas ruins. O erro sempre diz **onde**:

| Entrada | Resultado |
| --- | --- |
| Faltam desconto, taxa e divisor | `fatura-item: aguardando entradas (tag 3)`. Nenhum átomo rodou |
| Preço `19,90` | `fatura-item: etapa 0: operadores: formato inválido`. A etapa 0 é o `Converter` do preço |
| Preço e quantidade `9999999999999` | `etapa 2: operadores: estouro de inteiro`. Nunca dá a volta em silêncio |
| Divisor (`tMil`) igual a `0` | `etapa 9: operadores: divisão por zero` |

Trate assim:

```go
err := faturaItem.Executar(q)
switch {
case errors.Is(err, pipeline.ErrAguardando):
    // faltam entradas: espera, não é falha
case errors.Is(err, operadores.ErrFormato), errors.Is(err, operadores.ErrAusente):
    // entrada ruim: o contrato do firewall deveria ter barrado antes
case errors.Is(err, operadores.ErrEstouro):
    // conta que não cabe em int64: recuse o comando
case err != nil:
    // bug ou erro novo: registre com a mensagem inteira (traz a etapa)
}
```

## 6. Exemplo 3: duas pipelines, uma entidade (`Juncao`)

Uma pipeline entrega o preço, outra a quantidade, e a receita só dispara quando as duas chegaram:

```go
soma := pipeline.Declarar("soma", []quadro.Tag{tA, tB},
    operadores.Converter(operadores.R0, tA),
    operadores.Converter(operadores.R1, tB),
    operadores.Somar(operadores.R2, operadores.R0, operadores.R1),
    operadores.Escrever(tSoma, operadores.R2),
)
j := pipeline.NovaJuncao(soma, 1024) // até 1024 entidades pendentes

// Chegou o pedaço B da entidade 42. Falta A.
q, err := j.Entregar(42, quadroComB)
// q == nil, errors.Is(err, pipeline.ErrAguardando)  → a junção passou a ser dona de quadroComB

// Chegou o pedaço A. As entradas fecharam: a receita roda agora.
q, err = j.Entregar(42, quadroComA)
// err == nil, q.Saida() tem tSoma. Quem recebeu q:
quadro.Devolver(q) // sempre, inclusive se err != nil
```

`Ligar` faz `quadroComA` e `quadroComB` virarem **um** quadro lógico: os campos de cada um continuam onde estão, e o `Campo(tA)` da receita lê direto do buffer de origem. Nada é copiado. Tabela cheia devolve `quadro.ErrCheio`: é contrapressão, e quem entregou continua dono do quadro.

## 7. Exemplo 4: passar o quadro para outra pipeline (`Transmitir`)

`Transmitir` é o **último** passo da cadeia. Ele solta o quadro e o entrega; quem o recebe o acopla e continua:

```go
rota := interno.Nova(64)

origem := pipeline.Declarar("origem", []quadro.Tag{tPreco, tQtd},
    operadores.Converter(operadores.R0, tPreco),
    operadores.Converter(operadores.R1, tQtd),
    operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1),
    operadores.Escrever(tTotal, operadores.R2),
    operadores.Transmitir(idOrigem, rota.Encaminhar), // último passo
)

// Na pipeline de destino, o trabalhador:
for q := range rota.Recebidos() {
    _ = q.Acoplar(idDestino) // agora o quadro é desta pipeline
    // ... destino.Executar(q) ...
    quadro.Devolver(q)
}
```

Regras de posse:

- Depois de `origem.Executar(q)` retornar `nil`, **a origem não toca mais em `q`**, nem para ler. Outra goroutine pode estar usando.
- Rota cheia **não é devolução**: o quadro nunca saiu. `Executar` devolve `ErrRotaCheia`, `q` continua acoplado e da origem, e a origem pode tentar de novo ou recusar o comando.
- Um passo depois do `Transmitir` falha com `quadro.ErrSolto`, sem ler o quadro.
- O rastro (`q.Marcas`) guarda os últimos 8 saltos: `{origem soltou}, {destino acoplou}`.

## 8. Como escrever uma receita

1. **Só operadores.** A receita é uma lista de átomos da biblioteca. Se faltar uma instrução, ela entra em `operadores/`; ninguém escreve `func(*quadro.Quadro) error` fora de lá (`TestSoAOperadoresDefineInstrucoes` reprova).
2. **Declare todas as entradas.** `entradas` lista cada tag que algum `Converter` lê. Sem isso, a receita roda e falha no meio, em vez de esperar.
3. **Constantes de tag com nome.** `tPreco`, nunca `1`. Tags valem de 0 a 63.
4. **Desenhe os registradores.** Faça a tabela do exemplo 1 no papel. São 8 (`R0` a `R7`); reuse um que não será mais lido (`Subtrair(R2, R2, R3)` sobrescreve o bruto).
5. **Escreva na saída só o que o chamador pediu**, na ordem em que a resposta deve sair.
6. **Dinheiro é inteiro em centavos.** Divisão que pode sobrar resto usa `Arredondar` com o modo escrito na receita, nunca `Dividir`.
7. **`Transmitir`, se houver, é o último passo.**

Bom e ruim:

```go
// ERRADO: função grande que refaz o trabalho dos átomos por fora
func totalItem(q *quadro.Quadro) error {
    preco, _ := q.Campo(tPreco)
    p, _ := strconv.Atoi(string(preco)) // copia o campo e aloca
    ...
}

// CERTO: a mesma conta, em átomos
pipeline.Declarar("total-item", []quadro.Tag{tPreco, tQtd},
    operadores.Converter(operadores.R0, tPreco),
    operadores.Converter(operadores.R1, tQtd),
    operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1),
    operadores.Escrever(tTotal, operadores.R2),
)
```

```go
// ERRADO: duas receitas ao mesmo tempo sobre o mesmo quadro dividem Reg
go receitaA.Executar(q)
go receitaB.Executar(q) // sobrescreve R0..R7 de A (pendência 1 do ADR 0002)

// CERTO: um quadro por execução
go receitaA.Executar(qA)
go receitaB.Executar(qB)
```

### Conferir se a receita está decomposta

`reflexo.Ler(dir)` lê o código-fonte e devolve cada `pipeline.Declarar`. Para cada `Receita`, `Problemas()` aponta o que foge desta seção (etapa que não é chamada a `operadores.X`, tag lida e não declarada, `Transmitir` fora do fim, receita sem saída) e `Reflexo()` mostra como ela deveria estar descrita, com ✓ ou ✗ por etapa. `TestReceitasUsamOperadoresAoMaximo` roda isso em todo o repositório. A análise é estática, pelo nome do pacote `operadores`.

## 9. Como escrever um operador novo

Só quando uma receita real pedir (ADR 0001). Todo operador segue o mesmo molde: **valida na montagem, devolve um closure, e o closure só lê registradores e campos por faixa**. Exemplo ilustrativo (`Negar` não existe na biblioteca):

```go
// Negar faz dst = -a.
func Negar(dst, a Reg) Op {
    regs(dst, a) // registrador inválido quebra aqui, ao montar
    return func(q *quadro.Quadro) error {
        x := q.Reg[a]
        if x == math.MinInt64 { // -MinInt64 não cabe em int64
            return ErrEstouro
        }
        q.Reg[dst] = -x
        return nil
    }
}
```

Checklist de um operador novo:

- [ ] Fica em `operadores/` e importa só a stdlib e `quadro`.
- [ ] Chama `regs(...)` com todo registrador recebido, e valida qualquer outro parâmetro (modo, tag) na montagem, com `panic`.
- [ ] Confere estouro e divisão por zero; devolve um `ErrXxx` existente, ou um novo com prefixo `operadores:`.
- [ ] Lê campo com `q.Campo(tag)` e **não** converte para `string` nem copia.
- [ ] Escreve só em `q.Reg` ou pela saída do quadro (`AnexarInt`, `Anexar`).
- [ ] Tem teste de tabela (caso feliz, cada erro, extremos como `MinInt64`) e prova de 0 alocações.
- [ ] Documentado em uma linha neste guia e no README.

Teste mínimo, no estilo do repositório:

```go
func TestNegar(t *testing.T) {
    q := carregar(t, nil)
    q.Reg[operadores.R0] = 5
    if err := operadores.Negar(operadores.R1, operadores.R0)(q); err != nil || q.Reg[operadores.R1] != -5 {
        t.Fatalf("Reg = %d, erro = %v", q.Reg[operadores.R1], err)
    }
    q.Reg[operadores.R0] = math.MinInt64
    if err := operadores.Negar(operadores.R1, operadores.R0)(q); !errors.Is(err, operadores.ErrEstouro) {
        t.Fatalf("erro = %v, quero ErrEstouro", err)
    }
}
```

## 10. Limitações de hoje

- **Não há operador de constante.** Todo número de uma conta vem de um campo da mensagem (por isso `tMil` no exemplo 1). Quando uma receita real pedir, entra um `Constante(dst, valor)`.
- **`Converter` não aceita sinal.** Valores de entrada são contagens e centavos. Negativo só nasce de `Subtrair` e aparece na saída em decimal.
- **`Reg` mora no quadro:** uma execução por quadro (pendência 1 do ADR 0002).
- **`Transmitir` leva o quadro inteiro.** A transmissão por campo, que deixa um tipo seguir sem esperar os outros, está decidida no papel e não implementada.
- **No máximo 64 tags, 8 registradores, 4 origens por `Ligar`.** `MaxOrigens = 4` não cobre 10 tipos por struct (pendência 4).
- **A `Juncao` executa a receita na goroutine de quem entrega a última entrada** (pendência 3).
