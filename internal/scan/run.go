package scan

import (
	"context"
	"sync"
)

// Result is a full scan layer across all managers.
type Result struct {
	Rows []Row
}

// All returns the default scanner set.
func All() []Scanner {
	return []Scanner{Brew{}, Npm{}, Pnpm{}, Uv{}}
}

// Run scans every scanner in parallel and merges results. A failing
// scanner contributes no rows; its error is reported through errs.
func Run(ctx context.Context, scanners []Scanner) (Result, map[string]error) {
	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		rows []Row
		errs = map[string]error{}
	)
	for _, s := range scanners {
		wg.Add(1)
		go func(s Scanner) {
			defer wg.Done()
			r, err := s.Scan(ctx)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				errs[s.Name()] = err
			}
			rows = append(rows, r...)
		}(s)
	}
	wg.Wait()
	return Result{Rows: rows}, errs
}
