package quadro

import "errors"

// MaxMarcas é quantos saltos recentes o quadro lembra; passando disso, a
// marca mais antiga é sobrescrita. Fixo, para o rastro nunca alocar.
const MaxMarcas = 8

var (
	ErrSolto    = errors.New("quadro: solto, não pertence mais a esta pipeline")
	ErrAcoplado = errors.New("quadro: já está acoplado a uma pipeline")
)

// Marca registra um encaixe ou desencaixe do quadro numa pipeline. A ordem
// do rastro é a ordem em que aconteceram.
type Marca struct {
	Pipeline uint16
	Acoplou  bool // false: a pipeline soltou o quadro
}

// Solto diz se o quadro está em trânsito: nenhuma pipeline é dona dele.
func (q *Quadro) Solto() bool { return q.solto }

// Soltar desencaixa o quadro da pipeline: depois disso quem soltou não
// escreve mais nele. Só o Transmitir chama.
func (q *Quadro) Soltar(pipeline uint16) error {
	if q.solto {
		return ErrSolto
	}
	q.solto = true
	q.marcar(Marca{Pipeline: pipeline})
	return nil
}

// Acoplar encaixa o quadro numa pipeline. Só o Router chama, ao entregar o
// quadro (ou a entidade completa da junção) à pipeline de destino.
func (q *Quadro) Acoplar(pipeline uint16) error {
	if !q.solto {
		return ErrAcoplado
	}
	q.solto = false
	q.marcar(Marca{Pipeline: pipeline, Acoplou: true})
	return nil
}

// Marcas anexa a dst o rastro, da marca mais antiga lembrada para a mais
// nova. Com dst de capacidade suficiente não aloca.
func (q *Quadro) Marcas(dst []Marca) []Marca {
	for i := max(0, q.nMarcas-MaxMarcas); i < q.nMarcas; i++ {
		dst = append(dst, q.marcas[i%MaxMarcas])
	}
	return dst
}

func (q *Quadro) marcar(m Marca) {
	q.marcas[q.nMarcas%MaxMarcas] = m
	q.nMarcas++
}
