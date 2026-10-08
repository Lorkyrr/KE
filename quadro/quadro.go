// Package quadro é a peça de memória do KE: um buffer por comando e o índice
// de onde cada campo está nele. Os módulos não copiam o dado entre si; passam
// o endereço do quadro (*Quadro) e as posições (Faixa) dentro do buffer. O
// dado entra uma vez (leitura do transporte) e a saída é escrita no mesmo
// buffer, depois da entrada.
//
// É biblioteca: só stdlib, sem firewall, plataforma nem negócio. Qualquer
// projeto importa (ADR 0001).
package quadro

import (
	"errors"
	"sync"
)

const (
	// MaxTags cabe num uint64 de presença: tags de 0 a 63.
	MaxTags = 64
	// MaxRegs é o tamanho do banco de registradores que os operadores usam
	// para resultados intermediários (valores, não bytes).
	MaxRegs = 8
	// Capacidade é fixa: o buffer nunca cresce, então nenhum append realoca
	// e move o dado por trás de quem guarda uma Faixa.
	Capacidade = 64 << 10
)

// Tag identifica um campo. Em TLV, a tag é a declaração do campo.
type Tag uint16

// Faixa é o endereço de um campo dentro do buffer do quadro.
type Faixa struct{ Ini, Fim uint32 }

// Tamanho é o número de bytes do campo.
func (f Faixa) Tamanho() int { return int(f.Fim - f.Ini) }

var (
	ErrCorte    = errors.New("quadro: campo passa do fim da entrada")
	ErrTag      = errors.New("quadro: tag fora do intervalo")
	ErrRepetida = errors.New("quadro: tag repetida")
	ErrCheio    = errors.New("quadro: sem capacidade")
	ErrOcupado  = errors.New("quadro: já tem entrada")
)

// Quadro guarda um comando inteiro. len(buf) é o que já foi escrito
// (entrada + saída); cap(buf) é sempre Capacidade.
type Quadro struct {
	buf       []byte
	entrada   int
	faixas    [MaxTags]Faixa
	presentes uint64
	Reg       [MaxRegs]int64
}

// Novo cria um quadro fora do pool (testes, uso isolado).
func Novo() *Quadro { return &Quadro{buf: make([]byte, 0, Capacidade)} }

var pool = sync.Pool{New: func() any { return Novo() }}

// Pegar devolve um quadro vazio do pool.
func Pegar() *Quadro { return pool.Get().(*Quadro) }

// Devolver zera e devolve ao pool. Depois disso, quem devolveu não toca
// mais no quadro nem em nenhuma fatia obtida dele.
func Devolver(q *Quadro) {
	q.Zerar()
	pool.Put(q)
}

// Zerar sobrescreve os bytes usados (dado pessoal não fica em memória
// reaproveitada, spec v2 §7.5) e limpa o índice e os registradores.
func (q *Quadro) Zerar() {
	clear(q.buf)
	q.buf = q.buf[:0]
	q.entrada = 0
	q.faixas = [MaxTags]Faixa{}
	q.presentes = 0
	q.Reg = [MaxRegs]int64{}
}

// Reservar marca n bytes de entrada e devolve a área para o transporte
// preencher direto (io.ReadFull), sem buffer intermediário.
func (q *Quadro) Reservar(n int) ([]byte, error) {
	if len(q.buf) != 0 {
		return nil, ErrOcupado
	}
	if n < 0 || n > cap(q.buf) {
		return nil, ErrCheio
	}
	q.buf = q.buf[:n]
	q.entrada = n
	return q.buf[:n:n], nil
}

// Presente diz se a tag apareceu na entrada indexada.
func (q *Quadro) Presente(t Tag) bool {
	return t < MaxTags && q.presentes&(1<<t) != 0
}

// Faixa devolve a posição do campo na entrada.
func (q *Quadro) Faixa(t Tag) (Faixa, bool) {
	if !q.Presente(t) {
		return Faixa{}, false
	}
	return q.faixas[t], true
}

// Campo devolve uma janela sobre o campo, sem copiar. A capacidade é
// cortada no fim do campo: um append de quem recebe não escreve por cima
// do campo vizinho, aloca outro array.
func (q *Quadro) Campo(t Tag) ([]byte, bool) {
	f, ok := q.Faixa(t)
	if !ok {
		return nil, false
	}
	return q.buf[f.Ini:f.Fim:f.Fim], true
}

// Saida é o que foi anexado depois da entrada, pronto para o transporte.
func (q *Quadro) Saida() []byte {
	return q.buf[q.entrada:len(q.buf):len(q.buf)]
}
