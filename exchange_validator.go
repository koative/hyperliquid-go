package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
)

// CSignerAction jails or unjails the signer's validator. Set exactly one
// field.
type CSignerAction struct {
	JailSelf   bool
	UnjailSelf bool
}

func (CSignerAction) actionType() string { return "CSignerAction" }

// MarshalJSON encodes a as {"jailSelf":null} or {"unjailSelf":null}.
func (a CSignerAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.JailSelf && !a.UnjailSelf:
		return []byte(`{"jailSelf":null}`), nil
	case a.UnjailSelf && !a.JailSelf:
		return []byte(`{"unjailSelf":null}`), nil
	}
	return nil, errors.New("hyperliquid: CSignerAction: set exactly one of JailSelf and UnjailSelf")
}

// CSigner submits a validator signer action.
func (c *ExchangeClient) CSigner(ctx context.Context, a CSignerAction) error {
	return c.do(ctx, a, nil)
}

// CValidatorAction registers, updates or unregisters the signer's
// validator. Set exactly one field.
type CValidatorAction struct {
	ChangeProfile *ValidatorProfileChange `json:"changeProfile,omitempty"`
	Register      *ValidatorRegistration  `json:"register,omitempty"`
	Unregister    bool                    `json:"-"`
}

func (CValidatorAction) actionType() string { return "CValidatorAction" }

// MarshalJSON encodes Unregister as {"unregister":null}.
func (a CValidatorAction) MarshalJSON() ([]byte, error) {
	type plain CValidatorAction
	if a.Unregister {
		return []byte(`{"unregister":null}`), nil
	}
	return json.Marshal(plain(a))
}

// CValidator submits a validator action.
func (c *ExchangeClient) CValidator(ctx context.Context, a CValidatorAction) error {
	return c.do(ctx, a, nil)
}

// NodeIP is a validator node's IP address.
type NodeIP struct {
	IP string `json:"Ip"`
}

// ValidatorProfileChange updates a validator's profile; nil fields are left
// unchanged.
type ValidatorProfileChange struct {
	NodeIP             *NodeIP  `json:"node_ip"`
	Name               *string  `json:"name"`
	Description        *string  `json:"description"`
	Unjailed           bool     `json:"unjailed"`
	DisableDelegations *bool    `json:"disable_delegations"`
	CommissionBps      *int     `json:"commission_bps"`
	Signer             *Address `json:"signer"`
}

// ValidatorRegistration registers a validator.
type ValidatorRegistration struct {
	Profile  ValidatorProfile `json:"profile"`
	Unjailed bool             `json:"unjailed"`
	// InitialWei is the initial self-delegation, in HYPE wei (1e-8).
	InitialWei uint64 `json:"initial_wei"`
}

// ValidatorProfile is the profile of a new validator.
type ValidatorProfile struct {
	NodeIP              NodeIP  `json:"node_ip"`
	Name                string  `json:"name"`
	Description         string  `json:"description"`
	DelegationsDisabled bool    `json:"delegations_disabled"`
	CommissionBps       int     `json:"commission_bps"`
	Signer              Address `json:"signer"`
}

// ValidatorL1StreamAction is a validator's vote on the risk-free rate of
// aligned quote assets.
type ValidatorL1StreamAction struct {
	RiskFreeRate Decimal `json:"riskFreeRate"`
}

func (ValidatorL1StreamAction) actionType() string { return "validatorL1Stream" }

// ValidatorL1Stream submits a validator's risk-free rate vote.
func (c *ExchangeClient) ValidatorL1Stream(ctx context.Context, a ValidatorL1StreamAction) error {
	return c.do(ctx, a, nil)
}
