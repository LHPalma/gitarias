package cmd

type branchesDocument struct {
	Base     baseRecord     `json:"base"`
	Branches []branchRecord `json:"branches"`
}
