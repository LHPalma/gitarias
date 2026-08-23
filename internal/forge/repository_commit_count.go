package forge

// RepositoryCommitCount é quantos commits a conta autenticada contribuiu num
// repositório, dentro do período pedido. Repository é sempre dono/nome —
// nameWithOwner —, porque é o que desambigua repositórios de mesmo nome
// entre contas diferentes.
type RepositoryCommitCount struct {
	Repository string
	Private    bool
	Count      int
}
