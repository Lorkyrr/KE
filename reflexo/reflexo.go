// Package reflexo confere se as receitas declaradas com pipeline.Declarar
// usam o máximo de operadores pequenos, e mostra como cada uma deveria estar
// descrita. Lê só a sintaxe (go/ast): não executa nada e não importa o KE,
// então serve também aos projetos que montam as próprias receitas.
//
// É análise estática porque Op é uma função Go, que não carrega metadados:
// a lista de etapas só existe no código-fonte da declaração.
package reflexo

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Receita é uma chamada a Declarar, como está escrita no código.
type Receita struct {
	Nome   string
	Pos    token.Position
	Etapas []Etapa

	// Declaradas são as tags escritas em entradas; Lidas, as que algum
	// Converter lê. Ambas como texto do código, sem repetição.
	Declaradas []string
	Lidas      []string

	entradasLiterais bool
}

// Etapa é um argumento de Declarar depois das entradas.
type Etapa struct {
	Nome  string // "Converter", ou o texto do código se não for operador
	Atomo bool
}

// LerFonte extrai as receitas de um arquivo Go.
func LerFonte(nome string, src []byte) ([]Receita, error) {
	return ler(token.NewFileSet(), nome, src)
}

// Ler extrai as receitas de todos os .go sob dir, testes incluídos.
func Ler(dir string) ([]Receita, error) {
	fset := token.NewFileSet()
	var todas []Receita
	err := filepath.WalkDir(dir, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && caminho != dir && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(caminho, ".go") {
			return nil
		}
		rs, err := ler(fset, caminho, nil)
		todas = append(todas, rs...)
		return err
	})
	return todas, err
}

func ler(fset *token.FileSet, nome string, src any) ([]Receita, error) {
	arq, err := parser.ParseFile(fset, nome, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, fmt.Errorf("reflexo: %w", err)
	}
	var rs []Receita
	ast.Inspect(arq, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok && ehDeclarar(c.Fun) && len(c.Args) >= 2 {
			rs = append(rs, montar(fset, c))
		}
		return true
	})
	return rs, nil
}

func ehDeclarar(f ast.Expr) bool {
	switch f := f.(type) {
	case *ast.Ident: // dentro do próprio pacote pipeline
		return f.Name == "Declarar"
	case *ast.SelectorExpr:
		pkg, ok := f.X.(*ast.Ident)
		return ok && pkg.Name == "pipeline" && f.Sel.Name == "Declarar"
	}
	return false
}

func montar(fset *token.FileSet, c *ast.CallExpr) Receita {
	r := Receita{Pos: fset.Position(c.Pos()), Nome: types.ExprString(c.Args[0])}
	if lit, ok := c.Args[0].(*ast.BasicLit); ok {
		if s, err := strconv.Unquote(lit.Value); err == nil {
			r.Nome = s
		}
	}
	if cl, ok := c.Args[1].(*ast.CompositeLit); ok {
		r.entradasLiterais = true
		for _, e := range cl.Elts {
			r.Declaradas = append(r.Declaradas, types.ExprString(e))
		}
	}
	etapas := c.Args[2:]
	if c.Ellipsis.IsValid() && len(etapas) > 0 {
		// etapas... esconde a lista: não dá para ver os operadores.
		r.Etapas = []Etapa{{Nome: types.ExprString(etapas[len(etapas)-1]) + "..."}}
		return r
	}
	for _, e := range etapas {
		et := Etapa{Nome: types.ExprString(e), Atomo: ehAtomo(e)}
		if et.Atomo {
			et.Nome = nomeOperador(e)
		}
		if et.Nome == "Converter" && et.Atomo {
			if tag := tagLida(e); tag != "" && !slices.Contains(r.Lidas, tag) {
				r.Lidas = append(r.Lidas, tag)
			}
		}
		r.Etapas = append(r.Etapas, et)
	}
	return r
}

// ehAtomo diz se a etapa é uma chamada direta a um operador da biblioteca.
func ehAtomo(e ast.Expr) bool {
	// Só a chamada direta conta: func literal, helper local ou variável
	// escondem operação própria da receita, que deveria ser operadores.
	c, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && pkg.Name == "operadores"
}

func nomeOperador(e ast.Expr) string {
	return e.(*ast.CallExpr).Fun.(*ast.SelectorExpr).Sel.Name
}

// tagLida é o segundo argumento de Converter(dst, tag), como texto.
func tagLida(e ast.Expr) string {
	c := e.(*ast.CallExpr)
	if len(c.Args) != 2 {
		return ""
	}
	return types.ExprString(c.Args[1])
}

// Problemas lista o que foge do que o guia pede de uma receita (§8). Vazio
// quer dizer que a receita está como deveria.
func (r Receita) Problemas() []string {
	if len(r.Etapas) == 0 {
		return []string{"sem etapas"}
	}
	var ps []string
	saida := false
	for i, e := range r.Etapas {
		switch {
		case !e.Atomo:
			ps = append(ps, fmt.Sprintf("etapa %d (%s) não é um operador da biblioteca; decomponha em operadores", i, e.Nome))
		case e.Nome == "Transmitir" && i != len(r.Etapas)-1:
			ps = append(ps, fmt.Sprintf("etapa %d: Transmitir tem de ser o último passo", i))
		}
		saida = saida || e.Atomo && (e.Nome == "Escrever" || e.Nome == "Transmitir")
	}
	if !saida {
		ps = append(ps, "nenhuma etapa escreve a saída (Escrever) nem transmite o quadro (Transmitir)")
	}
	if !r.entradasLiterais {
		return append(ps, "entradas não é uma lista literal; não dá para conferir")
	}
	for _, t := range r.Lidas {
		if !slices.Contains(r.Declaradas, t) {
			ps = append(ps, fmt.Sprintf("Converter lê a tag %s, que não está nas entradas", t))
		}
	}
	return ps
}

// Reflexo mostra a receita como deveria estar descrita: as entradas que as
// etapas de fato leem, e cada etapa marcada como operador (✓) ou não (✗).
func (r Receita) Reflexo() string {
	atomos := 0
	for _, e := range r.Etapas {
		if e.Atomo {
			atomos++
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s (%s): %d/%d etapas são operadores\n", r.Nome, r.Pos, atomos, len(r.Etapas))
	fmt.Fprintf(&b, "  entradas deveriam ser: {%s}\n", strings.Join(r.Lidas, ", "))
	for i, e := range r.Etapas {
		marca := "✓"
		if !e.Atomo {
			marca = "✗"
		}
		fmt.Fprintf(&b, "  %d %s %s\n", i, marca, e.Nome)
	}
	return b.String()
}
