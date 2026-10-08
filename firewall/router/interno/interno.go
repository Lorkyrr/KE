// Package interno encaminha comandos dentro do KE (spec v2 §6, D22). Pela
// rota passa só o endereço do quadro (8 bytes), nunca o conteúdo.
//
// Posse: quem encaminha entrega o quadro junto com o ponteiro e não toca
// mais nele; quem recebe passa a ser o dono e o devolve ao pool no fim.
package interno

import (
	"errors"

	"github.com/lorkyrr/ke/quadro"
)

// ErrRotaCheia é o fail-fast da v1 (ErrFilaCheia): sob rajada a origem
// recebe recusa imediata em vez de travar esperando.
var ErrRotaCheia = errors.New("rota cheia")

type Rota struct{ c chan *quadro.Quadro }

// Nova cria uma rota com limite (backpressure).
func Nova(capacidade int) *Rota { return &Rota{c: make(chan *quadro.Quadro, capacidade)} }

// Encaminhar entrega o quadro à rota, ou recusa na hora se estiver cheia.
// Em caso de recusa, a posse continua com quem chamou.
func (r *Rota) Encaminhar(q *quadro.Quadro) error {
	select {
	case r.c <- q:
		return nil
	default:
		return ErrRotaCheia
	}
}

// Recebidos é o lado do consumidor.
func (r *Rota) Recebidos() <-chan *quadro.Quadro { return r.c }

// Fechar encerra a rota; só quem produz fecha.
func (r *Rota) Fechar() { close(r.c) }
