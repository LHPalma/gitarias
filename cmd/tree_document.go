package cmd

type treeDocument struct {
	Base   baseRecord    `json:"base"`
	Layers []layerRecord `json:"layers"`
}
