package testutils

import "github.com/stretchr/testify/mock"

var (
	ContextBackgroundMock = mock.AnythingOfType("context.backgroundCtx")
)
