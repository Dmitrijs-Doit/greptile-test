package dal

type Option func(dal *dal)

func WithPriorityUserName(userName string) Option {
	return func(d *dal) {
		d.priorityUserName = userName
	}
}

func WithPriorityPassword(password string) Option {
	return func(d *dal) {
		d.priorityPassword = password
	}
}
