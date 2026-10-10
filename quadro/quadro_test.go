package quadro

import (
	"bytes"
	"errors"
	"testing"
)

func carregar(t testing.TB, q *Quadro, entrada []byte) {
	t.Helper()
	area, err := q.Reservar(len(entrada))
	if err != nil {
		t.Fatal(err)
	}
	copy(area, entrada) // faz o papel da leitura do transporte
}

func TestIndexarMarcaFaixasSemCopiar(t *testing.T) {
	q := Novo()
	in := AnexarTLV(nil, 3, []byte("1990"))
	in = AnexarTLV(in, 7, []byte("2"))
	carregar(t, q, in)
	if err := q.Indexar(); err != nil {
		t.Fatal(err)
	}
	v, ok := q.Campo(3)
	if !ok || string(v) != "1990" {
		t.Fatalf("campo 3 = %q, %v", v, ok)
	}
	if &v[0] != &q.buf[CabecalhoCampo] {
		t.Fatal("Campo copiou o dado em vez de devolver uma janela")
	}
	if cap(v) != len(v) {
		t.Fatal("janela permite append sobre o campo vizinho")
	}
	if q.Presente(5) {
		t.Fatal("tag ausente marcada como presente")
	}
}

func TestIndexarRecusa(t *testing.T) {
	casos := map[string]struct {
		in  []byte
		err error
	}{
		"corte no cabecalho": {[]byte{0, 1, 0}, ErrCorte},
		"corte no valor":     {[]byte{0, 1, 0, 5, 'a'}, ErrCorte},
		"tag fora":           {AnexarTLV(nil, MaxTags, nil), ErrTag},
		"repetida":           {AnexarTLV(AnexarTLV(nil, 1, nil), 1, nil), ErrRepetida},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			q := Novo()
			carregar(t, q, c.in)
			if err := q.Indexar(); !errors.Is(err, c.err) {
				t.Fatalf("erro = %v, quero %v", err, c.err)
			}
		})
	}
}

func TestAnexarNoMesmoBufferSemRealocar(t *testing.T) {
	q := Novo()
	carregar(t, q, AnexarTLV(nil, 1, []byte("x")))
	base := &q.buf[:1][0]
	if err := q.AnexarInt(2, -42); err != nil {
		t.Fatal(err)
	}
	if &q.buf[:1][0] != base {
		t.Fatal("buffer realocado")
	}
	if want := AnexarTLV(nil, 2, []byte("-42")); !bytes.Equal(q.Saida(), want) {
		t.Fatalf("saida = %v, quero %v", q.Saida(), want)
	}
}

func TestAnexarSemCapacidadeRecusa(t *testing.T) {
	q := Novo()
	carregar(t, q, make([]byte, Capacidade-CabecalhoCampo))
	if err := q.Anexar(1, []byte("a")); !errors.Is(err, ErrCheio) {
		t.Fatalf("erro = %v, quero ErrCheio", err)
	}
	if err := q.AnexarInt(1, 1); !errors.Is(err, ErrCheio) {
		t.Fatalf("erro = %v, quero ErrCheio", err)
	}
}

func TestDevolverApagaODado(t *testing.T) {
	q := Novo()
	carregar(t, q, AnexarTLV(nil, 1, []byte("12345678909")))
	_ = q.Indexar()
	usado := q.buf[:len(q.buf)]
	q.Zerar()
	if !bytes.Equal(usado, make([]byte, len(usado))) {
		t.Fatal("bytes usados não foram sobrescritos")
	}
	if q.Presente(1) || len(q.buf) != 0 {
		t.Fatal("índice não foi limpo")
	}
}

func TestCaminhoNaoAloca(t *testing.T) {
	q := Novo()
	in := AnexarTLV(AnexarTLV(nil, 3, []byte("1990")), 7, []byte("2"))
	allocs := testing.AllocsPerRun(100, func() {
		q.Zerar()
		carregar(t, q, in)
		if err := q.Indexar(); err != nil {
			t.Fatal(err)
		}
		v, _ := q.Campo(3)
		if err := q.Anexar(9, v); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v alocações por comando, quero 0", allocs)
	}
}

func FuzzIndexar(f *testing.F) {
	f.Add(AnexarTLV(AnexarTLV(nil, 3, []byte("1990")), 7, []byte("2")))
	f.Add([]byte{0, 1, 0xFF, 0xFF})
	f.Fuzz(func(t *testing.T, in []byte) {
		if len(in) > Capacidade {
			return
		}
		q := Novo()
		carregar(t, q, in)
		if q.Indexar() != nil {
			return
		}
		for tag := range Tag(MaxTags) {
			if f, ok := q.Faixa(tag); ok && (f.Ini > f.Fim || int(f.Fim) > len(in)) {
				t.Fatalf("faixa %v fora da entrada de %d bytes", f, len(in))
			}
		}
	})
}

// Mede o que a Faixa evita: entregar o campo a cada etapa como cópia.
var (
	sumidouro  int
	sumidouroS string
)

func BenchmarkCampoPorFaixa(b *testing.B) {
	q := Novo()
	carregar(b, q, AnexarTLV(nil, 1, bytes.Repeat([]byte("a"), 512)))
	_ = q.Indexar()
	b.ReportAllocs()
	for b.Loop() {
		v, _ := q.Campo(1)
		sumidouro += len(v)
	}
}

func BenchmarkCampoPorCopia(b *testing.B) {
	q := Novo()
	carregar(b, q, AnexarTLV(nil, 1, bytes.Repeat([]byte("a"), 512)))
	_ = q.Indexar()
	b.ReportAllocs()
	for b.Loop() {
		v, _ := q.Campo(1)
		sumidouroS = string(v)
	}
}

func comCampo(t testing.TB, tag Tag, valor string) *Quadro {
	t.Helper()
	q := Novo()
	carregar(t, q, AnexarTLV(nil, tag, []byte(valor)))
	if err := q.Indexar(); err != nil {
		t.Fatal(err)
	}
	return q
}

func TestLigarEnxergaCamposDasOrigensSemCopiar(t *testing.T) {
	a, b := comCampo(t, 3, "1990"), comCampo(t, 7, "2")
	q := Novo()
	if err := q.Ligar(a); err != nil {
		t.Fatal(err)
	}
	if err := q.Ligar(b); err != nil {
		t.Fatal(err)
	}
	v, ok := q.Campo(3)
	if !ok || string(v) != "1990" || &v[0] != &a.buf[CabecalhoCampo] {
		t.Fatalf("campo 3 = %q, %v (copiou ou não achou)", v, ok)
	}
	if v, ok := q.Campo(7); !ok || string(v) != "2" {
		t.Fatalf("campo 7 = %q, %v", v, ok)
	}
	if q.Presente(5) {
		t.Fatal("tag ausente marcada como presente")
	}
	if _, ok := q.Faixa(3); ok {
		t.Fatal("Faixa é posição no buffer próprio; campo de origem não tem faixa aqui")
	}
	if n := testing.AllocsPerRun(100, func() { q.Campo(7) }); n != 0 {
		t.Fatalf("Campo via origem aloca %v", n)
	}
}

func TestLigarRecusa(t *testing.T) {
	q := Novo()
	if err := q.Ligar(q); !errors.Is(err, ErrOrigem) {
		t.Errorf("ligar a si mesmo = %v", err)
	}
	if err := q.Ligar(nil); !errors.Is(err, ErrOrigem) {
		t.Errorf("ligar nil = %v", err)
	}
	if err := q.Ligar(comCampo(t, 3, "1")); err != nil {
		t.Fatal(err)
	}
	if err := q.Ligar(comCampo(t, 3, "2")); !errors.Is(err, ErrRepetida) {
		t.Errorf("tag repetida entre origens = %v", err)
	}
	for i := 1; i < MaxOrigens; i++ {
		if err := q.Ligar(comCampo(t, Tag(10+i), "x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := q.Ligar(comCampo(t, 40, "x")); !errors.Is(err, ErrCheio) {
		t.Errorf("origem além do limite = %v", err)
	}
}

func TestIndexarRecusaTagJaVisivelPorOrigem(t *testing.T) {
	q := Novo()
	if err := q.Ligar(comCampo(t, 3, "1")); err != nil {
		t.Fatal(err)
	}
	carregar(t, q, AnexarTLV(nil, 3, []byte("2")))
	if err := q.Indexar(); !errors.Is(err, ErrRepetida) {
		t.Fatalf("Indexar = %v; quero ErrRepetida", err)
	}
}

func TestDevolverTambemDevolveAsOrigens(t *testing.T) {
	a := comCampo(t, 3, "dado pessoal")
	q := Novo()
	if err := q.Ligar(a); err != nil {
		t.Fatal(err)
	}
	Devolver(q)
	if a.Presente(3) || len(a.buf) != 0 || q.Presente(3) {
		t.Fatal("origem não foi zerada junto com o quadro agregado")
	}
}

func TestPromoverTornaASaidaEntradaSemCopiar(t *testing.T) {
	q := comCampo(t, 1, "7")
	if err := q.Anexar(2, []byte("abc")); err != nil {
		t.Fatal(err)
	}
	if err := q.AnexarInt(3, 12); err != nil {
		t.Fatal(err)
	}
	saida := q.Saida()
	if err := q.Promover(); err != nil {
		t.Fatal(err)
	}
	v, ok := q.Campo(2)
	if !ok || string(v) != "abc" || &v[0] != &saida[CabecalhoCampo] {
		t.Fatalf("campo 2 = %q, %v (copiou ou não achou)", v, ok)
	}
	if v, ok := q.Campo(3); !ok || string(v) != "12" {
		t.Fatalf("campo 3 = %q, %v", v, ok)
	}
	if !q.Presente(1) || len(q.Saida()) != 0 {
		t.Fatal("a entrada original sumiu ou a saída não esvaziou")
	}
	if err := q.Anexar(4, []byte("novo")); err != nil || string(q.Saida()) != string(AnexarTLV(nil, 4, []byte("novo"))) {
		t.Fatalf("a nova saída deve começar depois do promovido: %q, %v", q.Saida(), err)
	}
}

func TestPromoverRecusaSemMudarNada(t *testing.T) {
	q := comCampo(t, 1, "7")
	_ = q.Anexar(2, []byte("a"))
	_ = q.Anexar(1, []byte("repetida"))
	saida := string(q.Saida())
	if err := q.Promover(); !errors.Is(err, ErrRepetida) {
		t.Fatalf("Promover = %v; quero ErrRepetida", err)
	}
	if q.Presente(2) || string(q.Saida()) != saida {
		t.Fatal("recusa deixou o quadro pela metade")
	}
}

func TestPromoverSemSaidaNaoFazNada(t *testing.T) {
	q := comCampo(t, 1, "7")
	if err := q.Promover(); err != nil || !q.Presente(1) || len(q.Saida()) != 0 {
		t.Fatalf("Promover vazio = %v", err)
	}
}

func TestPromoverNaoAloca(t *testing.T) {
	q := Novo()
	in := AnexarTLV(nil, 1, []byte("7"))
	n := testing.AllocsPerRun(100, func() {
		q.Zerar()
		area, _ := q.Reservar(len(in))
		copy(area, in)
		_ = q.Indexar()
		_ = q.AnexarInt(2, 12)
		_ = q.Promover()
	})
	if n != 0 {
		t.Fatalf("alocações = %v", n)
	}
}

func TestSoltarEAcoplarRegistramORastro(t *testing.T) {
	q := Novo()
	if q.Solto() {
		t.Fatal("quadro novo nasce acoplado")
	}
	if err := q.Soltar(1); err != nil || !q.Solto() {
		t.Fatalf("Soltar = %v, solto = %v", err, q.Solto())
	}
	if err := q.Acoplar(2); err != nil || q.Solto() {
		t.Fatalf("Acoplar = %v, solto = %v", err, q.Solto())
	}
	quer := []Marca{{Pipeline: 1}, {Pipeline: 2, Acoplou: true}}
	if got := q.Marcas(nil); len(got) != 2 || got[0] != quer[0] || got[1] != quer[1] {
		t.Fatalf("rastro = %v; quero %v", got, quer)
	}
}

func TestPosseInvalidaRecusa(t *testing.T) {
	q := Novo()
	if err := q.Acoplar(1); !errors.Is(err, ErrAcoplado) {
		t.Errorf("acoplar quadro que já está acoplado = %v", err)
	}
	_ = q.Soltar(1)
	if err := q.Soltar(1); !errors.Is(err, ErrSolto) {
		t.Errorf("soltar duas vezes = %v", err)
	}
	if err := q.Anexar(1, []byte("x")); !errors.Is(err, ErrSolto) {
		t.Errorf("Anexar em quadro solto = %v", err)
	}
	if err := q.AnexarInt(1, 7); !errors.Is(err, ErrSolto) {
		t.Errorf("AnexarInt em quadro solto = %v", err)
	}
	if err := q.Promover(); !errors.Is(err, ErrSolto) {
		t.Errorf("Promover em quadro solto = %v", err)
	}
	if len(q.Marcas(nil)) != 1 {
		t.Error("recusa não pode gravar marca")
	}
}

func TestRastroGuardaOsUltimosSaltosEmOrdem(t *testing.T) {
	q := Novo()
	for p := range uint16(MaxMarcas) {
		_ = q.Soltar(p)
		_ = q.Acoplar(p + 1)
	}
	got := q.Marcas(nil)
	if len(got) != MaxMarcas {
		t.Fatalf("len = %d", len(got))
	}
	if ultima := got[MaxMarcas-1]; ultima != (Marca{Pipeline: MaxMarcas, Acoplou: true}) {
		t.Fatalf("última marca = %v", ultima)
	}
	for i := 1; i < len(got); i++ {
		if got[i].Pipeline < got[i-1].Pipeline {
			t.Fatalf("fora de ordem: %v", got)
		}
	}
}

func TestZerarVoltaAcopladoESemRastro(t *testing.T) {
	q := Novo()
	_ = q.Soltar(1)
	q.Zerar()
	if q.Solto() || len(q.Marcas(nil)) != 0 {
		t.Fatal("quadro reaproveitado herdou posse ou rastro")
	}
}

func TestTransmissaoNaoAloca(t *testing.T) {
	q := Novo()
	dst := make([]Marca, 0, MaxMarcas)
	n := testing.AllocsPerRun(100, func() {
		_ = q.Soltar(1)
		_ = q.Acoplar(2)
		dst = q.Marcas(dst[:0])
	})
	if n != 0 {
		t.Fatalf("alocações = %v", n)
	}
}

func BenchmarkCampoDeOrigem(b *testing.B) {
	o := comCampo(b, 1, "abc")
	q := Novo()
	if err := q.Ligar(o); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for b.Loop() {
		v, _ := q.Campo(1)
		sumidouro += len(v)
	}
}
