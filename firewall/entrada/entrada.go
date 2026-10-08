// Package entrada aplica o contrato declarado (spec v2 §3.2, D3): cada tag
// declara tamanho máximo, se é obrigatória e qual sanitizador a revista. Tag
// sem declaração derruba o comando. Tudo é feito sobre as faixas do quadro;
// nenhum campo é copiado.
package entrada

import (
	"fmt"

	"github.com/lorkyrr/ke/quadro"
)

// Campo é a declaração de uma tag. Max == 0 quer dizer "não declarada".
type Campo struct {
	Max         int
	Obrigatorio bool
	Sanitizar   func([]byte) error
}

// Contrato é indexado pela tag: consultar é acessar uma posição do array.
type Contrato [quadro.MaxTags]Campo

// Recusa diz qual campo derrubou o comando e por quê. Nunca carrega o
// valor do campo (pode ser dado pessoal).
type Recusa struct {
	Tag    quadro.Tag
	Motivo string
	Err    error
}

func (r *Recusa) Error() string {
	if r.Err != nil {
		return fmt.Sprintf("campo %d: %s: %v", r.Tag, r.Motivo, r.Err)
	}
	return fmt.Sprintf("campo %d: %s", r.Tag, r.Motivo)
}

func (r *Recusa) Unwrap() error { return r.Err }

// Verificar indexa o quadro e aplica o contrato, na ordem barata primeiro:
// declaração, presença, tamanho e só então o formato.
func Verificar(q *quadro.Quadro, c *Contrato) error {
	if err := q.Indexar(); err != nil {
		return &Recusa{Motivo: "mensagem malformada", Err: err}
	}
	for t := range quadro.Tag(quadro.MaxTags) {
		campo := &c[t]
		v, ok := q.Campo(t)
		switch {
		case !ok && campo.Obrigatorio:
			return &Recusa{Tag: t, Motivo: "obrigatório ausente"}
		case !ok:
			continue
		case campo.Max == 0:
			return &Recusa{Tag: t, Motivo: "não declarado"}
		case len(v) > campo.Max:
			return &Recusa{Tag: t, Motivo: fmt.Sprintf("maior que %d bytes", campo.Max)}
		case campo.Sanitizar != nil:
			if err := campo.Sanitizar(v); err != nil {
				return &Recusa{Tag: t, Motivo: "formato", Err: err}
			}
		}
	}
	return nil
}
