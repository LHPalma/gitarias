package platform

import (
	"os"
	"os/exec"
	"runtime"
)

// Finder é o que a detecção precisa saber do sistema. É interface para que o
// teste possa descrever uma máquina que não é esta — sem ela, só daria para
// testar o Linux em que a suíte roda.
type Finder interface {
	OS() string
	Look(name string) bool
	Release() string
	Root() bool
}

// System é a implementação de verdade, e nada nela passa por shell: o PATH é
// percorrido pelo exec.LookPath e a distribuição sai de um arquivo.
type System struct{}

func (System) OS() string {
	return runtime.GOOS
}

func (System) Look(name string) bool {
	_, err := exec.LookPath(name)

	return err == nil
}

// Release devolve o conteúdo do /etc/os-release, e a string vazia quando não
// há arquivo — que é o caso de todo sistema que não é Linux. A falha de
// leitura não é distinguida da ausência de propósito: as duas significam a
// mesma coisa aqui, e o os.ReadFile já devolve nada em ambas.
func (System) Release() string {
	content, _ := os.ReadFile("/etc/os-release")

	return string(content)
}

func (System) Root() bool {
	return os.Geteuid() == 0
}
