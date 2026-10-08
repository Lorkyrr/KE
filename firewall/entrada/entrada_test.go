package entrada

import (
	"errors"
	"strings"
	"testing"

	"github.com/lorkyrr/ke/firewall/sanitizar"
	"github.com/lorkyrr/ke/quadro"
)

var contrato = func() *Contrato {
	var c Contrato
	c[1] = Campo{Max: 12, Obrigatorio: true, Sanitizar: sanitizar.Digitos}
	c[2] = Campo{Max: 4, Sanitizar: sanitizar.Digitos}
	return &c
}()

func carregar(t testing.TB, in []byte) *quadro.Quadro {
	t.Helper()
	q := quadro.Novo()
	area, err := q.Reservar(len(in))
	if err != nil {
		t.Fatal(err)
	}
	copy(area, in)
	return q
}

func TestVerificar(t *testing.T) {
	cpf := []byte("12345678909")
	casos := map[string]struct {
		in     []byte
		motivo string
	}{
		"aceita":        {quadro.AnexarTLV(nil, 1, []byte("1990")), ""},
		"opcional ok":   {quadro.AnexarTLV(quadro.AnexarTLV(nil, 1, []byte("1")), 2, []byte("3")), ""},
		"obrigatório":   {quadro.AnexarTLV(nil, 2, []byte("3")), "obrigatório ausente"},
		"não declarado": {quadro.AnexarTLV(quadro.AnexarTLV(nil, 1, []byte("1")), 9, cpf), "não declarado"},
		"grande demais": {quadro.AnexarTLV(quadro.AnexarTLV(nil, 1, []byte("1")), 2, []byte("12345")), "maior que 4 bytes"},
		"formato":       {quadro.AnexarTLV(nil, 1, []byte("1a")), "formato"},
		"malformada":    {[]byte{0, 1, 0, 9}, "mensagem malformada"},
	}
	for nome, c := range casos {
		t.Run(nome, func(t *testing.T) {
			err := Verificar(carregar(t, c.in), contrato)
			if c.motivo == "" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			var r *Recusa
			if !errors.As(err, &r) || r.Motivo != c.motivo {
				t.Fatalf("erro = %v, quero motivo %q", err, c.motivo)
			}
			if strings.Contains(err.Error(), string(cpf)) {
				t.Fatal("recusa carrega o valor do campo")
			}
		})
	}
}

func TestVerificarNaoAloca(t *testing.T) {
	in := quadro.AnexarTLV(quadro.AnexarTLV(nil, 1, []byte("1990")), 2, []byte("3"))
	q := quadro.Novo()
	allocs := testing.AllocsPerRun(100, func() {
		q.Zerar()
		area, _ := q.Reservar(len(in))
		copy(area, in)
		if err := Verificar(q, contrato); err != nil {
			t.Fatal(err)
		}
	})
	if allocs != 0 {
		t.Fatalf("%v alocações, quero 0", allocs)
	}
}
