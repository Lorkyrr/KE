// Package pipeline é a pipeline padrão do núcleo (spec v2 §7.5, D23): executa
// uma receita, um comando por vez. Não interpreta nada e não conhece
// negócio; a receita é de quem a declara.
package pipeline

import (
	"errors"
	"fmt"

	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

// ErrAguardando: faltam entradas. Não é falha, é o estado de espera do D16;
// as etapas só disparam quando todas as entradas chegaram.
var ErrAguardando = errors.New("aguardando entradas")

// Receita é o nome, as entradas exigidas e as etapas em ordem.
type Receita struct {
	Nome     string
	entradas []quadro.Tag
	etapas   operadores.Op
}

// Declarar monta a receita só com operadores (a regra ISA).
func Declarar(nome string, entradas []quadro.Tag, etapas ...operadores.Op) Receita {
	return Receita{Nome: nome, entradas: entradas, etapas: operadores.Seq(etapas...)}
}

// Executar roda a receita sobre o quadro.
func (r Receita) Executar(q *quadro.Quadro) error {
	for _, t := range r.entradas {
		if !q.Presente(t) {
			return fmt.Errorf("%s: %w (tag %d)", r.Nome, ErrAguardando, t)
		}
	}
	if err := r.etapas(q); err != nil {
		return fmt.Errorf("%s: %w", r.Nome, err)
	}
	return nil
}
