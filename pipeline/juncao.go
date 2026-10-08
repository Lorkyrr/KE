package pipeline

import (
	"fmt"
	"sync"

	"github.com/lorkyrr/ke/quadro"
)

// Juncao é a entrada sincronizada de uma pipeline: recebe, de pipelines
// anteriores, os pedaços de uma mesma entidade e só dispara a receita quando
// todas as entradas exigidas chegaram. Antes disso nada roda; depois, a
// receita corre livre. É o único ponto de espera entre pipelines.
//
// A tabela de entidades pendentes tem tamanho fixo: cheia, recusa em vez de
// crescer, e a recusa chega a quem entrega como contrapressão.
type Juncao struct {
	receita Receita
	mu      sync.Mutex
	vagas   []vaga
	usadas  int
}

type vaga struct {
	chave uint64
	q     *quadro.Quadro // nil: livre
}

// NovaJuncao cria a junção de r com espaço para capacidade entidades pendentes.
func NovaJuncao(r Receita, capacidade int) *Juncao {
	return &Juncao{receita: r, vagas: make([]vaga, capacidade)}
}

// Pendentes é quantas entidades esperam entradas.
func (j *Juncao) Pendentes() int {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.usadas
}

// Entregar junta o quadro o à entidade chave. Sem as entradas todas, devolve
// ErrAguardando. Quando a última chega, executa a receita na goroutine de
// quem entregou e devolve o quadro da entidade (com a Saida); quem recebe
// chama quadro.Devolver nele, mesmo se o erro não for nulo.
//
// Em sucesso ou espera, a posse de o passa à junção. Em qualquer outro erro
// (tag repetida, tabela cheia), o continua de quem entregou e a entidade
// segue como estava.
func (j *Juncao) Entregar(chave uint64, o *quadro.Quadro) (*quadro.Quadro, error) {
	j.mu.Lock()
	v := j.achar(chave)
	if v == nil {
		j.mu.Unlock()
		return nil, quadro.ErrCheio
	}
	nova := v.q == nil
	if nova {
		*v = vaga{chave: chave, q: quadro.Pegar()}
		j.usadas++
	}
	if err := v.q.Ligar(o); err != nil {
		if nova {
			j.liberar(v) // sem origens, Devolver só zera o quadro novo
		}
		j.mu.Unlock()
		return nil, err
	}
	if t, falta := j.receita.faltando(v.q); falta {
		j.mu.Unlock()
		return nil, fmt.Errorf("%s: %w (tag %d)", j.receita.Nome, ErrAguardando, t)
	}
	q := v.q
	*v = vaga{}
	j.usadas--
	j.mu.Unlock()
	return q, j.receita.Executar(q)
}

// achar devolve a vaga da entidade ou, se ela é nova, a primeira livre.
func (j *Juncao) achar(chave uint64) *vaga {
	var livre *vaga
	for i := range j.vagas {
		v := &j.vagas[i]
		switch {
		case v.q != nil && v.chave == chave:
			return v
		case v.q == nil && livre == nil:
			livre = v
		}
	}
	return livre
}

func (j *Juncao) liberar(v *vaga) {
	quadro.Devolver(v.q)
	*v = vaga{}
	j.usadas--
}
