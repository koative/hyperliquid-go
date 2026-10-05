package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

// Hip3LiquidatorTransferAction deposits into or withdraws from a HIP-3 dex's
// backstop liquidator.
type Hip3LiquidatorTransferAction struct {
	Dex string `json:"dex"`
	// Ntl is the amount in quote token units ×1e6; it must be a multiple of
	// 1e9.
	Ntl       uint64 `json:"ntl"`
	IsDeposit bool   `json:"isDeposit"`
}

func (Hip3LiquidatorTransferAction) actionType() string { return "hip3LiquidatorTransfer" }

// Hip3LiquidatorTransfer moves funds to or from a HIP-3 backstop
// liquidator.
func (c *ExchangeClient) Hip3LiquidatorTransfer(ctx context.Context, a Hip3LiquidatorTransferAction) error {
	return c.do(ctx, a, nil)
}

// GossipPriorityBidAction bids in the Dutch auction for a gossip priority
// slot, which delivers prioritized mempool data to an IP address.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/priority-fees.
type GossipPriorityBidAction struct {
	// SlotID is 0 or 1.
	SlotID int    `json:"slotId"`
	IP     string `json:"ip"`
	// MaxGas is the most to pay, in HYPE wei (1e-8).
	MaxGas uint64 `json:"maxGas"`
}

func (GossipPriorityBidAction) actionType() string { return "gossipPriorityBid" }

// GossipPriorityBid bids for a gossip priority slot.
func (c *ExchangeClient) GossipPriorityBid(ctx context.Context, a GossipPriorityBidAction) error {
	return c.do(ctx, a, nil)
}

// FinalizeEvmContractAction completes the link between a spot token and an
// ERC-20 contract requested with [SpotDeployAction.RequestEvmContract]. It
// is signed by the contract's deployer.
type FinalizeEvmContractAction struct {
	Token int                      `json:"token"`
	Input FinalizeEvmContractInput `json:"input"`
}

func (FinalizeEvmContractAction) actionType() string { return "finalizeEvmContract" }

// FinalizeEvmContract finalizes a spot token's EVM contract link.
func (c *ExchangeClient) FinalizeEvmContract(ctx context.Context, a FinalizeEvmContractAction) error {
	return c.do(ctx, a, nil)
}

// FinalizeEvmContractInput is how the finalizer proves it deployed the
// contract.
type FinalizeEvmContractInput struct {
	slot  string
	nonce uint64
}

var (
	// FinalizeFirstStorageSlot proves deployment by the finalizer address
	// stored in the contract's first storage slot.
	FinalizeFirstStorageSlot = FinalizeEvmContractInput{slot: "firstStorageSlot"}
	// FinalizeCustomStorageSlot proves deployment by the finalizer address
	// stored at slot keccak256("HyperCore deployer").
	FinalizeCustomStorageSlot = FinalizeEvmContractInput{slot: "customStorageSlot"}
)

// FinalizeCreate proves deployment of a contract created by the finalizer
// account with the given transaction nonce.
func FinalizeCreate(nonce uint64) FinalizeEvmContractInput {
	return FinalizeEvmContractInput{nonce: nonce}
}

// MarshalJSON encodes in as "firstStorageSlot", "customStorageSlot" or
// {"create":{"nonce":n}}.
func (in FinalizeEvmContractInput) MarshalJSON() ([]byte, error) {
	if in.slot != "" {
		return json.Marshal(in.slot)
	}
	return fmt.Appendf(nil, `{"create":{"nonce":%d}}`, in.nonce), nil
}

// Aqav2Role is a role of an aligned quote asset (AQAv2) token.
type Aqav2Role string

// AQAv2 roles.
const (
	Aqav2RoleTechnical Aqav2Role = "technical"
	Aqav2RoleTreasury  Aqav2Role = "treasury"
)

// AuthorizeAqav2RoleAction authorizes the signer for a role of an aligned
// quote asset token.
type AuthorizeAqav2RoleAction struct {
	Token int       `json:"token"`
	Role  Aqav2Role `json:"role"`
}

func (AuthorizeAqav2RoleAction) actionType() string { return "authorizeAqav2Role" }

// AuthorizeAqav2Role authorizes an aligned quote asset role.
func (c *ExchangeClient) AuthorizeAqav2Role(ctx context.Context, a AuthorizeAqav2RoleAction) error {
	return c.do(ctx, a, nil)
}
