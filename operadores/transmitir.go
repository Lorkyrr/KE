package operadores

import (
	"errors"

	"github.com/lorkyrr/ke/quadro"
)

// ErrTransmitido não é falha: avisa ao Seq que o quadro saiu para outra
// pipeline e que ninguém mais pode tocar nele, nem para ler. Seq e
// pipeline.Receita.Executar o tratam como sucesso quando Transmitir é o
// último passo.
var ErrTransmitido = errors.New("operadores: quadro transmitido")

// Entrega leva o quadro à pipeline de destino. Quem implementa é o Router
// (por exemplo, interno.Rota.Encaminhar). Em caso de erro a posse continua
// com quem chamou; em sucesso o quadro é de quem recebeu.
type Entrega func(*quadro.Quadro) error

// Transmitir é o único operador de passagem entre pipelines (ADR 0002). Solta
// o quadro da pipeline e o entrega; o destino o encaixa com Acoplar. Sentido
// único: não há devolução. Entrega recusada (ErrCheio, rota cheia) não é
// devolução: o quadro nunca saiu, volta acoplado à origem e o erro sobe.
//
// Tem de ser o último passo do Seq: depois dele o quadro não é mais desta
// pipeline, e um passo seguinte falha com quadro.ErrSolto.
func Transmitir(pipeline uint16, entrega Entrega) Op {
	if entrega == nil {
		panic("operadores: Transmitir sem entrega")
	}
	return func(q *quadro.Quadro) error {
		if err := q.Soltar(pipeline); err != nil {
			return err
		}
		if err := entrega(q); err != nil {
			// Soltar acabou de marcar o quadro, então Acoplar não falha.
			_ = q.Acoplar(pipeline)
			return err
		}
		return ErrTransmitido
	}
}
