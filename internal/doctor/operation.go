package doctor

// operation é uma operação de git que fica no meio do caminho quando para, com
// a ref que o git mantém enquanto ela dura.
type operation struct {
	name string
	ref  string
}

// operations são as operações detectáveis por ref. O git am não está aqui
// porque não deixa ref; é reconhecido à parte, em Doctor.applying.
var operations = []operation{
	{name: "merge", ref: "MERGE_HEAD"},
	{name: "cherry-pick", ref: "CHERRY_PICK_HEAD"},
	{name: "revert", ref: "REVERT_HEAD"},
	{name: "rebase", ref: "REBASE_HEAD"},
}
