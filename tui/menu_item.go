package tui

// MenuItem é uma entrada do menu interativo de comandos exibido pelo
// HelpMenu: o nome usado para invocar o comando, e a descrição de uma
// linha exibida ao lado.
type MenuItem struct {
	Name  string
	Short string
}
