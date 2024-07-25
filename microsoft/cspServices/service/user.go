package service

type UsersService struct {
	s *CSPServiceClient
}

func NewUsersService(s *CSPServiceClient) *UsersService {
	us := &UsersService{s: s}
	return us
}
