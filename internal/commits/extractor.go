package commits

import "context"

type Extractor interface {
	Extract(ctx context.Context, sha string, destination string) error
	Release(ctx context.Context, destination string)
}
