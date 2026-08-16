package platform

// manager é um gerenciador de pacotes: como se chama, se precisa de
// privilégio, o que se digita para instalar, e como cada ferramenta se chama
// no repositório dele — o nome do pacote nem sempre é o nome do comando.
type manager struct {
	name     string
	elevates bool
	refresh  []string
	install  []string
	packages map[string]string
}

// managers está na ordem em que se procura, por sistema operacional. O
// primeiro que estiver no PATH ganha.
var managers = map[string][]manager{
	"darwin": {
		{
			name:     "brew",
			install:  []string{"brew", "install"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
	},
	"windows": {
		{
			name:     "winget",
			install:  []string{"winget", "install", "--id"},
			packages: map[string]string{"git": "Git.Git", "gh": "GitHub.cli"},
		},
		{
			name:     "choco",
			install:  []string{"choco", "install", "-y"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
		{
			name:     "scoop",
			install:  []string{"scoop", "install"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
	},
	"linux": {
		{
			name:     "apt-get",
			elevates: true,
			refresh:  []string{"apt-get", "update"},
			install:  []string{"apt-get", "install", "-y"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
		{
			name:     "dnf",
			elevates: true,
			install:  []string{"dnf", "install", "-y"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
		{
			name:     "pacman",
			elevates: true,
			install:  []string{"pacman", "-S", "--needed"},
			packages: map[string]string{"git": "git", "gh": "github-cli"},
		},
		{
			name:     "zypper",
			elevates: true,
			install:  []string{"zypper", "install", "-y"},
			packages: map[string]string{"git": "git", "gh": "gh"},
		},
		{
			name:     "apk",
			elevates: true,
			install:  []string{"apk", "add"},
			packages: map[string]string{"git": "git", "gh": "github-cli"},
		},
	},
}
