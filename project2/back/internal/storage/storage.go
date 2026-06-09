package storage

import (
	"context"
	"io"
)

type Uploader interface {
	Save(ctx context.Context, path string, r io.Reader) error
	Delete(ctx context.Context, path string) error
}
