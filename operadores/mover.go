package operadores

import (
	"math"

	"github.com/lorkyrr/ke/quadro"
)

// Converter lê os dígitos ASCII do campo direto da faixa no buffer e guarda
// o inteiro no registrador. Sem string intermediária, sem sinal: valores de
// entrada são contagens e centavos.
func Converter(dst Reg, t quadro.Tag) Op {
	regs(dst)
	return func(q *quadro.Quadro) error {
		v, ok := q.Campo(t)
		if !ok {
			return ErrAusente
		}
		if len(v) == 0 {
			return ErrFormato
		}
		var n int64
		for _, c := range v {
			if c < '0' || c > '9' {
				return ErrFormato
			}
			d := int64(c - '0')
			if n > (math.MaxInt64-d)/10 {
				return ErrEstouro
			}
			n = n*10 + d
		}
		q.Reg[dst] = n
		return nil
	}
}

// Escrever anexa o registrador à saída do quadro, em decimal.
func Escrever(t quadro.Tag, src Reg) Op {
	regs(src)
	return func(q *quadro.Quadro) error {
		return q.AnexarInt(t, q.Reg[src])
	}
}
