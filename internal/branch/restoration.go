package branch

import "errors"

// ErrAlreadyExists e ErrGone são as duas recusas do Restore, distinguidas por
// errors.Is para que o front diga qual foi sem inspecionar texto.
var (
	ErrAlreadyExists = errors.New("já existe uma branch com esse nome")
	ErrGone          = errors.New("o commit não está mais no repositório")
)

// Restoration é o pedido de recriar uma branch numa ponta.
type Restoration struct {
	Name string
	SHA  string
}
