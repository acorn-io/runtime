package middleware

import "github.com/acorn-io/mink/pkg/strategy"

type CompleteStrategy func(strategy strategy.CompleteStrategy) strategy.CompleteStrategy

func ForCompleteStrategy(s strategy.CompleteStrategy, middlewares ...CompleteStrategy) strategy.CompleteStrategy {
	for i := len(middlewares) - 1; i >= 0; i-- {
		s = middlewares[i](s)
	}
	return s
}
