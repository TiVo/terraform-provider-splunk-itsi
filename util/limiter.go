package util

type Limiter chan struct{}

func NewLimiter(n int) *Limiter {
	l := make(Limiter, n)
	for range n {
		l <- struct{}{}
	}
	return &l
}

func (l *Limiter) Acquire() {
	if cap(*l) == 0 {
		return
	}
	<-(*l)
}

func (l *Limiter) Release() {
	if cap(*l) == 0 {
		return
	}
	(*l) <- struct{}{}
}
