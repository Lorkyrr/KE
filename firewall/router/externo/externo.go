// Package externo é o transporte do KE e o único pacote que importa net
// (spec v2 §6, D5). Lê do fio direto para dentro do quadro e envia a saída
// do quadro sem montar um buffer novo.
//
// No fio, cada mensagem é [tamanho u32][campos TLV].
package externo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/lorkyrr/ke/quadro"
)

// MaxEntrada deixa metade do quadro para a saída: a resposta sempre cabe
// no mesmo buffer da pergunta.
const MaxEntrada = quadro.Capacidade / 2

var ErrGrande = errors.New("mensagem maior que o limite")

// Receber lê uma mensagem para dentro de q. O tamanho é conferido antes de
// ler o corpo (spec v2 §3.2: o limite vale já no transporte), e o corpo vai
// direto do fio para o buffer do quadro.
func Receber(r io.Reader, q *quadro.Quadro) error {
	var cab [4]byte
	if _, err := io.ReadFull(r, cab[:]); err != nil {
		return fmt.Errorf("ler cabeçalho: %w", err)
	}
	n := binary.BigEndian.Uint32(cab[:])
	if n > MaxEntrada {
		return ErrGrande
	}
	area, err := q.Reservar(int(n))
	if err != nil {
		return err
	}
	if _, err := io.ReadFull(r, area); err != nil {
		return fmt.Errorf("ler corpo: %w", err)
	}
	return nil
}

// Enviar manda a saída do quadro. net.Buffers vira writev em conexões de
// rede: cabeçalho e corpo saem juntos sem serem concatenados.
func Enviar(w io.Writer, q *quadro.Quadro) error {
	saida := q.Saida()
	var cab [4]byte
	binary.BigEndian.PutUint32(cab[:], uint32(len(saida)))
	bufs := net.Buffers{cab[:], saida}
	if _, err := bufs.WriteTo(w); err != nil {
		return fmt.Errorf("enviar: %w", err)
	}
	return nil
}
