package main

import (
	"strings"
	"sync"
)

// call f(0), f(1), ..., f(n-1) on separate goroutines; run up to numCores goroutines at once.
func parallelDo(n int, numCores int, f func(int)) {
	wg := sync.WaitGroup{}
	wg.Add(n)
	sema := make(chan struct{}, numCores)
	for i := 0; i < n; i++ {
		i := i // sigh
		go func() {
			sema <- struct{}{}
			f(i)
			<-sema
			wg.Done()
		}()
	}
	wg.Wait()
}

func join(n int, sep string, f func(int) string) string {
	ss := make([]string, n)
	for i := 0; i < n; i++ {
		ss[i] = f(i)
	}
	return strings.Join(ss, sep)
}
