package operadores_test

import (
	"errors"
	"testing"

	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

const pipelineOrigem, pipelineDestino uint16 = 1, 2

func TestTransmitirSoltaOQuadroEEntrega(t *testing.T) {
	q := carregar(t, map[quadro.Tag]string{tagPreco: "10"})
	var entregue *quadro.Quadro
	var soltoNaEntrega bool
	err := operadores.Seq(
		operadores.Converter(operadores.R0, tagPreco),
		operadores.Escrever(tagTotal, operadores.R0),
		operadores.Transmitir(pipelineOrigem, func(o *quadro.Quadro) error {
			entregue, soltoNaEntrega = o, o.Solto()
			return nil
		}),
	)(q)
	if !errors.Is(err, operadores.ErrTransmitido) {
		t.Fatalf("erro = %v, quero ErrTransmitido (aviso, não falha)", err)
	}
	if entregue != q || !soltoNaEntrega {
		t.Fatal("a entrega recebeu outro quadro, ou um quadro ainda acoplado")
	}
	if !q.Solto() {
		t.Fatal("quadro ficou acoplado depois de transmitido")
	}
}

func TestPassoDepoisDeTransmitirFalhaComErrSolto(t *testing.T) {
	q := carregar(t, map[quadro.Tag]string{tagPreco: "10"})
	err := operadores.Seq(
		operadores.Transmitir(pipelineOrigem, func(*quadro.Quadro) error { return nil }),
		operadores.Converter(operadores.R0, tagPreco),
	)(q)
	if !errors.Is(err, quadro.ErrSolto) {
		t.Fatalf("erro = %v, quero ErrSolto", err)
	}
	if q.Reg[operadores.R0] != 0 {
		t.Fatal("passo depois de Transmitir rodou")
	}
}

// Recusa por falta de espaço não é devolução: o quadro nunca saiu.
func TestTransmitirRecusadoDevolveOQuadroAOrigem(t *testing.T) {
	q := carregar(t, map[quadro.Tag]string{tagPreco: "10"})
	cheia := errors.New("destino cheio")
	err := operadores.Seq(
		operadores.Transmitir(pipelineOrigem, func(*quadro.Quadro) error { return cheia }),
	)(q)
	if !errors.Is(err, cheia) {
		t.Fatalf("erro = %v, quero o da entrega", err)
	}
	if q.Solto() {
		t.Fatal("recusa deixou o quadro solto")
	}
	if err := q.Anexar(tagTotal, []byte("1")); err != nil {
		t.Fatalf("origem não escreve no quadro que continua dela: %v", err)
	}
}

func TestTransmitirSemEntregaQuebraNaDeclaracao(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("entrega nula aceita")
		}
	}()
	operadores.Transmitir(pipelineOrigem, nil)
}

func TestTransmitirNaoAloca(t *testing.T) {
	q := quadro.Novo()
	prog := operadores.Seq(operadores.Transmitir(pipelineOrigem, func(*quadro.Quadro) error { return nil }))
	allocs := testing.AllocsPerRun(200, func() {
		q.Zerar()
		if err := prog(q); !errors.Is(err, operadores.ErrTransmitido) {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v alocações, quero 0", allocs)
	}
}
