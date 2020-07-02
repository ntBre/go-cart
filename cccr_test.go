package main

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

var c = CcCR{}

func TestMakeCcCRIn(t *testing.T) {
	got := strings.Join(c.MakeIn([]string{"C", "N", "H"},
		[]float64{0.000000000, 0.000000000, -0.557831285,
			0.000000000, 0.000000000, 0.595166388,
			0.000000000, 0.000000000, -1.623316351}), "\n")
	want := cccrWant
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, wanted %v\n", got, want)
	}
}

func TestWriteCcCRIn(t *testing.T) {
	filename := "testfiles/cccr.in"
	c.WriteIn(filename, []string{"C", "N", "H"},
		[]float64{0.000000000, 0.000000000, -0.557831285,
			0.000000000, 0.000000000, 0.595166388,
			0.000000000, 0.000000000, -1.623316351})
}

func TestReadCcCROut(t *testing.T) {
	temp := energyLine
	energyLine = regexp.MustCompile(`^\s*CCCRE\s+=`)
	defer func() { energyLine = temp }()
	got, err := c.ReadOut("testfiles/cccr.out")
	want := -93.471416880846
	if got != want {
		t.Errorf("got %v, wanted %v, err %v\n", got, want, err)
	}
}

const cccrWant = `memory,1125,m
gthresh,energy=1.d-10,zero=1.d-16,oneint=1.d-16,twoint=1.d-16;
gthresh,optgrad=1.d-8,optstep=1.d-8;
nocompress
geomtyp=xyz
angstrom
geometry={
C 0.0000000000 0.0000000000 -0.5578312850
N 0.0000000000 0.0000000000 0.5951663880
H 0.0000000000 0.0000000000 -1.6233163510
}
set,charge=0
set,spin=0
basis=avtz
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
etz=energy
basis=avqz
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
eqz=energy
basis=av5z
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
e5z=energy
basis=vtz-dk
dkroll=0
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
edk=energy
basis=vtz-dk
dkroll=1
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
edkr=energy
dkroll=0
basis={
default=aug-cc-pvtz
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
f,Mg,1.372,0.588,0.094,0.252;
}
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;}
emt=energy
basis={
default=aug-cc-pvtz
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
f,Mg,1.372,0.588,0.094,0.252;
}
{hf,maxit=500;accu,20;}
{ccsd(t),nocheck,maxit=250;orbital,IGNORE_ERROR;core}
emtc=energy
cccre=etz-((eqz-etz)/(4.5^(-4)-3.5^(-4)))*3.5^(-4)+((e5z-etz+((eqz-etz)/(4.5^(-4)-3.5^(-4)))*(3.5^(-4)-5.5^(-4)))/(0.7477488413*((3.5^(-4)-5.5^(-4)))-3.5^(-6)+5.5^(-6)))*((0.7477488413*(3.5^(-4)))-3.5^(-6))+emtc-emt+edkr-edk
show[1,f20.12],cccre`
