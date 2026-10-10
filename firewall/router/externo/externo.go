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
	"sync"

	"github.com/lorkyrr/ke/quadro"
)

// MaxEntrada deixa metade do quadro para a saída: a resposta sempre cabe
// no mesmo buffer da pergunta.
const MaxEntrada = quadro.Capacidade / 2

var ErrGrande = errors.New("mensagem maior que o limite")

// rascunho é o que o transporte usa por mensagem. Passa por io.Reader e
// io.Writer, então o compilador o mandaria ao heap a cada chamada; o pool
// paga isso uma vez só.
type rascunho struct {
	cab  [4]byte
	lote [2][]byte
	bufs net.Buffers // WriteTo consome o slice; mora aqui para não ir ao heap
}

var rascunhos = sync.Pool{New: func() any { return new(rascunho) }}

// Receber lê uma mensagem para dentro de q. O tamanho é conferido antes de
// ler o corpo (spec v2 §3.2: o limite vale já no transporte), e o corpo vai
// direto do fio para o buffer do quadro.
func Receber(r io.Reader, q *quadro.Quadro) error {
	rs := rascunhos.Get().(*rascunho)
	_, err := io.ReadFull(r, rs.cab[:])
	n := binary.BigEndian.Uint32(rs.cab[:])
	rascunhos.Put(rs)
	if err != nil {
		return fmt.Errorf("ler cabeçalho: %w", err)
	}
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
	rs := rascunhos.Get().(*rascunho)
	binary.BigEndian.PutUint32(rs.cab[:], uint32(len(saida)))
	rs.lote = [2][]byte{rs.cab[:], saida}
	rs.bufs = rs.lote[:]
	_, err := rs.bufs.WriteTo(w)
	rs.lote, rs.bufs = [2][]byte{}, nil // o pool não segura o buffer do quadro
	rascunhos.Put(rs)
	if err != nil {
		return fmt.Errorf("enviar: %w", err)
	}
	return nil
}
