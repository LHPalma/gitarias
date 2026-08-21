package ignore

// Destination é onde o Add grava o padrão novo — um dos três arquivos que
// --exclude-standard enxerga (ver Entry). A escolha é sempre de quem roda,
// nunca inferida: o gtr não elege destino em silêncio.
type Destination int

const (
	// Repository é <raiz>/.gitignore — convenção do time, revisável em PR.
	Repository Destination = iota
	// Local é .git/info/exclude — vale só neste clone.
	Local
	// Global é core.excludesFile — vale na máquina inteira.
	Global
)
