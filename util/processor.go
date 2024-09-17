package util

import (
	"context"
	"errors"
	"sync"
)

type Processor[T any] interface {
	Items() []T
	Process(context.Context, T) error
}

func ProcessInParallel[T any](ctx context.Context, p Processor[T], concurrency int) (err error) {
	items := p.Items()
	if len(items) == 0 {
		return
	}

	workers := make(chan struct{}, concurrency)
	errChan := make(chan error, len(items))

	var wg sync.WaitGroup
	wg.Add(len(items))
	for _, item := range items {
		workers <- struct{}{}
		go func() {
			defer wg.Done()
			errChan <- p.Process(ctx, item)
			<-workers
		}()
	}

	wg.Wait()
	close(errChan)

	for e := range errChan {
		if e != nil {
			err = errors.Join(err, e)
		}
	}

	return
}
