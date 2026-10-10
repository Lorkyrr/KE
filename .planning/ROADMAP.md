# Roadmap

Índice do que está feito. A próxima sessão lê isto antes de reexplorar o repositório.

## Estado

**Em pausa desde 2026-10-10.** Nenhuma funcionalidade nova até existir um desenho escrito da nova direção ([ADR 0003](../docs/adr/0003-pausa-e-direcao-transpilador-python.md)).

## Intenção

Continuar como biblioteca de transpilação para Python: uma estrutura de classes Python compilada para Go, com o Go dentro do interpretador Python, dando uma multi-thread leve ao Python. Perguntas em aberto no ADR 0003.

## Feito

- Quadro com faixas, registradores, `Ligar`, `Promover`, posse e rastro (ADR 0001, 0002).
- Operadores: aritmética, `Converter`, `Escrever`, `Arredondar`, `Seq`, `Transmitir` (quadro inteiro).
- `pipeline`: `Receita` e `Juncao`.
- Firewall: sanitizar, contrato por tag, transporte externo e rota interna.
- `reflexo`: verificador estático de receitas (`TestReceitasUsamOperadoresAoMaximo`).
- Guia das funções, diretrizes e medições de custo e escala (ADR 0003).

## Parado

- Transmissão por campo e as pendências do ADR 0002.
