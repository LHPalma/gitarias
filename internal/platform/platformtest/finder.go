package platformtest

// Finder descreve uma máquina que não é a de quem roda o teste, para que a
// detecção seja exercitada em todos os sistemas a partir de um só.
type Finder struct {
	System   string
	Present  []string
	Contents string
	Elevated bool
}

func (finder Finder) OS() string {
	return finder.System
}

func (finder Finder) Look(name string) bool {
	for _, present := range finder.Present {
		if present == name {
			return true
		}
	}

	return false
}

func (finder Finder) Release() string {
	return finder.Contents
}

func (finder Finder) Root() bool {
	return finder.Elevated
}
