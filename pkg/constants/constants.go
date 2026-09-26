package constants

import (
	"errors"
	"fmt"
)

const (
	DefaultLimit = 10
	DefaultPage  = 1
	MaxPageLimit = 1000
)

const (
	NoRowAffectedMsg            = "no row affected"
	FailProduceStoreLogoURLMsg  = "fail to produce store logo url"
	RefundNotSupportMsgTmpl     = "refund not support for oid: %d"
	TransactionClientNotGormMsg = "transaction client is not gorm"
)

var (
	ErrNoRowAffected         = errors.New(NoRowAffectedMsg)
	ErrInvalidTask           = fmt.Errorf("invalid task")
	ErrTaskAlreadyRegistered = fmt.Errorf("task already registered")
)

var (
	ErrInvalidID = errors.New("invalid id")
)
