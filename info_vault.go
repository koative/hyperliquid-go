package hyperliquid

import "context"

// VaultRelationship is a vault's place in a vault hierarchy.
type VaultRelationship struct {
	// Type is "normal", "parent" or "child".
	Type string `json:"type"`
	// Data is set for "parent" vaults.
	Data *VaultRelationshipData `json:"data,omitempty"`
}

// VaultRelationshipData lists a parent vault's children.
type VaultRelationshipData struct {
	ChildAddresses []Address `json:"childAddresses"`
}

// VaultDetailsRequest is the request for [InfoClient.VaultDetails].
type VaultDetailsRequest struct {
	VaultAddress Address `json:"vaultAddress"`
	// User optionally selects the follower reported in
	// [VaultDetails.FollowerState].
	User *Address `json:"user,omitempty"`
}

// VaultDetails describes a vault.
type VaultDetails struct {
	Name         string    `json:"name"`
	VaultAddress Address   `json:"vaultAddress"`
	Leader       Address   `json:"leader"`
	Description  string    `json:"description"`
	Portfolio    Portfolio `json:"portfolio"`
	// APR is a fraction: 0.05 means 5%.
	APR Decimal `json:"apr"`
	// FollowerState is the requested user's stake, or nil.
	FollowerState *VaultFollower `json:"followerState"`
	// LeaderFraction is the leader's share of the vault; LeaderCommission
	// the leader's share of follower profits.
	LeaderFraction   Decimal `json:"leaderFraction"`
	LeaderCommission Decimal `json:"leaderCommission"`
	// Followers includes the leader, whose User is Leader.
	Followers             []VaultFollower   `json:"followers"`
	MaxDistributable      Decimal           `json:"maxDistributable"`
	MaxWithdrawable       Decimal           `json:"maxWithdrawable"`
	IsClosed              bool              `json:"isClosed"`
	Relationship          VaultRelationship `json:"relationship"`
	AllowDeposits         bool              `json:"allowDeposits"`
	AlwaysCloseOnWithdraw bool              `json:"alwaysCloseOnWithdraw"`
}

// UnmarshalJSON decodes d, resolving the leader's follower entry, which the
// wire marks "Leader" instead of an address.
func (d *VaultDetails) UnmarshalJSON(b []byte) error {
	type plain VaultDetails
	var v struct {
		plain
		Followers []struct {
			User string `json:"user"`
			VaultFollower
		} `json:"followers"`
	}
	if err := unmarshal(b, &v); err != nil {
		return err
	}
	*d = VaultDetails(v.plain)
	d.Followers = make([]VaultFollower, len(v.Followers))
	for i, f := range v.Followers {
		d.Followers[i] = f.VaultFollower
		d.Followers[i].User = d.Leader
		if f.User != "Leader" {
			if err := d.Followers[i].User.UnmarshalText([]byte(f.User)); err != nil {
				return err
			}
		}
	}
	return nil
}

// VaultFollower is a depositor's stake in a vault.
type VaultFollower struct {
	User        Address `json:"user"`
	VaultEquity Decimal `json:"vaultEquity"`
	// Pnl is since the last deposit; AllTimePnl over the follower's
	// lifetime.
	Pnl           Decimal `json:"pnl"`
	AllTimePnl    Decimal `json:"allTimePnl"`
	DaysFollowing int     `json:"daysFollowing"`
	// VaultEntryTime and LockupUntil are in milliseconds since the Unix
	// epoch.
	VaultEntryTime int64 `json:"vaultEntryTime"`
	LockupUntil    int64 `json:"lockupUntil"`
}

// VaultDetails returns a vault's details, or nil if it does not exist.
func (c *InfoClient) VaultDetails(ctx context.Context, req VaultDetailsRequest) (*VaultDetails, error) {
	return infoRequest[*VaultDetails](ctx, c, "vaultDetails", req)
}

// VaultSummary is a vault in [InfoClient.VaultSummaries].
type VaultSummary struct {
	Name         string            `json:"name"`
	VaultAddress Address           `json:"vaultAddress"`
	Leader       Address           `json:"leader"`
	TVL          Decimal           `json:"tvl"`
	IsClosed     bool              `json:"isClosed"`
	Relationship VaultRelationship `json:"relationship"`
	// CreateTimeMillis is in milliseconds since the Unix epoch.
	CreateTimeMillis int64 `json:"createTimeMillis"`
}

// VaultSummaries returns summaries of vaults.
func (c *InfoClient) VaultSummaries(ctx context.Context) ([]VaultSummary, error) {
	return infoRequest[[]VaultSummary](ctx, c, "vaultSummaries", noParams{})
}

// UserVaultEquitiesRequest is the request for [InfoClient.UserVaultEquities].
type UserVaultEquitiesRequest struct {
	User Address `json:"user"`
}

// VaultEquity is a user's equity in a vault.
type VaultEquity struct {
	VaultAddress Address `json:"vaultAddress"`
	Equity       Decimal `json:"equity"`
	// LockedUntilTimestamp is in milliseconds since the Unix epoch.
	LockedUntilTimestamp int64 `json:"lockedUntilTimestamp"`
}

// UserVaultEquities returns the vaults a user has deposited into.
func (c *InfoClient) UserVaultEquities(ctx context.Context, req UserVaultEquitiesRequest) ([]VaultEquity, error) {
	return infoRequest[[]VaultEquity](ctx, c, "userVaultEquities", req)
}

// LeadingVaultsRequest is the request for [InfoClient.LeadingVaults].
type LeadingVaultsRequest struct {
	User Address `json:"user"`
}

// LeadingVault is a vault led by a user.
type LeadingVault struct {
	Address Address `json:"address"`
	Name    string  `json:"name"`
}

// LeadingVaults returns the vaults a user leads.
func (c *InfoClient) LeadingVaults(ctx context.Context, req LeadingVaultsRequest) ([]LeadingVault, error) {
	return infoRequest[[]LeadingVault](ctx, c, "leadingVaults", req)
}
