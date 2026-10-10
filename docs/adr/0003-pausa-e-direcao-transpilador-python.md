# 0003. Pausa do KE e nova direção: biblioteca de transpilação para Python

- **Status:** aceita (registro de intenção; o desenho da nova direção não existe)
- **Data:** 2026-10-10

## Contexto

O KE foi desenhado como biblioteca de quadros, operadores e receitas (ADR 0001
e 0002). A ideia por trás: escrever o processo como sequência de funções
mínimas, para poder criar centenas de rotinas (goroutines) de baixo custo.

Em 2026-10-10 foram medidos o custo e a escala dessa estrutura, numa máquina
AMD Ryzen 7 7445HS (12 threads), com a receita `fatura-item` do guia (11
operadores) contra a mesma conta numa função Go nativa:

| Versão | ns/op sem a recarga do quadro | Alocações |
| --- | --- | --- |
| Receita (11 operadores) | ~76 | 0 |
| Nativa, com a mesma checagem de estouro | ~60 | 0 |
| Nativa crua, sem checagem | ~48 | 0 |

- A chamada de um operador pela `Seq` é indireta (`op(q)` sobre um `func`
  guardado em lista). O compilador não faz inline entre operadores, e
  `Receita.Executar` não entra em inline (custo 360 contra o limite de 80).
  Dentro de cada operador os helpers são fundidos.
- Em paralelo, com um quadro por goroutine, a receita ganha 5,7× de 1 para 12
  threads e a nativa 5,5×. A receita não cria contenção.
- Cada goroutine que segura um quadro custa cerca de 64 KiB: 10.000 vivas
  ocupam ~637 MiB. O limite de rotinas simultâneas é a memória dos quadros.
- A função nativa equivalente não pode existir neste repositório: o teste ISA
  (`TestSoAOperadoresDefineInstrucoes`) a reprova.

## Decisão

1. **O KE fica parado.** Nenhuma funcionalidade nova entra até a nova direção
   ter um desenho escrito. O código, os testes e as regras atuais ficam como
   estão.
2. **A estrutura modular atual não é o caminho para criar centenas de
   rotinas.** A expectativa do autor era essa; a estrutura em operadores e
   receitas não entrega isso do jeito pensado.
3. **A intenção é continuar como biblioteca de transpilação para Python:**
   - o usuário escreve a estrutura em **classes Python**;
   - essa estrutura é **compilada para Go**;
   - o Go roda **dentro do interpretador Python**;
   - o resultado é uma **multi-thread leve para Python**.

## Em aberto

Nada abaixo está decidido. São perguntas para o desenho futuro, a escrever
antes de qualquer código:

1. Como o Go entra no interpretador Python (extensão nativa, outro
   mecanismo) e como isso é distribuído.
2. Que subconjunto de classes Python é aceito pelo transpilador, e como o
   que fica fora é recusado (nada se perde em silêncio).
3. Como dados e erros cruzam a fronteira Python/Go sem cópia, ou onde a
   cópia é aceita.
4. O que do KE atual (quadro, operadores, receitas, firewall) é reaproveitado
   e o que é descartado. Hoje nada disso foi decidido.
5. Como as goroutines aparecem para o código Python (nome, API, ciclo de
   vida, cancelamento).

## Consequências

- `README.md`, `CLAUDE.md` e `.planning/ROADMAP.md` dizem que o projeto está
  em pausa e apontam para este ADR.
- As pendências do ADR 0002 deixam de ser trabalho corrente. Continuam
  válidas se o KE atual for retomado.
- As medições acima são a referência para comparar a nova direção.
