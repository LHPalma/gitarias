package platformtest

import "testing"

func TestFinderAnswersWhatItWasToldToAnswer(t *testing.T) {
	finder := Finder{
		System:   "darwin",
		Present:  []string{"brew", "git"},
		Contents: "ID=macos\n",
		Elevated: true,
	}

	if finder.OS() != "darwin" {
		t.Errorf("sistema = %q", finder.OS())
	}
	if !finder.Look("brew") || !finder.Look("git") {
		t.Error("o que foi declarado presente tem de ser achado")
	}
	if finder.Look("apt-get") {
		t.Error("o que nao foi declarado nao pode ser achado, senao o teste do dominio mente")
	}
	if finder.Release() != "ID=macos\n" {
		t.Errorf("release = %q", finder.Release())
	}
	if !finder.Root() {
		t.Error("root declarado tem de sair como root")
	}
}

func TestFinderOfAnEmptyMachine(t *testing.T) {
	finder := Finder{}

	if finder.Look("qualquer") {
		t.Error("maquina sem nada declarado nao acha nada")
	}
	if finder.Root() {
		t.Error("sem declarar, nao e root")
	}
}
