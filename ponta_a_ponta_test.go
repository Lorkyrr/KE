package ke_test

import (
	"encoding/binary"
	"errors"
	"net"
	"testing"

	"github.com/lorkyrr/ke/firewall/entrada"
	"github.com/lorkyrr/ke/firewall/router/externo"
	"github.com/lorkyrr/ke/firewall/router/interno"
	"github.com/lorkyrr/ke/firewall/sanitizar"
	"github.com/lorkyrr/ke/nucleo/pipeline"
	"github.com/lorkyrr/ke/operadores"
	"github.com/lorkyrr/ke/quadro"
)

// Uma receita de exemplo, montada como um projeto consumidor montaria: só
// com operadores da biblioteca.
const (
	tagPreco quadro.Tag = iota + 1
	tagQuantidade
	tagTotal
)

var (
	contrato = func() *entrada.Contrato {
		var c entrada.Contrato
		c[tagPreco] = entrada.Campo{Max: 12, Obrigatorio: true, Sanitizar: sanitizar.Digitos}
		c[tagQuantidade] = entrada.Campo{Max: 3, Obrigatorio: true, Sanitizar: sanitizar.Digitos}
		return &c
	}()
	totalItem = pipeline.Declarar("total-item", []quadro.Tag{tagPreco, tagQuantidade},
		operadores.Converter(operadores.R0, tagPreco),
		operadores.Converter(operadores.R1, tagQuantidade),
		operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1),
		operadores.Escrever(tagTotal, operadores.R2),
	)
)

func mensagem(campos ...[2]string) []byte {
	var corpo []byte
	for _, c := range campos {
		corpo = quadro.AnexarTLV(corpo, quadro.Tag(c[0][0]-'0'), []byte(c[1]))
	}
	return append(binary.BigEndian.AppendUint32(nil, uint32(len(corpo))), corpo...)
}

// O caminho inteiro: fio → quadro → contrato → rota → receita → fio. O dado
// entra no buffer uma vez e a resposta sai do mesmo buffer.
func TestPontaAPonta(t *testing.T) {
	cliente, servico := net.Pipe()
	defer cliente.Close()
	rota := interno.Nova(8)
	pronto := make(chan error, 1)

	go func() { // trabalhador: dono do quadro depois do Encaminhar
		for q := range rota.Recebidos() {
			err := totalItem.Executar(q)
			if err == nil {
				err = externo.Enviar(servico, q)
			}
			quadro.Devolver(q)
			pronto <- err
		}
	}()
	go func() { // transporte + firewall
		q := quadro.Pegar()
		if err := externo.Receber(servico, q); err != nil {
			pronto <- err
			return
		}
		if err := entrada.Verificar(q, contrato); err != nil {
			pronto <- err
			return
		}
		if err := rota.Encaminhar(q); err != nil {
			pronto <- err
		}
	}()

	go func() { _, _ = cliente.Write(mensagem([2]string{"1", "1990"}, [2]string{"2", "3"})) }()
	resp := quadro.Novo()
	if err := externo.Receber(cliente, resp); err != nil {
		t.Fatal(err)
	}
	if err := <-pronto; err != nil {
		t.Fatal(err)
	}
	rota.Fechar()
	if err := resp.Indexar(); err != nil {
		t.Fatal(err)
	}
	if v, _ := resp.Campo(tagTotal); string(v) != "5970" {
		t.Fatalf("total = %q, quero 5970", v)
	}
}

func TestPontaAPontaRecusas(t *testing.T) {
	carregar := func(m []byte) *quadro.Quadro {
		q := quadro.Novo()
		area, _ := q.Reservar(len(m) - 4)
		copy(area, m[4:])
		return q
	}
	var r *entrada.Recusa
	err := entrada.Verificar(carregar(mensagem([2]string{"1", "1"}, [2]string{"2", "1"}, [2]string{"9", "x"})), contrato)
	if !errors.As(err, &r) || r.Motivo != "não declarado" {
		t.Fatalf("tag não declarada: %v", err)
	}
	q := carregar(mensagem([2]string{"1", "999999999999"}, [2]string{"2", "999"}))
	if err := entrada.Verificar(q, contrato); err != nil {
		t.Fatal(err)
	}
	// Os limites do contrato (12 e 3 dígitos) já garantem que o produto
	// cabe em int64; o operador confere de novo porque é biblioteca e não
	// sabe que contrato veio antes.
	if err := totalItem.Executar(q); err != nil {
		t.Fatal(err)
	}
}

// Prova do zero-cópia: depois da leitura, o caminho contrato → receita →
// saída não aloca nem copia.
func TestCaminhoNaoAloca(t *testing.T) {
	m := mensagem([2]string{"1", "1990"}, [2]string{"2", "3"})[4:]
	q := quadro.Novo()
	allocs := testing.AllocsPerRun(200, func() {
		q.Zerar()
		area, _ := q.Reservar(len(m))
		copy(area, m) // a leitura do transporte
		if err := entrada.Verificar(q, contrato); err != nil {
			t.Fatal(err)
		}
		if err := totalItem.Executar(q); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v alocações por comando, quero 0", allocs)
	}
}
