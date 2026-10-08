package operadores

import (
	"math"

	"github.com/lorkyrr/ke/quadro"
)

// Dinheiro é inteiro (centavos); toda conta confere estouro em vez de dar a
// volta em silêncio (spec v2 §7.2).

// Somar faz dst = a + b.
func Somar(dst, a, b Reg) Op {
	regs(dst, a, b)
	return func(q *quadro.Quadro) error {
		x, y := q.Reg[a], q.Reg[b]
		s := x + y
		if (y > 0 && s < x) || (y < 0 && s > x) {
			return ErrEstouro
		}
		q.Reg[dst] = s
		return nil
	}
}

// Multiplicar faz dst = a * b.
func Multiplicar(dst, a, b Reg) Op {
	regs(dst, a, b)
	return func(q *quadro.Quadro) error {
		p, err := multiplicar(q.Reg[a], q.Reg[b])
		if err != nil {
			return err
		}
		q.Reg[dst] = p
		return nil
	}
}

// multiplicar devolve x*y, ou ErrEstouro se o produto não cabe em int64.
func multiplicar(x, y int64) (int64, error) {
	if x == 0 || y == 0 {
		return 0, nil
	}
	// MinInt64 × -1 não cabe e ainda faria a divisão de conferência
	// estourar; o resto se confere desfazendo a conta.
	if (x == -1 && y == math.MinInt64) || (y == -1 && x == math.MinInt64) {
		return 0, ErrEstouro
	}
	p := x * y
	if p/y != x {
		return 0, ErrEstouro
	}
	return p, nil
}
