package hyperliquid

import "context"

// CreateVaultAction creates a vault led by the account.
type CreateVaultAction struct {
	// Name is 3 to 50 characters.
	Name string `json:"name"`
	// Description is 10 to 250 characters.
	Description string `json:"description"`
	// InitialUsd is the leader's initial deposit in USDC times 1e6; at least
	// 100 USDC.
	InitialUsd uint64 `json:"initialUsd"`
}

func (CreateVaultAction) actionType() string { return "createVault" }
func (CreateVaultAction) l1Nonce()           {}

// CreateVault creates a vault and returns its address.
func (c *ExchangeClient) CreateVault(ctx context.Context, a CreateVaultAction) (Address, error) {
	var addr Address
	return addr, c.do(ctx, a, &addr)
}

// VaultModifyAction changes a vault's settings. Nil fields are unchanged.
type VaultModifyAction struct {
	VaultAddress          Address `json:"vaultAddress"`
	AllowDeposits         *bool   `json:"allowDeposits"`
	AlwaysCloseOnWithdraw *bool   `json:"alwaysCloseOnWithdraw"`
}

func (VaultModifyAction) actionType() string { return "vaultModify" }

// VaultModify changes a vault's settings.
func (c *ExchangeClient) VaultModify(ctx context.Context, a VaultModifyAction) error {
	return c.do(ctx, a, nil)
}

// VaultDistributeAction distributes vault funds to its followers.
type VaultDistributeAction struct {
	VaultAddress Address `json:"vaultAddress"`
	// Usd is the USDC amount times 1e6; 0 closes the vault.
	Usd uint64 `json:"usd"`
}

func (VaultDistributeAction) actionType() string { return "vaultDistribute" }

// VaultDistribute distributes vault funds to its followers.
func (c *ExchangeClient) VaultDistribute(ctx context.Context, a VaultDistributeAction) error {
	return c.do(ctx, a, nil)
}

// VaultTransferAction deposits USDC into or withdraws it from a vault.
type VaultTransferAction struct {
	VaultAddress Address `json:"vaultAddress"`
	IsDeposit    bool    `json:"isDeposit"`
	// Usd is the USDC amount times 1e6.
	Usd uint64 `json:"usd"`
}

func (VaultTransferAction) actionType() string { return "vaultTransfer" }

// VaultTransfer deposits into or withdraws from a vault.
func (c *ExchangeClient) VaultTransfer(ctx context.Context, a VaultTransferAction) error {
	return c.do(ctx, a, nil)
}
