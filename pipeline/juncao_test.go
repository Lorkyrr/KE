package pipeline

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

const (
	tagA quadro.Tag = iota + 1
	tagB
	tagSoma
)

// origem faz o papel do quadro entregue por uma pipeline anterior.
func origem(t testing.TB, tag quadro.Tag, valor int) *quadro.Quadro {
	t.Helper()
	in := quadro.AnexarTLV(nil, tag, []byte(strconv.Itoa(valor)))
	q := quadro.Pegar()
	area, err := q.Reservar(len(in))
	if err != nil {
		t.Fatal(err)
	}
	copy(area, in)
	if err := q.Indexar(); err != nil {
		t.Fatal(err)
	}
	return q
}

func receitaSoma() Receita {
	return Declarar("soma", []quadro.Tag{tagA, tagB},
		operadores.Converter(operadores.R0, tagA),
		operadores.Converter(operadores.R1, tagB),
		operadores.Somar(operadores.R2, operadores.R0, operadores.R1),
		operadores.Escrever(tagSoma, operadores.R2))
}

func saidaEsperada(soma int) string {
	return string(quadro.AnexarTLV(nil, tagSoma, []byte(strconv.Itoa(soma))))
}

func TestJuncaoDisparaQuandoAsEntradasFecham(t *testing.T) {
	j := NovaJuncao(receitaSoma(), 4)
	q, err := j.Entregar(42, origem(t, tagB, 5))
	if !errors.Is(err, ErrAguardando) || q != nil {
		t.Fatalf("1ª entrega = %v, %v; quero ErrAguardando e nada para devolver", q, err)
	}
	if j.Pendentes() != 1 {
		t.Fatalf("pendentes = %d", j.Pendentes())
	}
	q, err = j.Entregar(42, origem(t, tagA, 7))
	if err != nil {
		t.Fatal(err)
	}
	defer quadro.Devolver(q)
	if got := string(q.Saida()); got != saidaEsperada(12) {
		t.Fatalf("saída = %q", got)
	}
	if j.Pendentes() != 0 {
		t.Fatal("entidade completa continuou pendente")
	}
}

func TestJuncaoEntidadesSaoIndependentes(t *testing.T) {
	j := NovaJuncao(receitaSoma(), 4)
	for _, e := range []struct {
		chave uint64
		tag   quadro.Tag
		v     int
	}{{1, tagA, 1}, {2, tagA, 10}, {2, tagB, 20}} {
		q, err := j.Entregar(e.chave, origem(t, e.tag, e.v))
		if e.chave == 2 && e.tag == tagB {
			if err != nil || string(q.Saida()) != saidaEsperada(30) {
				t.Fatalf("entidade 2 = %v", err)
			}
			quadro.Devolver(q)
		} else if !errors.Is(err, ErrAguardando) {
			t.Fatalf("entrega %+v = %v", e, err)
		}
	}
	if j.Pendentes() != 1 {
		t.Fatalf("pendentes = %d; só a entidade 1 devia esperar", j.Pendentes())
	}
}

func TestJuncaoRecusaTagRepetidaSemPerderOQueChegou(t *testing.T) {
	j := NovaJuncao(receitaSoma(), 4)
	if _, err := j.Entregar(1, origem(t, tagA, 1)); !errors.Is(err, ErrAguardando) {
		t.Fatal(err)
	}
	reenvio := origem(t, tagA, 999)
	if _, err := j.Entregar(1, reenvio); !errors.Is(err, quadro.ErrRepetida) {
		t.Fatalf("reenvio = %v; quero ErrRepetida", err)
	}
	quadro.Devolver(reenvio) // na recusa, a posse continua com quem entregou
	q, err := j.Entregar(1, origem(t, tagB, 2))
	if err != nil || string(q.Saida()) != saidaEsperada(3) {
		t.Fatalf("o valor original foi perdido ou sobrescrito: %v", err)
	}
	quadro.Devolver(q)
}

func TestJuncaoCheiaRecusa(t *testing.T) {
	j := NovaJuncao(receitaSoma(), 1)
	if _, err := j.Entregar(1, origem(t, tagA, 1)); !errors.Is(err, ErrAguardando) {
		t.Fatal(err)
	}
	if _, err := j.Entregar(2, origem(t, tagA, 1)); !errors.Is(err, quadro.ErrCheio) {
		t.Fatalf("tabela cheia = %v; quero ErrCheio", err)
	}
	if _, err := j.Entregar(1, origem(t, tagB, 1)); err != nil {
		t.Fatalf("entidade já pendente precisa continuar entregável: %v", err)
	}
}

func TestJuncaoDisparaUmaVezSobConcorrencia(t *testing.T) {
	const entidades = 200
	j := NovaJuncao(receitaSoma(), entidades)
	var disparos atomic.Int64
	var wg sync.WaitGroup
	for e := range entidades {
		for _, tag := range []quadro.Tag{tagA, tagB} {
			wg.Add(1)
			go func() {
				defer wg.Done()
				q, err := j.Entregar(uint64(e), origem(t, tag, e))
				if err == nil {
					disparos.Add(1)
					if string(q.Saida()) != saidaEsperada(2*e) {
						t.Errorf("entidade %d: saída %q", e, q.Saida())
					}
					quadro.Devolver(q)
				} else if !errors.Is(err, ErrAguardando) {
					t.Errorf("entidade %d: %v", e, err)
				}
			}()
		}
	}
	wg.Wait()
	if disparos.Load() != entidades || j.Pendentes() != 0 {
		t.Fatalf("disparos = %d, pendentes = %d; quero %d e 0", disparos.Load(), j.Pendentes(), entidades)
	}
}
