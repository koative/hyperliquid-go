---
title: Vaults and sub-accounts
weight: 6
---

Sub-accounts and vaults are separate addresses controlled by a master
account. Both are traded through
[`ExchangeClient.WithVault`](/docs/reference#ExchangeClient.WithVault),
and both are funded with integer micro-USDC transfers.

{{< callout type="info" >}}
`USD` and `InitialUSD` fields are USDC × 10⁶: `25_000_000` is 25 USDC.
Spot amounts (`SubAccountSpotTransferAction.Amount`) are decimal strings.
{{< /callout >}}

## Sub-accounts

### Create and fund

Hyperliquid unlocks sub-accounts after $100,000 of trading volume (see
[Sub-accounts](https://hyperliquid.gitbook.io/hyperliquid-docs/trading/sub-accounts)).
[`CreateSubAccount`](/docs/reference#ExchangeClient.CreateSubAccount)
returns the new address. Move perp USDC in or out with
[`SubAccountTransfer`](/docs/reference#ExchangeClient.SubAccountTransfer)
and spot tokens with
[`SubAccountSpotTransfer`](/docs/reference#ExchangeClient.SubAccountSpotTransfer):

```go
subAccount, err := ex.CreateSubAccount(ctx, hyperliquid.CreateSubAccountAction{Name: "mm-eth"})
if err != nil {
	log.Fatal(err)
}

// 250 USDC from the master's perp balance into the sub-account.
err = ex.SubAccountTransfer(ctx, hyperliquid.SubAccountTransferAction{
	SubAccountUser: subAccount,
	IsDeposit:      true,
	USD:            250_000_000,
})
if err != nil {
	log.Fatal(err)
}

// 10 PURR from the master's spot balance into the sub-account.
err = ex.SubAccountSpotTransfer(ctx, hyperliquid.SubAccountSpotTransferAction{
	SubAccountUser: subAccount,
	IsDeposit:      true,
	Token:          "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2", // Testnet
	Amount:         "10",
})
if err != nil {
	log.Fatal(err)
}
```

Set `IsDeposit: false` to move funds back to the master.
[`SubAccountModify`](/docs/reference#ExchangeClient.SubAccountModify)
renames a sub-account. For a sub-account's own spot/perp moves and
cross-dex transfers, use `USDClassTransferAction.SubAccount` and
`SendAssetAction.FromSubAccount` (see [Transfers](/docs/guides/transfers)).

### Trade as a sub-account

A sub-account has no key of its own. Sign with the master's key (or an API
wallet the master approved) and pass the sub-account to `WithVault`:

```go
subEx := ex.WithVault(subAccount)
results, err := subEx.Order(ctx, hyperliquid.OrderAction{Orders: []hyperliquid.Order{order}})
if err != nil {
	log.Fatal(err)
}
fmt.Println(results[0])
```

`WithVault` returns a copy, so `ex` keeps trading for the master. It applies
to L1 actions (orders, cancels, leverage, ...) only; user-signed actions
such as transfers always act for the signer.

Nonces are tracked per signing key, so one API wallet trading several
sub-accounts shares one nonce set across them. Hyperliquid recommends a
separate API wallet per sub-account (or per trading process).

### Read sub-account state

```go
subs, err := info.SubAccounts(ctx, hyperliquid.SubAccountsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, s := range subs {
	fmt.Println(s.Name, s.SubAccountUser, s.ClearinghouseState.MarginSummary.AccountValue)
}
```

[`SubAccounts2`](/docs/reference#InfoClient.SubAccounts2)
returns each sub-account's state on every perp dex instead of only the main
dex.

## Vaults

These actions manage HyperCore vaults, which Hyperliquid now calls
[legacy vaults](https://hyperliquid.gitbook.io/hyperliquid-docs/hypercore/vaults):
they trade main-dex perps only, without HIP-3 or spot. Newer vaults live on
the HyperEVM.

### Create and manage a vault

[`CreateVault`](/docs/reference#ExchangeClient.CreateVault)
makes the signer the vault's leader and returns its address. The name is 3
to 50 characters, the description 10 to 250, and the initial deposit at
least 100 USDC.

```go
vault, err := ex.CreateVault(ctx, hyperliquid.CreateVaultAction{
	Name:        "ETH basis",
	Description: "Delta-neutral ETH funding capture.",
	InitialUSD:  100_000_000,
})
if err != nil {
	log.Fatal(err)
}

// Trade the vault's funds.
vaultEx := ex.WithVault(vault)
_ = vaultEx // vaultEx.Order(...), vaultEx.Cancel(...)

// Stop accepting deposits. Nil fields stay unchanged.
closed := false
err = ex.VaultModify(ctx, hyperliquid.VaultModifyAction{
	VaultAddress:  vault,
	AllowDeposits: &closed,
})
if err != nil {
	log.Fatal(err)
}
```

[`VaultDistribute`](/docs/reference#ExchangeClient.VaultDistribute)
pays vault funds out to its followers; `USD: 0` closes the vault:

```go
err := ex.VaultDistribute(ctx, hyperliquid.VaultDistributeAction{
	VaultAddress: vault,
	USD:          1_000_000_000, // 1,000 USDC
})
if err != nil {
	log.Fatal(err)
}
```

### Deposit and withdraw

Any account can deposit into a vault that allows deposits, and withdraw
after the vault's lock-up:

```go
err := ex.VaultTransfer(ctx, hyperliquid.VaultTransferAction{
	VaultAddress: vault,
	IsDeposit:    true,
	USD:          50_000_000, // 50 USDC
})
if err != nil {
	log.Fatal(err)
}
```

### Read vault state

```go
hlp := hyperliquid.MustParseAddress("0xdfc24b077bc1425ad1dea75bcb6f8158e10df303")
details, err := info.VaultDetails(ctx, hyperliquid.VaultDetailsRequest{
	VaultAddress: hlp,
	User:         &user, // optional: fills FollowerState
})
if err != nil {
	log.Fatal(err)
}
if details == nil {
	log.Fatal("no such vault")
}
fmt.Println(details.Name, "leader", details.Leader, "APR", details.APR)
if f := details.FollowerState; f != nil {
	fmt.Println("my equity", f.VaultEquity, "locked until", time.UnixMilli(f.LockupUntil))
}

equities, err := info.UserVaultEquities(ctx, hyperliquid.UserVaultEquitiesRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, e := range equities {
	fmt.Println(e.VaultAddress, e.Equity)
}
```

[`VaultDetails`](/docs/reference#InfoClient.VaultDetails)
returns `nil` for an unknown address. `APR` is a fraction (`0.05` is 5%).
[`LeadingVaults`](/docs/reference#InfoClient.LeadingVaults)
lists the vaults a user leads, and
[`VaultSummaries`](/docs/reference#InfoClient.VaultSummaries)
lists vaults across the exchange.
