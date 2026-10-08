// Package sanitizar reúne as folhas puras do firewall (spec v2 §3.2, D4): uma
// função por formato, sobre a janela do campo no buffer ([]byte), sem
// estado, sem converter para string e sem copiar.
package sanitizar

import "errors"

var ErrFormato = errors.New("formato inválido")

// Digitos aceita só 0-9 e pelo menos um dígito.
func Digitos(v []byte) error {
	if len(v) == 0 {
		return ErrFormato
	}
	for _, c := range v {
		if c < '0' || c > '9' {
			return ErrFormato
		}
	}
	return nil
}
