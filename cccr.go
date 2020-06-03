package main

import (
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const mtcBasis = `default=aug-cc-pvtz
s,C,8236.0,1235.0,280.8, 79.27,25.59, 8.997,3.319
s,C,0.9059,0.3643,0.1285000
p,C,56.0,18.71,4.133,0.2827,0.3827,0.1209
d,C,30.0,10.0,3.3,1.097,0.318
f,C,7.0,2.3,0.7610
s,N,11420.0,1712.0,389.3,110.0,35.57,12.54,4.644
s,N,1.293,0.5118,0.1787
p,N,79.89,26.63,5.948,1.742,0.555,0.1725
d,N,45.0,15.0,5.0,1.654,0.469
f,N,9.9,3.3,1.093
s,O,15330.0,2299.0,522.4,147.3,47.55,16.76,6.207
s,O,1.752,0.6882,0.2384
p,O,103.5,34.46,7.749,2.28,0.7156,0.214
d,O,63.0,21.0,7.0,2.314,0.645
f,O,12.9,4.3,1.428
s,Mg,164900.0,24710.0,5628.0,1596.0,521.0;
s,Mg,188.0,73.01,29.90,12.54,4.306,1.826;
s,Mg,0.7417,0.0761,0.145,0.033,0.0129;
p,Mg,950.70,316.90,74.86,23.72,8.669,3.363;
p,Mg,1.310,0.4911,0.2364,0.08733,0.03237,0.00745;
d,Mg,1.601,0.686,0.126,0.294,0.0468;
f,Mg,1.372,0.588,0.094,0.252;`

// CcCR is implemented as a Program, assuming Molpro
type CcCR struct{}

func (c CcCR) MakeHead() []string {
	return []string{"memory,1125,m",
		"nocompress",
		"geomtyp=xyz",
		"angstrom",
		"geometry={"}
}

func (c CcCR) MakeFoot() []string {
	return []string{"}",
		"set,charge=" + charge,
		"set,spin=" + spin,
		"basis=" + "avtz",
		"hf",
		"{ccsd(t)}",
		"etz=energy",

		"basis=" + "avqz",
		"hf",
		"{ccsd(t)}",
		"eqz=energy",

		"basis=" + "av5z",
		"hf",
		"{ccsd(t)}",
		"e5z=energy",

		"basis=" + "vtz-dk",
		"dkroll=0",
		"hf",
		"{ccsd(t)}",
		"edk=energy",

		"basis=" + "vtz-dk",
		"dkroll=1",
		"hf",
		"{ccsd(t)}",
		"edkr=energy",
		"dkroll=0",

		"basis={",
		mtcBasis,
		"}",
		"hf",
		"{ccsd(t)}",
		"emt=energy",

		"basis={",
		mtcBasis,
		"}",
		"hf",
		"{ccsd(t);core}",
		"emtc=energy",

		"cccre=etz-((eqz-etz)/(4.5^(-4)-3.5^(-4)))*3.5^(-4)" +
			"+((e5z-etz+((eqz-etz)/(4.5^(-4)-3.5^(-4)))*(3.5^(-4)-5.5^(-4)))" +
			"/(0.7477488413*((3.5^(-4)-5.5^(-4)))-3.5^(-6)+5.5^(-6)))" +
			"*((0.7477488413*(3.5^(-4)))-3.5^(-6))" +
			"+emtc-emt+edkr-edk",
	}
}

func (c CcCR) MakeIn(names []string, coords []float64) []string {
	body := make([]string, 0)
	for i, _ := range names {
		tmp := make([]string, 0)
		tmp = append(tmp, names[i])
		for _, c := range coords[3*i : 3*i+3] {
			s := strconv.FormatFloat(c, 'f', 10, 64)
			tmp = append(tmp, s)
		}
		body = append(body, strings.Join(tmp, " "))
	}
	return MakeInput(c.MakeHead(), c.MakeFoot(), body)
}

func (c CcCR) WriteIn(filename string, names []string, coords []float64) {
	lines := c.MakeIn(names, coords)
	writelines := strings.Join(lines, "\n")
	err := ioutil.WriteFile(filename, []byte(writelines), 0755)
	if err != nil {
		panic(err)
	}
}

func (c CcCR) ReadOut(filename string) (result float64, err error) {
	runtime.LockOSThread()
	if _, err = os.Stat(filename); os.IsNotExist(err) {
		runtime.UnlockOSThread()
		return brokenFloat, ErrFileNotFound
	}
	err = ErrEnergyNotFound
	result = brokenFloat
	lines, _ := ReadFile(filename)
	// ASSUME blank file is only created when PBS runs
	// blank file has a single newline
	if len(lines) == 1 {
		if strings.Contains(strings.ToUpper(lines[0]), "ERROR") {
			return result, ErrFileContainsError
		}
		return result, ErrBlankOutput
	}
	for _, line := range lines {
		if strings.Contains(strings.ToUpper(line), "ERROR") {
			return result, ErrFileContainsError
		}
		if strings.Contains(line, energyLine) {
			split := SplitLine(line)
			for i, _ := range split {
				if strings.Contains(split[i], energyLine) {
					// take the thing right after search term
					// not the last entry in the line
					if i+1 < len(split) {
						// assume we found energy so no error
						// from default EnergyNotFound
						err = nil
						result, err = strconv.ParseFloat(split[i+2], 64)
						if err != nil {
							err = ErrEnergyNotParsed
							// false if parse fails
						}
					}
				}
			}
		}
		if strings.Contains(line, molproTerminated) && err != nil {
			err = ErrFinishedButNoEnergy
		}
	}
	runtime.UnlockOSThread()
	return result, err
}
