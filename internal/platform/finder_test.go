package platform

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

func TestSystemReportsTheOperatingSystemItWasBuiltFor(t *testing.T) {
	if got := (System{}).OS(); got != runtime.GOOS {
		t.Errorf("sistema = %q, queria %q", got, runtime.GOOS)
	}
}

func TestSystemLooksInThePath(t *testing.T) {
	if !(System{}).Look("go") {
		t.Error("o go esta no PATH de quem roda a suite, senao ela nao teria rodado")
	}
	if (System{}).Look("nao-existe-mesmo-este-comando") {
		t.Error("comando inventado nao pode ser encontrado")
	}
}

func TestSystemReadsTheDistributionFile(t *testing.T) {
	release := (System{}).Release()

	if runtime.GOOS != "linux" {
		if release != "" {
			t.Errorf("release = %q; fora do Linux nao ha /etc/os-release a ler", release)
		}
		return
	}

	if _, err := os.Stat("/etc/os-release"); err != nil {
		t.Skip("este Linux nao tem /etc/os-release")
	}
	if !strings.Contains(release, "ID=") {
		t.Errorf("release = %q, queria o conteudo do /etc/os-release", release)
	}
}

func TestSystemKnowsWhetherItIsRoot(t *testing.T) {
	if got := (System{}).Root(); got != (os.Geteuid() == 0) {
		t.Errorf("root = %v, queria %v", got, os.Geteuid() == 0)
	}
}
