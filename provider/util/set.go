package util

type Set[T comparable] map[T]struct{}

func (s Set[T]) Add(item ...T) {
	for _, item := range item {
		s[item] = struct{}{}
	}
}

func (s Set[T]) Contains(item T) bool {
	_, ok := s[item]
	return ok
}

func (s Set[T]) ToSlice() (list []T) {
	list = make([]T, 0, len(s))
	for k := range s {
		list = append(list, k)
	}
	return
}

func NewSet[T comparable](item ...T) Set[T] {
	set := make(Set[T])
	set.Add(item...)
	return set
}

func NewSetFromSlice[T comparable](s []T) (set Set[T]) {
	set = NewSet[T]()
	for _, item := range s {
		set.Add(item)
	}
	return
}

func Unique[T comparable](list []T) []T {
	return NewSetFromSlice(list).ToSlice()
}
