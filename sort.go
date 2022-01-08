package ogdl

import (
	"sort"
	"strings"
)

type stringSorter struct {
	fields string
	array  []*Graph
}

func (g *Graph) SortByString(fields string) {

	ff := strings.Fields(fields)

	for _, f := range ff {
		s := stringSorter{f, g.Out}
		sort.Sort(s)
	}
}

func (g stringSorter) Swap(i, j int) { g.array[i], g.array[j] = g.array[j], g.array[i] }
func (g stringSorter) Less(i, j int) bool {

	a := g.array[i].Get(g.fields).String()
	b := g.array[j].Get(g.fields).String()

	return strings.Compare(a, b) == -1
}
func (g stringSorter) Len() int {
	return len(g.array)
}
