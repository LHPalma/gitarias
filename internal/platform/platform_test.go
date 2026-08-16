package platform

import (
	"strings"
	"testing"

	"github.com/LHPalma/gitarias/internal/platform/platformtest"
)

const ubuntu = "PRETTY_NAME=\"Ubuntu 24.04.1 LTS\"\nID=ubuntu\nVERSION_ID=\"24.04\"\n"

func TestDetectFindsTheManagerOfEachSystem(t *testing.T) {
	tests := []struct {
		name    string
		system  string
		present []string
		want    string
	}{
		{name: "debian e derivados", system: "linux", present: []string{"apt-get"}, want: "apt-get"},
		{name: "fedora", system: "linux", present: []string{"dnf"}, want: "dnf"},
		{name: "arch", system: "linux", present: []string{"pacman"}, want: "pacman"},
		{name: "suse", system: "linux", present: []string{"zypper"}, want: "zypper"},
		{name: "alpine", system: "linux", present: []string{"apk"}, want: "apk"},
		{name: "macos", system: "darwin", present: []string{"brew"}, want: "brew"},
		{name: "windows", system: "windows", present: []string{"winget"}, want: "winget"},
		{name: "windows com choco", system: "windows", present: []string{"choco"}, want: "choco"},
		{name: "windows com scoop", system: "windows", present: []string{"scoop"}, want: "scoop"},
		{name: "linux sem gerenciador", system: "linux", want: ""},
		{name: "sistema desconhecido", system: "plan9", present: []string{"apt-get"}, want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := Detect(platformtest.Finder{System: test.system, Present: test.present})

			if got := machine.Manager(); got != test.want {
				t.Errorf("gerenciador = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestDetectPrefersTheFirstOfTheOrder(t *testing.T) {
	machine := Detect(platformtest.Finder{System: "windows", Present: []string{"scoop", "winget", "choco"}})

	if machine.Manager() != "winget" {
		t.Errorf("gerenciador = %q; havendo mais de um, a ordem da tabela decide e nao a do PATH", machine.Manager())
	}
}

func TestInstallSpellsOutTheCommandsInOrder(t *testing.T) {
	machine := Detect(platformtest.Finder{System: "linux", Present: []string{"apt-get"}, Elevated: true})

	got := strings.Join(machine.Install("git"), " ; ")
	want := "apt-get update ; apt-get install -y git"

	if got != want {
		t.Errorf("comandos = %q, queria %q: o update vem antes e nao pode se perder", got, want)
	}
}

func TestInstallAsksForPrivilegeOnlyWhenItIsNeeded(t *testing.T) {
	tests := []struct {
		name     string
		system   string
		present  []string
		elevated bool
		sudo     bool
	}{
		{name: "apt sem ser root", system: "linux", present: []string{"apt-get"}, sudo: true},
		{name: "apt como root", system: "linux", present: []string{"apt-get"}, elevated: true},
		{name: "brew nunca pede", system: "darwin", present: []string{"brew"}},
		{name: "winget nunca pede", system: "windows", present: []string{"winget"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := Detect(platformtest.Finder{System: test.system, Present: test.present, Elevated: test.elevated})

			commands := strings.Join(machine.Install("git"), " ")
			if got := strings.Contains(commands, "sudo"); got != test.sudo {
				t.Errorf("comandos = %q, queria sudo = %v", commands, test.sudo)
			}
		})
	}
}

func TestInstallKnowsThatThePackageIsNotAlwaysTheCommand(t *testing.T) {
	arch := Detect(platformtest.Finder{System: "linux", Present: []string{"pacman"}, Elevated: true})

	if got := strings.Join(arch.Install("gh"), " "); !strings.Contains(got, "github-cli") {
		t.Errorf("comandos = %q; no Arch o pacote do gh chama github-cli, e chutar o nome daria um comando que falha", got)
	}

	windows := Detect(platformtest.Finder{System: "windows", Present: []string{"winget"}})
	if got := strings.Join(windows.Install("git"), " "); !strings.Contains(got, "Git.Git") {
		t.Errorf("comandos = %q; o winget instala por identificador", got)
	}
}

func TestInstallSaysNothingWhenItCannotPromise(t *testing.T) {
	tests := []struct {
		name    string
		machine Platform
		tool    string
	}{
		{
			name:    "sem gerenciador",
			machine: Detect(platformtest.Finder{System: "linux"}),
			tool:    "git",
		},
		{
			name:    "ferramenta desconhecida",
			machine: Detect(platformtest.Finder{System: "linux", Present: []string{"apt-get"}}),
			tool:    "kubectl",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.machine.Install(test.tool); got != nil {
				t.Errorf("comandos = %v; comando inventado e pior que nenhum", got)
			}
		})
	}
}

func TestDescribeNamesTheDistributionWhenThereIsOne(t *testing.T) {
	tests := []struct {
		name     string
		system   string
		contents string
		want     string
	}{
		{name: "ubuntu", system: "linux", contents: ubuntu, want: "ubuntu 24.04"},
		{name: "sem versao", system: "linux", contents: "ID=arch\n", want: "arch"},
		{name: "sem os-release", system: "darwin", want: "darwin"},
		{name: "os-release sem ID", system: "linux", contents: "PRETTY_NAME=\"qualquer\"\n", want: "linux"},
		{name: "linha sem igual", system: "linux", contents: "lixo\nID=debian\n", want: "debian"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			machine := Detect(platformtest.Finder{System: test.system, Contents: test.contents})

			if got := machine.Describe(); got != test.want {
				t.Errorf("descricao = %q, queria %q", got, test.want)
			}
		})
	}
}

func TestHomepageCoversEveryToolTheTableKnows(t *testing.T) {
	for _, list := range managers {
		for _, candidate := range list {
			for tool := range candidate.packages {
				if Homepage[tool] == "" {
					t.Errorf("a ferramenta %q nao tem pagina: sem gerenciador reconhecido nao haveria o que dizer", tool)
				}
			}
		}
	}
}
