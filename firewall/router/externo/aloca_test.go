//go:build !race

// Fora do -race: o detector faz o sync.Pool descartar itens de propósito, e a
// contagem de alocações deixa de ser determinística.

package externo

import (
	"bytes"
	"io"
	"testing"

	"github.com/lorkyrr/ke/quadro"
)

// O transporte também é zero-alocação: cabeçalho e lote de envio não podem ir
// para o heap a cada mensagem (ADR 0001, "o dado não se move").
func TestTransporteNaoAloca(t *testing.T) {
	msg := []byte{0, 0, 0, 4, 0, 1, 0, 0}
	q := quadro.Novo()
	leitor := bytes.NewReader(msg)
	if n := testing.AllocsPerRun(200, func() {
		q.Zerar()
		leitor.Reset(msg)
		if err := Receber(leitor, q); err != nil {
			t.Fatal(err)
		}
	}); n != 0 {
		t.Errorf("Receber: %v alocações por mensagem, quero 0", n)
	}

	if err := q.Anexar(1, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if n := testing.AllocsPerRun(200, func() {
		if err := Enviar(io.Discard, q); err != nil {
			t.Fatal(err)
		}
	}); n != 0 {
		t.Errorf("Enviar: %v alocações por mensagem, quero 0", n)
	}
}
