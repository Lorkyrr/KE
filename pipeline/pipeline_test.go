package pipeline

import (
	"errors"
	"testing"

	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

func TestEtapasSoDisparamComTodasAsEntradas(t *testing.T) {
	r := Declarar("teste", []quadro.Tag{1, 2}, operadores.Converter(operadores.R0, 1),
		operadores.Escrever(3, operadores.R0))
	in := quadro.AnexarTLV(nil, 1, []byte("7"))
	q := quadro.Novo()
	area, _ := q.Reservar(len(in))
	copy(area, in)
	_ = q.Indexar()
	if err := r.Executar(q); !errors.Is(err, ErrAguardando) {
		t.Fatalf("erro = %v, quero ErrAguardando", err)
	}
	if len(q.Saida()) != 0 {
		t.Fatal("etapa rodou sem todas as entradas")
	}
}
