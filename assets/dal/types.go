package dal

import "cloud.google.com/go/firestore"

type (
	QueryOption = func(firestore.Query) firestore.Query
	WhereQuery  struct {
		Path  string
		Op    string
		Value interface{}
	}
)

// WithWhereQuery applies a Firestore WhereQuery to a Firestore Query.
func WithWhereQuery(args WhereQuery) QueryOption {
	return func(q firestore.Query) firestore.Query {
		return q.Where(args.Path, args.Op, args.Value)
	}
}
