// Package operadores é o conjunto de instruções do KE: funções pequenas e
// determinísticas sobre um *quadro.Quadro. Um processo é uma sequência delas
// (Seq), nunca uma função grande que refaz o trabalho por fora (spec v2
// §1.1, o princípio do ISA). O número de operadores é pequeno e previsível;
// o que diferencia um processo de outro é a ordem.
//
// É biblioteca: não conhece firewall, núcleo, plataforma nem negócio, e só
// importa a stdlib e o quadro. Qualquer projeto importa e monta as próprias
// receitas (ADR 0001). Operador novo só entra quando uma receita real pedir.
//
// Os operadores leem campos pela posição no buffer (quadro.Campo) e guardam
// valores intermediários nos registradores do quadro; nenhum copia o dado.
package operadores

import (
	"errors"
	"fmt"

	"github.com/lorkyrr/ke/quadro"
)

// Op é uma instrução. Recebe o endereço do quadro e opera no lugar.
type Op func(*quadro.Quadro) error

// Reg indexa o banco de registradores do quadro.
type Reg uint8

const (
	R0 Reg = iota
	R1
	R2
	R3
	R4
	R5
	R6
	R7
)

var (
	ErrAusente = errors.New("operadores: campo ausente")
	ErrFormato = errors.New("operadores: formato inválido")
	ErrEstouro = errors.New("operadores: estouro de inteiro")

	ErrDivisaoPorZero = errors.New("operadores: divisão por zero")
	ErrInexato        = errors.New("operadores: divisão não é exata")
)

// Seq compõe operadores em ordem; para no primeiro erro, dizendo em qual
// etapa. É a única forma de montar um processo.
func Seq(ops ...Op) Op {
	return func(q *quadro.Quadro) error {
		for i, op := range ops {
			if err := op(q); err != nil {
				return fmt.Errorf("etapa %d: %w", i, err)
			}
		}
		return nil
	}
}

// regs confere na declaração (não na execução) que o registrador existe:
// receita errada quebra ao montar o programa, nunca no meio de um comando.
func regs(rs ...Reg) {
	for _, r := range rs {
		if int(r) >= quadro.MaxRegs {
			panic(fmt.Sprintf("operadores: registrador %d fora de 0..%d", r, quadro.MaxRegs-1))
		}
	}
}
