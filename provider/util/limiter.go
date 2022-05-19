package util

type Limiter chan struct{}

func NewLimiter(n int) *Limiter {
	l := make(Limiter, n)
	for i := 0; i < n; i++ {
		l <- struct{}{}
	}
	return &l
}

func (l *Limiter) Acquire() {
	if len(*l) == 0 {
		return
	}
	<-(*l)
}

func (l *Limiter) Release() {
	if len(*l) == 0 {
		return
	}
	(*l) <- struct{}{}
}
