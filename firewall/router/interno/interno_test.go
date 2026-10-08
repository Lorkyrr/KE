package interno

import (
	"errors"
	"testing"

	"github.com/lorkyrr/ke/quadro"
)

func TestEncaminharPassaOEnderecoEFalhaRapido(t *testing.T) {
	r := Nova(1)
	q := quadro.Novo()
	if err := r.Encaminhar(q); err != nil {
		t.Fatal(err)
	}
	if err := r.Encaminhar(quadro.Novo()); !errors.Is(err, ErrRotaCheia) {
		t.Fatalf("erro = %v, quero ErrRotaCheia", err)
	}
	if recebido := <-r.Recebidos(); recebido != q {
		t.Fatal("chegou outro quadro: a rota copiou em vez de passar o endereço")
	}
}

// Com -race: produtor escreve, encaminha e não toca mais; consumidor lê.
func TestPosseVaiComOPonteiro(t *testing.T) {
	r := Nova(4)
	feito := make(chan int64)
	go func() {
		var soma int64
		for q := range r.Recebidos() {
			soma += q.Reg[0]
		}
		feito <- soma
	}()
	for i := range int64(4) {
		q := quadro.Novo()
		q.Reg[0] = i
		for r.Encaminhar(q) != nil {
		}
	}
	r.Fechar()
	if soma := <-feito; soma != 6 {
		t.Fatalf("soma = %d", soma)
	}
}
