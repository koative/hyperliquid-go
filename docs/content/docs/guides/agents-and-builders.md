---
title: Agents, builders and referrals
weight: 10
---

API (agent) wallets let a separate key trade for an account; builder codes
let an app charge a fee on the orders it sends; referral codes share fees.
Approvals are user-signed: the account's own key must sign them.

## API wallets

An API wallet signs L1 actions (orders, cancels, leverage, vault and
sub-account trading through `WithVault`) for the account that approved it.
It cannot sign user-signed actions, so it cannot transfer or withdraw funds.

### Approve an agent

Generate a fresh key and approve its address with
[`ApproveAgent`](/docs/reference#ExchangeClient.ApproveAgent),
signed by the account:

```go
var key [32]byte
rand.Read(key[:]) // crypto/rand
agent, err := hyperliquid.NewPrivateKeySigner(hex.EncodeToString(key[:]))
if err != nil {
	log.Fatal(err)
}

validUntil := time.Now().Add(30 * 24 * time.Hour).UnixMilli()
err = accountEx.ApproveAgent(ctx, hyperliquid.ApproveAgentAction{
	AgentAddress: agent.Address(),
	AgentName:    fmt.Sprintf("mm-eth valid_until %d", validUntil),
})
if err != nil {
	log.Fatal(err)
}

// Store key securely; trade with it from now on.
agentEx := hyperliquid.NewExchangeClient(hyperliquid.Testnet, agent)
_ = agentEx
```

`AgentName` selects the slot:

- `""` approves the account's single unnamed agent, replacing the previous
  one.
- A name (at most 16 characters) approves a named agent; approving the same
  name again replaces it.
- An optional `" valid_until <ms>"` suffix makes the approval expire at that
  Unix time in milliseconds.

[`ExtraAgents`](/docs/reference#InfoClient.ExtraAgents)
lists the named agents and their expiry:

```go
agents, err := info.ExtraAgents(ctx, hyperliquid.ExtraAgentsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, a := range agents {
	if a.ValidUntil != nil {
		fmt.Println(a.Name, a.Address, "until", time.UnixMilli(*a.ValidUntil))
	} else {
		fmt.Println(a.Name, a.Address)
	}
}
```

### Agent gotchas

- **Query the account, not the agent.** Info requests take the account's
  address (or the sub-account's); the agent's address has no state.
- **Nonces are per signer.** The SDK's nonces are unique within a process,
  and the exchange keeps one nonce set per signing key. Give each trading
  process (and, per Hyperliquid, each sub-account traded in parallel) its
  own agent.
- **Do not reuse agent keys.** An agent that is replaced, expires, or whose
  account has no funds may be pruned along with its nonce state, after
  which old signed actions could be replayed. Generate a new key instead.
- An account starts with 3 API wallet slots, plus 2 per sub-account.

See [Nonces, vaults and expiry](/docs/concepts/nonces) and
Hyperliquid's [Nonces and API wallets](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/nonces-and-api-wallets).

## Builder codes

A builder charges a fee on orders it submits for a user, up to a maximum the
user approved. The user approves with
[`ApproveBuilderFee`](/docs/reference#ExchangeClient.ApproveBuilderFee),
signed by the account's own key. `MaxFeeRate` is a percentage of notional:

```go
err := accountEx.ApproveBuilderFee(ctx, hyperliquid.ApproveBuilderFeeAction{
	Builder:    builder,
	MaxFeeRate: "0.05", // 0.05%
})
if err != nil {
	log.Fatal(err)
}
```

The app then names itself in
[`OrderAction.Builder`](/docs/reference#OrderAction). The
`Fee` there is in tenths of a basis point, so `10` is 0.01%:

```go
results, err := ex.Order(ctx, hyperliquid.OrderAction{
	Orders:  []hyperliquid.Order{order},
	Builder: &hyperliquid.Builder{Address: builder, Fee: 50}, // 0.05%
})
if err != nil {
	log.Fatal(err)
}
fmt.Println(results[0])
```

Check an approval before attaching a fee:

```go
maxFee, err := info.MaxBuilderFee(ctx, hyperliquid.MaxBuilderFeeRequest{User: user, Builder: builder})
if err != nil {
	log.Fatal(err)
}
fmt.Println("max fee:", maxFee, "tenths of a basis point")

builders, err := info.ApprovedBuilders(ctx, hyperliquid.ApprovedBuildersRequest{User: user})
if err != nil {
	log.Fatal(err)
}
fmt.Println("approved builders:", builders)
```

Per [Hyperliquid's builder code rules](https://hyperliquid.gitbook.io/hyperliquid-docs/trading/builder-codes),
fees are capped at 0.1% on perps and 1% on spot, a user can have at most 10
active approvals, and the builder needs at least 100 USDC of perp account
value. Builders collect their fees with `ClaimRewards` (below).

## Referrals

```go
// Use someone's referral code.
err := ex.SetReferrer(ctx, hyperliquid.SetReferrerAction{Code: "FRIEND"})
if err != nil {
	log.Fatal(err)
}

// Create your own code, once eligible.
err = ex.RegisterReferrer(ctx, hyperliquid.RegisterReferrerAction{Code: "MYCODE"})
if err != nil {
	log.Fatal(err)
}
```

[`Referral`](/docs/reference#InfoClient.Referral) reports
who referred the user, the user's progress toward a code
(`ReferrerState.Stage` is `"needToTrade"`, `"needToCreateCode"` or
`"ready"`), and unclaimed referral and builder rewards.
[`ClaimRewards`](/docs/reference#ExchangeClient.ClaimRewards)
claims them:

```go
ref, err := info.Referral(ctx, hyperliquid.ReferralRequest{User: user})
if err != nil {
	log.Fatal(err)
}
if ref.ReferredBy != nil {
	fmt.Println("referred by", ref.ReferredBy.Code)
}
fmt.Println("stage", ref.ReferrerState.Stage, "unclaimed", ref.UnclaimedRewards, "builder", ref.BuilderRewards)

if !ref.UnclaimedRewards.IsZero() {
	if err := ex.ClaimRewards(ctx); err != nil {
		log.Fatal(err)
	}
}
```
