package operadores

import (
	"fmt"

	"github.com/lorkyrr/ke/quadro"
)

// Modo é a regra de arredondamento de um quociente inexato.
type Modo uint8

const (
	Truncar   Modo = iota // em direção a zero
	Piso                  // em direção a -∞
	Teto                  // em direção a +∞
	MeioAcima             // ao mais próximo; metade vai para longe de zero
	MeioPar               // ao mais próximo; metade vai para o par (bancário)
)

// Arredondar faz dst = a / b, arredondado pelo modo. Com divisão exata, todos
// os modos dão o mesmo resultado de Dividir.
func Arredondar(dst, a, b Reg, modo Modo) Op {
	regs(dst, a, b)
	if modo > MeioPar {
		panic(fmt.Sprintf("operadores: modo de arredondamento %d inexistente", modo))
	}
	return func(q *quadro.Quadro) error {
		x, y := q.Reg[a], q.Reg[b]
		quo, resto, err := dividirComResto(x, y)
		if err != nil {
			return err
		}
		if resto != 0 {
			quo += ajuste(modo, quo, resto, y, (x < 0) != (y < 0))
		}
		q.Reg[dst] = quo
		return nil
	}
}

// ajuste devolve -1, 0 ou +1 a somar ao quociente truncado. Como resto != 0,
// |y| >= 2 e o ajuste nunca estoura.
func ajuste(modo Modo, quo, resto, y int64, negativo bool) int64 {
	longe := int64(1) // sentido "para longe de zero"
	if negativo {
		longe = -1
	}
	switch modo {
	case Piso:
		if negativo {
			return longe
		}
	case Teto:
		if !negativo {
			return longe
		}
	case MeioAcima, MeioPar:
		// 2|resto| nunca passa de 2^64-1, então a comparação não estoura.
		dobro, divisor := 2*modulo(resto), modulo(y)
		if dobro > divisor || (dobro == divisor && (modo == MeioAcima || quo%2 != 0)) {
			return longe
		}
	}
	return 0
}

// modulo devolve |x| sem estourar em MinInt64.
func modulo(x int64) uint64 {
	if x < 0 {
		return -uint64(x)
	}
	return uint64(x)
}
