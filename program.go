package main

type Program interface {
	MakeHead() []string
	MakeFoot() []string
	MakeIn([]string, []float64) []string
	WriteIn(string, []string, []float64)
	ReadOut(string) (float64, error)
}
