package parse

// Pred is a predicate for type T.
type Pred[T any] func(T) bool

// and combines the given predicates into one, using the and logical operator.
func and[T any](preds ...Pred[T]) Pred[T] {
	return func(t T) bool {
		for _, p := range preds {
			if !p(t) {
				return false
			}
		}
		return true
	}
}

// nor combines the given predicates into one, using the nor logical operator.
func nor[T any](preds ...Pred[T]) Pred[T] {
	return func(t T) bool {
		for _, p := range preds {
			if p(t) {
				return false
			}
		}
		return true
	}
}

func Map[T, U any](fun func(T) U, source []T) []U {
	res := make([]U, len(source))
	for i, el := range source {
		res[i] = fun(el)
	}
	return res
}
