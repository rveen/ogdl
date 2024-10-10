package ogdl

import (
	"sort"
	"strings"
)

type stringSorter struct {
	fields string
	array  []*Graph
}

func (g *Graph) Sort(field string) {
	s := stringSorter{field, g.Out}
	sort.Sort(s)
}

func (g *Graph) SortAsString(field string) {
	s := stringSorter{field, g.Out}
	sort.Sort(s)
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

type intSorter struct {
	fields string
	array  []*Graph
}

func (g *Graph) SortAsInt(field string) {
	s := intSorter{field, g.Out}
	sort.Sort(s)
}

func (g intSorter) Swap(i, j int) { g.array[i], g.array[j] = g.array[j], g.array[i] }
func (g intSorter) Less(i, j int) bool {

	a := g.array[i].Get(g.fields).Int64()
	b := g.array[j].Get(g.fields).Int64()

	return a < b
}
func (g intSorter) Len() int {
	return len(g.array)
}
