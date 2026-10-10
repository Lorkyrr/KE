// Testes de fronteira do KE (spec v2 §7, §10): cada regra de import vira um
// teste, e a regra ISA (só a biblioteca de operadores define instruções)
// é conferida na árvore de sintaxe.
package ke_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lorkyrr/ke/reflexo"
)

const modulo = "github.com/lorkyrr/ke"

// deps devolve, por pacote do KE, tudo o que ele importa (transitivo, sem
// os testes).
func deps(t *testing.T) map[string][]string {
	t.Helper()
	out, err := exec.Command("go", "list", "-f", `{{.ImportPath}} {{join .Deps " "}}`, "./...").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	m := map[string][]string{}
	for linha := range strings.Lines(string(out)) {
		campos := strings.Fields(linha)
		if len(campos) > 0 {
			m[strings.TrimPrefix(campos[0], modulo)] = campos[1:]
		}
	}
	return m
}

func stdlib(p string) bool { return !strings.Contains(strings.SplitN(p, "/", 2)[0], ".") }

func TestSoORouterExternoImportaRede(t *testing.T) {
	for pkg, ds := range deps(t) {
		if pkg == "/firewall/router/externo" {
			continue
		}
		for _, d := range ds {
			if d == "net" || d == "net/http" || strings.HasPrefix(d, "database/sql") {
				t.Errorf("%s importa %s; só firewall/router/externo pode", pkg, d)
			}
		}
	}
}

// quadro e operadores são biblioteca: qualquer projeto importa sem carregar
// nada do KE além deles e da stdlib (ADR 0001).
func TestBibliotecaSoDependeDaStdlib(t *testing.T) {
	permitidos := map[string]bool{modulo + "/quadro": true, modulo + "/operadores": true}
	for pkg, ds := range deps(t) {
		if pkg != "/quadro" && pkg != "/operadores" && pkg != "/pipeline" {
			continue
		}
		for _, d := range ds {
			if !stdlib(d) && !permitidos[d] {
				t.Errorf("%s importa %s; a biblioteca só pode importar stdlib e a si mesma", pkg, d)
			}
		}
	}
}

func TestDirecaoDasCamadas(t *testing.T) {
	proibidos := map[string][]string{
		"/firewall/sanitizar": {modulo + "/"},
		"/pipeline":           {modulo + "/firewall"},
		"/firewall/entrada":   {modulo + "/pipeline", modulo + "/operadores"},
	}
	ds := deps(t)
	for pkg, prefixos := range proibidos {
		for _, d := range ds[pkg] {
			for _, p := range prefixos {
				if strings.HasPrefix(d, p) {
					t.Errorf("%s importa %s", pkg, d)
				}
			}
		}
	}
}

// Regra ISA: fora de operadores/, ninguém escreve uma instrução crua
// (func(*quadro.Quadro) error). Processo novo é sequência de operadores; se
// faltar uma instrução, ela entra na biblioteca.
func TestSoAOperadoresDefineInstrucoes(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir(".", func(caminho string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (caminho == "operadores" || strings.HasPrefix(d.Name(), ".")) && caminho != "." {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(caminho, ".go") {
			return nil
		}
		arq, err := parser.ParseFile(fset, caminho, nil, parser.SkipObjectResolution)
		if err != nil {
			return err
		}
		ast.Inspect(arq, func(n ast.Node) bool {
			var ft *ast.FuncType
			switch f := n.(type) {
			case *ast.FuncLit:
				ft = f.Type
			case *ast.FuncDecl:
				if f.Recv == nil {
					ft = f.Type
				}
			}
			if ft != nil && instrucao(ft) {
				t.Errorf("%s: instrução crua fora de operadores/", fset.Position(n.Pos()))
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func instrucao(ft *ast.FuncType) bool {
	if ft.Params.NumFields() != 1 || ft.Results.NumFields() != 1 {
		return false
	}
	estrela, ok := ft.Params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := estrela.X.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Quadro" {
		return false
	}
	res, ok := ft.Results.List[0].Type.(*ast.Ident)
	return ok && res.Name == "error"
}

// Regra do guia (§8): receita é uma lista de operadores, com todas as
// entradas declaradas. O reflexo mostra como a receita deveria ser descrita.
func TestReceitasUsamOperadoresAoMaximo(t *testing.T) {
	rs, err := reflexo.Ler(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(rs) == 0 {
		t.Fatal("nenhuma receita encontrada; a checagem não está olhando para nada")
	}
	for _, r := range rs {
		if len(r.Problemas()) > 0 {
			t.Errorf("%s\n  problemas: %s", r.Reflexo(), strings.Join(r.Problemas(), "; "))
		}
	}
}
