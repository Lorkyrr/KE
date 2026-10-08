package externo

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/lorkyrr/ke/quadro"
)

func mensagem(corpo []byte) []byte {
	return append(binary.BigEndian.AppendUint32(nil, uint32(len(corpo))), corpo...)
}

func TestReceberLeDiretoNoQuadro(t *testing.T) {
	corpo := quadro.AnexarTLV(nil, 1, []byte("1990"))
	q := quadro.Novo()
	if err := Receber(bytes.NewReader(mensagem(corpo)), q); err != nil {
		t.Fatal(err)
	}
	if err := q.Indexar(); err != nil {
		t.Fatal(err)
	}
	if v, _ := q.Campo(1); string(v) != "1990" {
		t.Fatalf("campo = %q", v)
	}
}

func TestReceberRecusaGrandeAntesDeLerOCorpo(t *testing.T) {
	cab := binary.BigEndian.AppendUint32(nil, MaxEntrada+1)
	if err := Receber(bytes.NewReader(cab), quadro.Novo()); !errors.Is(err, ErrGrande) {
		t.Fatalf("erro = %v, quero ErrGrande", err)
	}
}

func TestEnviarMandaSoASaida(t *testing.T) {
	q := quadro.Novo()
	area, _ := q.Reservar(4)
	copy(area, quadro.AnexarTLV(nil, 1, nil))
	if err := q.AnexarInt(2, 5970); err != nil {
		t.Fatal(err)
	}
	var fio bytes.Buffer
	if err := Enviar(&fio, q); err != nil {
		t.Fatal(err)
	}
	if want := mensagem(quadro.AnexarTLV(nil, 2, []byte("5970"))); !bytes.Equal(fio.Bytes(), want) {
		t.Fatalf("fio = %v, quero %v", fio.Bytes(), want)
	}
}
