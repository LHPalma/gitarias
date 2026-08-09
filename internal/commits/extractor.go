package commits

type Extractor interface {
	Extract(sha string, destination string) error
	Release(destination string)
}
