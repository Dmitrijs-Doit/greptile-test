package iface

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

//go:generate mockery --name Billing
type Billing interface {
	GetCoveredUsage(ctx context.Context, accountID string, from Payer) (CoveredUsage, error)
}

type EventType int

const (
	MovedAccount EventType = iota
	LeftAccount
)

func (t EventType) String() string {
	switch t {
	case MovedAccount:
		return "Moved account"
	case LeftAccount:
		return "Left account"
	default:
		return ""
	}
}

//go:generate mockery --name Notifier
type Notifier interface {
	NotifyIfNecessary(ctx context.Context, move AccountMove, eventType EventType) error
}

//go:generate mockery --name NotificationPublisher
type NotificationPublisher interface {
	PublishSlackNotification(ctx context.Context, notification map[string]interface{}) error
}

type AccountMove struct {
	AccountID   string
	AccountName string
	FromPayer   Payer
	ToPayer     Payer
}

func NewAccountMove(accountID, accountName string, fromPayer, toPayer Payer) AccountMove {
	return AccountMove{
		AccountID:   accountID,
		AccountName: accountName,
		FromPayer:   fromPayer,
		ToPayer:     toPayer,
	}
}

type Payer struct {
	ID          string
	DisplayName string
}

func (p *Payer) GetNumber() (int, error) {
	parts := strings.Split(p.DisplayName, "#")
	if len(parts) != 2 {
		return 0, fmt.Errorf("unexpected display name %s format", p.DisplayName)
	}

	return strconv.Atoi(parts[1])
}

func NewPayer(id, displayName string) Payer {
	return Payer{
		ID:          id,
		DisplayName: displayName,
	}
}

type CoveredUsage struct {
	SPCost float64
	RICost float64
}

func NewCoveredUsage(spCost, riCost float64) CoveredUsage {
	return CoveredUsage{SPCost: spCost, RICost: riCost}
}
