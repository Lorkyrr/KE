package sanitizar

import "testing"

func TestDigitos(t *testing.T) {
	for v, ok := range map[string]bool{"0": true, "1990": true, "": false, "1a": false, " 1": false, "١": false} {
		if (Digitos([]byte(v)) == nil) != ok {
			t.Errorf("Digitos(%q) aceito = %v", v, !ok)
		}
	}
}
