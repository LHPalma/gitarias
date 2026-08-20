package cmd

type aiTrailerRecord struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Tool  string `json:"tool"`
}

type aiFindingRecord struct {
	Hash     string            `json:"hash"`
	Subject  string            `json:"subject"`
	Trailers []aiTrailerRecord `json:"trailers"`
}
