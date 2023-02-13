package typed

import "time"

func Debounce[T any](in <-chan T) <-chan T {
	result := make(chan T, 1)
	go func() {
		for msg := range in {
			select {
			case result <- msg:
			default:
			}
		}
	}()
	return result
}

func Every[T any](duration time.Duration, in <-chan T) <-chan T {
	result := make(chan T)
	go func() {
		var (
			lastUpdate T
			timer      = time.NewTicker(duration)
		)
		defer close(result)
		defer timer.Stop()
		for {
			select {
			case currentUpdate, ok := <-in:
				if !ok {
					result <- lastUpdate
					return
				}
				lastUpdate = currentUpdate
			case <-timer.C:
				result <- lastUpdate
			}
		}
	}()
	return result
}

func Tee[T any](in <-chan T) (<-chan T, <-chan T) {
	one := make(chan T, 1)
	two := make(chan T, 1)

	go func() {
		defer close(one)
		defer close(two)
		for x := range in {
			one <- x
			two <- x
		}
	}()

	return one, two
}
