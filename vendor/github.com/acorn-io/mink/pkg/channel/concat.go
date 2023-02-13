package channel

func Concat[T any](left, right chan T) chan T {
	c := &concatChan[T]{
		left:  left,
		right: right,
		C:     make(chan T),
	}
	go c.run()
	return c.C
}

type concatChan[T any] struct {
	left, right chan T
	C           chan T
}

func (c *concatChan[T]) run() {
	defer close(c.C)

	var buffer []T
loop1:
	for {
		select {
		case x, ok := <-c.left:
			if !ok {
				break loop1
			}
			c.C <- x
		case x, ok := <-c.right:
			if !ok {
				break loop1
			}
			buffer = append(buffer, x)
		}
	}

	// left might be open and right closed, so ensure left is closed
	for x := range c.left {
		c.C <- x
	}

	for _, x := range buffer {
		c.C <- x
	}

	for x := range c.right {
		c.C <- x
	}
}
