package domain

type Permission struct {
	Desc  string `firestore:"desc"`
	Order int64  `firestore:"order"`
	Title string `firestore:"title"`
	ID    string `firestore:"-"`
}
