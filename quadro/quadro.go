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
	// MaxOrigens é quantos quadros um quadro de junção enxerga (fan-in).
	MaxOrigens = 4
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
	ErrOrigem   = errors.New("quadro: origem inválida")
)

// Quadro guarda um comando inteiro. len(buf) é o que já foi escrito
// (entrada + saída); cap(buf) é sempre Capacidade.
type Quadro struct {
	buf       []byte
	entrada   int
	faixas    [MaxTags]Faixa
	presentes uint64 // tags indexadas neste buffer
	visiveis  uint64 // presentes + as de todas as origens
	origens   [MaxOrigens]*Quadro
	nOrigens  int
	solto     bool // em trânsito entre pipelines (posse.go)
	marcas    [MaxMarcas]Marca
	nMarcas   int
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
	for _, o := range q.origens[:q.nOrigens] {
		Devolver(o)
	}
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
	q.visiveis = 0
	q.origens = [MaxOrigens]*Quadro{}
	q.nOrigens = 0
	q.solto = false
	q.marcas = [MaxMarcas]Marca{}
	q.nMarcas = 0
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

// Ligar faz o quadro enxergar os campos de o sem copiá-los (fan-in da
// junção). A posse de o passa para q: ninguém mais o usa, e Devolver(q)
// devolve as origens junto. Tag que já é visível em q recusa com ErrRepetida.
func (q *Quadro) Ligar(o *Quadro) error {
	if o == nil || o == q {
		return ErrOrigem
	}
	if q.nOrigens == MaxOrigens {
		return ErrCheio
	}
	if q.visiveis&o.visiveis != 0 {
		return ErrRepetida
	}
	q.origens[q.nOrigens] = o
	q.nOrigens++
	q.visiveis |= o.visiveis
	return nil
}

// Presente diz se a tag é visível: indexada na entrada própria ou numa origem.
func (q *Quadro) Presente(t Tag) bool {
	return t < MaxTags && q.visiveis&(1<<t) != 0
}

// Faixa devolve a posição do campo no buffer próprio. Campo de origem não
// tem faixa aqui: a faixa só vale dentro do buffer que a indexou.
func (q *Quadro) Faixa(t Tag) (Faixa, bool) {
	if t >= MaxTags || q.presentes&(1<<t) == 0 {
		return Faixa{}, false
	}
	return q.faixas[t], true
}

// Campo devolve uma janela sobre o campo, sem copiar. A capacidade é
// cortada no fim do campo: um append de quem recebe não escreve por cima
// do campo vizinho, aloca outro array.
func (q *Quadro) Campo(t Tag) ([]byte, bool) {
	if f, ok := q.Faixa(t); ok {
		return q.buf[f.Ini:f.Fim:f.Fim], true
	}
	if !q.Presente(t) {
		return nil, false
	}
	for _, o := range q.origens[:q.nOrigens] {
		if v, ok := o.Campo(t); ok {
			return v, true
		}
	}
	return nil, false
}

// Saida é o que foi anexado depois da entrada, pronto para o transporte.
func (q *Quadro) Saida() []byte {
	return q.buf[q.entrada:len(q.buf):len(q.buf)]
}
