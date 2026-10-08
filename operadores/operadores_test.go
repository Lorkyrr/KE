package operadores_test

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"testing"

	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

const (
	tagPreco quadro.Tag = iota + 1
	tagQuantidade
	tagTaxa
	tagTotal
)

func carregar(t testing.TB, campos map[quadro.Tag]string) *quadro.Quadro {
	t.Helper()
	var in []byte
	for tag := range quadro.Tag(quadro.MaxTags) {
		if v, ok := campos[tag]; ok {
			in = quadro.AnexarTLV(in, tag, []byte(v))
		}
	}
	q := quadro.Novo()
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

func TestConverter(t *testing.T) {
	casos := []struct {
		valor string
		quer  int64
		err   error
	}{
		{"0", 0, nil},
		{"1990", 1990, nil},
		{strconv.FormatInt(math.MaxInt64, 10), math.MaxInt64, nil},
		{"9223372036854775808", 0, operadores.ErrEstouro},
		{"", 0, operadores.ErrFormato},
		{"-1", 0, operadores.ErrFormato},
		{"1a", 0, operadores.ErrFormato},
	}
	for _, c := range casos {
		q := carregar(t, map[quadro.Tag]string{tagPreco: c.valor})
		err := operadores.Converter(operadores.R0, tagPreco)(q)
		if !errors.Is(err, c.err) || (err == nil && q.Reg[0] != c.quer) {
			t.Errorf("Converter(%q) = %d, %v; quero %d, %v", c.valor, q.Reg[0], err, c.quer, c.err)
		}
	}
	if err := operadores.Converter(operadores.R0, tagTaxa)(carregar(t, nil)); !errors.Is(err, operadores.ErrAusente) {
		t.Errorf("campo ausente: %v", err)
	}
}

func TestAritmeticaConfereEstouro(t *testing.T) {
	casos := []struct {
		nome string
		op   func(dst, a, b operadores.Reg) operadores.Op
		a, b int64
		quer int64
		err  error
	}{
		{"soma", operadores.Somar, 2, 3, 5, nil},
		{"soma negativa", operadores.Somar, -2, -3, -5, nil},
		{"soma estoura", operadores.Somar, math.MaxInt64, 1, 0, operadores.ErrEstouro},
		{"soma estoura para baixo", operadores.Somar, math.MinInt64, -1, 0, operadores.ErrEstouro},
		{"subtração", operadores.Subtrair, 5, 3, 2, nil},
		{"subtração negativa", operadores.Subtrair, 3, 5, -2, nil},
		{"subtração de negativo", operadores.Subtrair, -1, math.MinInt64, math.MaxInt64, nil},
		{"subtração estoura", operadores.Subtrair, math.MinInt64, 1, 0, operadores.ErrEstouro},
		{"subtração estoura para cima", operadores.Subtrair, math.MaxInt64, -1, 0, operadores.ErrEstouro},
		{"0 - MinInt64", operadores.Subtrair, 0, math.MinInt64, 0, operadores.ErrEstouro},
		{"divisão exata", operadores.Dividir, 6000, 3, 2000, nil},
		{"divisão exata negativa", operadores.Dividir, -6000, 3, -2000, nil},
		{"divisão por zero", operadores.Dividir, 1, 0, 0, operadores.ErrDivisaoPorZero},
		{"MinInt64 / -1", operadores.Dividir, math.MinInt64, -1, 0, operadores.ErrEstouro},
		{"divisão inexata", operadores.Dividir, 10, 3, 0, operadores.ErrInexato},
		{"divisão inexata negativa", operadores.Dividir, -5, 2, 0, operadores.ErrInexato},
		{"produto", operadores.Multiplicar, 1990, 3, 5970, nil},
		{"produto por zero", operadores.Multiplicar, 0, math.MinInt64, 0, nil},
		{"produto negativo", operadores.Multiplicar, -4, 5, -20, nil},
		{"produto estoura", operadores.Multiplicar, math.MaxInt64/2 + 1, 2, 0, operadores.ErrEstouro},
		{"produto estoura negativo", operadores.Multiplicar, math.MinInt64, 2, 0, operadores.ErrEstouro},
		{"MinInt64 * -1", operadores.Multiplicar, math.MinInt64, -1, 0, operadores.ErrEstouro},
		{"-1 * MinInt64", operadores.Multiplicar, -1, math.MinInt64, 0, operadores.ErrEstouro},
		{"limite exato", operadores.Multiplicar, math.MinInt64 / 2, 2, math.MinInt64, nil},
	}
	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			q := quadro.Novo()
			q.Reg[1], q.Reg[2] = c.a, c.b
			err := c.op(operadores.R0, operadores.R1, operadores.R2)(q)
			if !errors.Is(err, c.err) || (err == nil && q.Reg[0] != c.quer) {
				t.Fatalf("= %d, %v; quero %d, %v", q.Reg[0], err, c.quer, c.err)
			}
		})
	}
}

func TestArredondar(t *testing.T) {
	const (
		T = operadores.Truncar
		P = operadores.Piso
		C = operadores.Teto
		A = operadores.MeioAcima
		E = operadores.MeioPar
	)
	casos := []struct {
		a, b int64
		quer map[operadores.Modo]int64
	}{
		{6, 3, map[operadores.Modo]int64{T: 2, P: 2, C: 2, A: 2, E: 2}},
		{10, 3, map[operadores.Modo]int64{T: 3, P: 3, C: 4, A: 3, E: 3}},
		{-10, 3, map[operadores.Modo]int64{T: -3, P: -4, C: -3, A: -3, E: -3}},
		{10, -3, map[operadores.Modo]int64{T: -3, P: -4, C: -3, A: -3, E: -3}},
		{-10, -3, map[operadores.Modo]int64{T: 3, P: 3, C: 4, A: 3, E: 3}},
		{5, 2, map[operadores.Modo]int64{T: 2, P: 2, C: 3, A: 3, E: 2}},
		{-5, 2, map[operadores.Modo]int64{T: -2, P: -3, C: -2, A: -3, E: -2}},
		{7, 2, map[operadores.Modo]int64{T: 3, P: 3, C: 4, A: 4, E: 4}},
		{-7, 2, map[operadores.Modo]int64{T: -3, P: -4, C: -3, A: -4, E: -4}},
		{2, 3, map[operadores.Modo]int64{T: 0, P: 0, C: 1, A: 1, E: 1}},
		{math.MinInt64, 2, map[operadores.Modo]int64{T: math.MinInt64 / 2, P: math.MinInt64 / 2, C: math.MinInt64 / 2, A: math.MinInt64 / 2, E: math.MinInt64 / 2}},
		{math.MinInt64, math.MaxInt64, map[operadores.Modo]int64{T: -1, P: -2, C: -1, A: -1, E: -1}},
	}
	for _, c := range casos {
		for modo, quer := range c.quer {
			q := quadro.Novo()
			q.Reg[1], q.Reg[2] = c.a, c.b
			if err := operadores.Arredondar(operadores.R0, operadores.R1, operadores.R2, modo)(q); err != nil || q.Reg[0] != quer {
				t.Errorf("Arredondar(%d, %d, modo %d) = %d, %v; quero %d", c.a, c.b, modo, q.Reg[0], err, quer)
			}
		}
	}
}

func TestArredondarRecusa(t *testing.T) {
	for _, c := range []struct {
		a, b int64
		err  error
	}{
		{1, 0, operadores.ErrDivisaoPorZero},
		{math.MinInt64, -1, operadores.ErrEstouro},
	} {
		q := quadro.Novo()
		q.Reg[1], q.Reg[2] = c.a, c.b
		if err := operadores.Arredondar(operadores.R0, operadores.R1, operadores.R2, operadores.Truncar)(q); !errors.Is(err, c.err) {
			t.Errorf("Arredondar(%d, %d) = %v; quero %v", c.a, c.b, err, c.err)
		}
	}
}

func TestModoInvalidoQuebraNaDeclaracao(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("modo inexistente não quebrou na declaração")
		}
	}()
	operadores.Arredondar(operadores.R0, operadores.R1, operadores.R2, operadores.Modo(99))
}

func TestSeqParaNoPrimeiroErroEDizAEtapa(t *testing.T) {
	q := carregar(t, map[quadro.Tag]string{tagPreco: "10"})
	err := operadores.Seq(
		operadores.Converter(operadores.R0, tagPreco),
		operadores.Converter(operadores.R1, tagQuantidade),
		operadores.Escrever(tagTotal, operadores.R0),
	)(q)
	if !errors.Is(err, operadores.ErrAusente) || err.Error() != "etapa 1: operadores: campo ausente" {
		t.Fatalf("erro = %v", err)
	}
	if len(q.Saida()) != 0 {
		t.Fatal("etapa depois do erro rodou")
	}
}

func TestRegistradorInvalidoQuebraNaDeclaracao(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("registrador 8 aceito")
		}
	}()
	operadores.Somar(operadores.R0, operadores.R1, operadores.Reg(quadro.MaxRegs))
}

// Um projeto qualquer monta o próprio processo só com operadores da
// biblioteca: total = preço × quantidade + taxa.
func Example() {
	total := operadores.Seq(
		operadores.Converter(operadores.R0, tagPreco),
		operadores.Converter(operadores.R1, tagQuantidade),
		operadores.Converter(operadores.R2, tagTaxa),
		operadores.Multiplicar(operadores.R3, operadores.R0, operadores.R1),
		operadores.Somar(operadores.R3, operadores.R3, operadores.R2),
		operadores.Escrever(tagTotal, operadores.R3),
	)

	var in []byte
	in = quadro.AnexarTLV(in, tagPreco, []byte("1990"))
	in = quadro.AnexarTLV(in, tagQuantidade, []byte("3"))
	in = quadro.AnexarTLV(in, tagTaxa, []byte("500"))

	q := quadro.Pegar()
	defer quadro.Devolver(q)
	area, _ := q.Reservar(len(in))
	copy(area, in) // o transporte lê direto aqui
	_ = q.Indexar()

	if err := total(q); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(q.Reg[operadores.R3])
	// Output: 6470
}
