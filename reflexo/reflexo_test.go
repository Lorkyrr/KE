package reflexo

import (
	"slices"
	"strings"
	"testing"
)

const cabecalho = "package x\n"

func uma(t *testing.T, corpo string) Receita {
	t.Helper()
	rs, err := LerFonte("x.go", []byte(cabecalho+corpo))
	if err != nil || len(rs) != 1 {
		t.Fatalf("receitas = %d, erro = %v; quero 1 sem erro", len(rs), err)
	}
	return rs[0]
}

func TestReceitaSoDeOperadoresNaoTemProblema(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("soma", []quadro.Tag{tA, tB},
		operadores.Converter(operadores.R0, tA),
		operadores.Converter(operadores.R1, tB),
		operadores.Somar(operadores.R2, operadores.R0, operadores.R1),
		operadores.Escrever(tSoma, operadores.R2),
	)`)
	if ps := r.Problemas(); len(ps) != 0 {
		t.Fatalf("problemas = %v, quero nenhum", ps)
	}
	if !slices.Equal(r.Lidas, []string{"tA", "tB"}) {
		t.Fatalf("Lidas = %v, quero [tA tB]", r.Lidas)
	}
	if r.Nome != "soma" {
		t.Fatalf("Nome = %q", r.Nome)
	}
}

func TestEtapaQueNaoEOperadorEProblema(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("grande", []quadro.Tag{tA},
		operadores.Converter(operadores.R0, tA),
		func(q *quadro.Quadro) error { return nil },
		operadores.Escrever(tSoma, operadores.R0),
	)`)
	ps := r.Problemas()
	if len(ps) != 1 || !strings.Contains(ps[0], "etapa 1") {
		t.Fatalf("problemas = %v, quero só a etapa 1", ps)
	}
}

func TestEntradaLidaESemDeclararEProblema(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("r", []quadro.Tag{tA},
		operadores.Converter(operadores.R0, tA),
		operadores.Converter(operadores.R1, tB),
		operadores.Escrever(tSoma, operadores.R1),
	)`)
	ps := r.Problemas()
	if len(ps) != 1 || !strings.Contains(ps[0], "tB") {
		t.Fatalf("problemas = %v, quero só tB", ps)
	}
}

func TestTransmitirForaDoFimEProblema(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("r", []quadro.Tag{tA},
		operadores.Converter(operadores.R0, tA),
		operadores.Transmitir(1, rota.Encaminhar),
		operadores.Escrever(tSoma, operadores.R0),
	)`)
	ps := r.Problemas()
	if len(ps) != 1 || !strings.Contains(ps[0], "Transmitir") {
		t.Fatalf("problemas = %v, quero só Transmitir", ps)
	}
}

func TestReceitaSemSaidaEProblema(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("r", []quadro.Tag{tA},
		operadores.Converter(operadores.R0, tA),
	)`)
	if ps := r.Problemas(); len(ps) != 1 || !strings.Contains(ps[0], "saída") {
		t.Fatalf("problemas = %v, quero só a falta de saída", ps)
	}
}

func TestReflexoDizComoDeveriaSerDescrita(t *testing.T) {
	r := uma(t, `var r = pipeline.Declarar("r", []quadro.Tag{tA},
		operadores.Converter(operadores.R0, tA),
		operadores.Converter(operadores.R1, tB),
		func(q *quadro.Quadro) error { return nil },
	)`)
	got := r.Reflexo()
	for _, quero := range []string{"2/3", "{tA, tB}", "✗"} {
		if !strings.Contains(got, quero) {
			t.Errorf("Reflexo() sem %q:\n%s", quero, got)
		}
	}
}
