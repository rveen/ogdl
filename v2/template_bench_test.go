package ogdl

import (
	htmltemplate "html/template"
	"strconv"
	"strings"
	"testing"
)

// This file benchmarks the OGDL template engine in HTML mode against the
// standard library's html/template. The two templates below are written to
// produce byte-identical output (checked by TestTemplateBenchEquivalence)
// and exercise the same features: substitution with HTML escaping, if /
// else-if / else, a comparison, nested loops, a break, and a length.
//
// Run with:
//
//	go test -run '^$' -bench 'Template(Execute|Parse)' -benchmem

const benchOGDLTemplate = `<!DOCTYPE html>
<html>
<head><title>$title</title></head>
<body>
<h1>$title</h1>
$if(user.admin)<p class="admin">Welcome back, $user.name (administrator)</p>
$else<p>Welcome, $user.name</p>
$end<table>
$for(p, products)<tr>
<td>$p.name</td>
<td>$p.price</td>
<td>$if(p.price > 100)premium$elseif(p.stock)in stock$else<em>sold out</em>$end</td>
<td>$for(t, p.tags)$if(t == 'hidden')$break$end<span>$t</span>$end</td>
</tr>
$end</table>
<p>$products._len products</p>
</body>
</html>
`

const benchGoTemplate = `<!DOCTYPE html>
<html>
<head><title>{{.Title}}</title></head>
<body>
<h1>{{.Title}}</h1>
{{if .User.Admin}}<p class="admin">Welcome back, {{.User.Name}} (administrator)</p>
{{else}}<p>Welcome, {{.User.Name}}</p>
{{end}}<table>
{{range .Products}}<tr>
<td>{{.Name}}</td>
<td>{{.Price}}</td>
<td>{{if gt .Price 100}}premium{{else if .Stock}}in stock{{else}}<em>sold out</em>{{end}}</td>
<td>{{range .Tags}}{{if eq . "hidden"}}{{break}}{{end}}<span>{{.}}</span>{{end}}</td>
</tr>
{{end}}</table>
<p>{{len .Products}} products</p>
</body>
</html>
`

type benchUser struct {
	Name  string
	Admin bool
}

type benchProduct struct {
	Name  string
	Price int
	Stock bool
	Tags  []string
}

type benchPage struct {
	Title    string
	User     benchUser
	Products []benchProduct
}

// benchData builds the same page twice: as Go structs for html/template and
// as an OGDL tree (string data only, as a parsed stream would give).
func benchData(n int) (*benchPage, *Node) {
	page := &benchPage{
		Title: "Catalog & <Specials>",
		User:  benchUser{Name: `Jenny "J" O'Neil`, Admin: true},
	}
	allTags := []string{"new", "sale", "hidden", "<b>bold</b>", "eco"}
	for i := 0; i < n; i++ {
		p := benchProduct{
			Name:  "Widget " + strconv.Itoa(i),
			Price: (i * 37) % 250,
			Stock: i%3 != 0,
		}
		if i%5 == 0 {
			p.Name += ` <XL> & "Co"`
		}
		for j := 0; j < 1+i%5; j++ {
			p.Tags = append(p.Tags, allTags[(i+j)%len(allTags)])
		}
		page.Products = append(page.Products, p)
	}

	leaf := func(s string) *Node { return &Node{Data: s} }
	pair := func(k, v string) *Node { return &Node{Data: k, Out: []*Node{leaf(v)}} }

	products := &Node{Data: "products"}
	for _, p := range page.Products {
		tags := &Node{Data: "tags"}
		for _, t := range p.Tags {
			tags.Out = append(tags.Out, leaf(t))
		}
		products.Out = append(products.Out, &Node{Data: "item", Out: []*Node{
			pair("name", p.Name),
			pair("price", strconv.Itoa(p.Price)),
			pair("stock", strconv.FormatBool(p.Stock)),
			tags,
		}})
	}
	root := &Node{Out: []*Node{
		pair("title", page.Title),
		{Data: "user", Out: []*Node{
			pair("name", page.User.Name),
			pair("admin", strconv.FormatBool(page.User.Admin)),
		}},
		products,
	}}
	return page, root
}

var benchSizes = []int{10, 100, 1000}

func benchName(n int) string { return "n=" + strconv.Itoa(n) }

func renderGo(t *htmltemplate.Template, page *benchPage) (string, error) {
	var sb strings.Builder
	err := t.Execute(&sb, page)
	return sb.String(), err
}

// TestTemplateBenchEquivalence guards the benchmark: both engines must render
// exactly the same bytes, or the comparison means nothing.
func TestTemplateBenchEquivalence(t *testing.T) {
	ot, err := ParseHTMLTemplate(benchOGDLTemplate)
	if err != nil {
		t.Fatal(err)
	}
	gt, err := htmltemplate.New("page").Parse(benchGoTemplate)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range benchSizes {
		page, root := benchData(n)
		want, err := renderGo(gt, page)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ot.Process(root)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("n=%d: outputs differ\n--- ogdl ---\n%s\n--- html/template ---\n%s", n, got, want)
		}
	}
}

// BenchmarkTemplateExecute measures rendering a pre-parsed template.
func BenchmarkTemplateExecute(b *testing.B) {
	ot, err := ParseHTMLTemplate(benchOGDLTemplate)
	if err != nil {
		b.Fatal(err)
	}
	gt, err := htmltemplate.New("page").Parse(benchGoTemplate)
	if err != nil {
		b.Fatal(err)
	}

	for _, n := range benchSizes {
		page, root := benchData(n)
		out, err := renderGo(gt, page)
		if err != nil {
			b.Fatal(err)
		}

		b.Run("ogdl/"+benchName(n), func(b *testing.B) {
			b.SetBytes(int64(len(out)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ot.Process(root); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run("html-template/"+benchName(n), func(b *testing.B) {
			b.SetBytes(int64(len(out)))
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := renderGo(gt, page); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkTemplateParse measures compiling the template. html/template
// escapes contextually on first execution, so its cost is included by
// executing once against an empty page; the OGDL side does the same for
// symmetry.
func BenchmarkTemplateParse(b *testing.B) {
	page, root := benchData(0)

	b.Run("ogdl", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t, err := ParseHTMLTemplate(benchOGDLTemplate)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := t.Process(root); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("html-template", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			t, err := htmltemplate.New("page").Parse(benchGoTemplate)
			if err != nil {
				b.Fatal(err)
			}
			if _, err := renderGo(t, page); err != nil {
				b.Fatal(err)
			}
		}
	})
}
