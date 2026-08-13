package cmd

type layerRecord struct {
	Branch string `json:"branch"`
	Parent string `json:"parent"`
	Merged bool   `json:"merged"`
	Merge  string `json:"merge"`
}
