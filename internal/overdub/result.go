package overdub

// Result é o que Overdub mudou: os shas completos de depois, para quem
// chama verificar o resultado (NewTarget, NewHead). A recuperação usa o
// Head que Plan já devolveu antes da confirmação — Overdub não repete essa
// resolução.
type Result struct {
	NewTarget string
	NewHead   string
}
