package quadro

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// Formato de cada campo: [tag u16][tamanho u16][valor], big-endian.
const (
	CabecalhoCampo = 4
	MaxValor       = 0xFFFF
)

// Indexar percorre a entrada uma vez e marca a faixa de cada tag. Nada é
// copiado nem decodificado: só posições.
func (q *Quadro) Indexar() error {
	b := q.buf[:q.entrada]
	for i := 0; i < len(b); {
		if len(b)-i < CabecalhoCampo {
			return ErrCorte
		}
		t := Tag(binary.BigEndian.Uint16(b[i:]))
		n := int(binary.BigEndian.Uint16(b[i+2:]))
		i += CabecalhoCampo
		if n > len(b)-i {
			return ErrCorte
		}
		if t >= MaxTags {
			return fmt.Errorf("%w: %d", ErrTag, t)
		}
		if q.presentes&(1<<t) != 0 {
			return fmt.Errorf("%w: %d", ErrRepetida, t)
		}
		q.presentes |= 1 << t
		q.faixas[t] = Faixa{Ini: uint32(i), Fim: uint32(i + n)}
		i += n
	}
	return nil
}

// Anexar escreve um campo na saída, no mesmo buffer. Sem capacidade,
// recusa em vez de realocar: realocar moveria o dado em silêncio.
func (q *Quadro) Anexar(t Tag, valor []byte) error {
	if len(valor) > MaxValor || cap(q.buf)-len(q.buf) < CabecalhoCampo+len(valor) {
		return ErrCheio
	}
	q.buf = AnexarTLV(q.buf, t, valor)
	return nil
}

// AnexarInt escreve v em decimal direto no buffer, sem string intermediária.
func (q *Quadro) AnexarInt(t Tag, v int64) error {
	const maxDigitos = 20 // "-9223372036854775808"
	if cap(q.buf)-len(q.buf) < CabecalhoCampo+maxDigitos {
		return ErrCheio
	}
	ini := len(q.buf)
	q.buf = binary.BigEndian.AppendUint16(q.buf, uint16(t))
	q.buf = append(q.buf, 0, 0) // tamanho, preenchido abaixo
	q.buf = strconv.AppendInt(q.buf, v, 10)
	binary.BigEndian.PutUint16(q.buf[ini+2:], uint16(len(q.buf)-ini-CabecalhoCampo))
	return nil
}

// AnexarTLV codifica um campo em dst. É o lado de quem monta a mensagem
// (tela, cliente, testes); o quadro usa o mesmo formato.
func AnexarTLV(dst []byte, t Tag, valor []byte) []byte {
	dst = binary.BigEndian.AppendUint16(dst, uint16(t))
	dst = binary.BigEndian.AppendUint16(dst, uint16(len(valor)))
	return append(dst, valor...)
}
