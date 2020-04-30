package main

import (
	"fmt"
	"gonum.org/v1/gonum/stat/combin"
)

func Pow(i, j int) int {
	if j == 0 {
		return 1
	}
	total := 1
	for n := 0; n < j; n++ {
		total *= i
	}
	return total
}

func Central(n float64, v string) string {
	formula := ""
	for i := 0; i <= int(n); i++ {
		coeff := Pow(-1, i) * combin.Binomial(int(n), i)
		if coeff > 0 {
			formula += fmt.Sprintf("+%df(%.0fD%s)", coeff, 2*(n/2 - float64(i)), v)
		} else {
			formula += fmt.Sprintf("%df(%.0fD%s)", coeff, 2*(n/2 - float64(i)), v)
		}

	}
	return formula
}
