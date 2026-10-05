---
title: Staking
weight: 7
---

HYPE staking happens on HyperCore: move HYPE from spot into your staking
balance, delegate it to validators, and move it back when you are done.
Every staking action is user-signed, so the client must sign with the
account's own key, not an API wallet.

{{< callout type="info" >}}
Staking amounts are integer `Wei` fields in units of 10⁻⁸ HYPE:
`100_000_000` is 1 HYPE. Info results report amounts as decimal HYPE
strings.
{{< /callout >}}

## Stake and delegate

```go
const hype = 100_000_000 // 1 HYPE in wei

// Spot -> staking balance.
err := ex.CDeposit(ctx, hyperliquid.CDepositAction{Wei: 10 * hype})
if err != nil {
	log.Fatal(err)
}

// Delegate to a validator.
err = ex.TokenDelegate(ctx, hyperliquid.TokenDelegateAction{
	Validator: validator,
	Wei:       10 * hype,
})
if err != nil {
	log.Fatal(err)
}
```

[`CDeposit`](/docs/reference#ExchangeClient.CDeposit) is
instant. Pick a validator from
[`ValidatorSummaries`](/docs/reference#InfoClient.ValidatorSummaries):

```go
validators, err := info.ValidatorSummaries(ctx)
if err != nil {
	log.Fatal(err)
}
for _, v := range validators {
	if v.IsActive && !v.IsJailed {
		fmt.Println(v.Name, v.Validator, "commission", v.Commission)
	}
}
```

## Undelegate and unstake

```go
// Undelegate back to the staking balance.
err := ex.TokenDelegate(ctx, hyperliquid.TokenDelegateAction{
	Validator:    validator,
	Wei:          5 * hype,
	IsUndelegate: true,
})
if err != nil {
	log.Fatal(err)
}

// Staking balance -> spot, through the unstaking queue.
err = ex.CWithdraw(ctx, hyperliquid.CWithdrawAction{Wei: 5 * hype})
if err != nil {
	log.Fatal(err)
}
```

Per the [Hyperliquid staking docs](https://hyperliquid.gitbook.io/hyperliquid-docs/hypercore/staking):
a delegation is locked for one day, [`CWithdraw`](/docs/reference#ExchangeClient.CWithdraw)
only moves undelegated HYPE and finalizes after a 7-day queue, and an
address can have at most 5 pending withdrawals. Rewards are redelegated to
the validator automatically; there is no claim step.
([`ClaimRewards`](/docs/reference#ExchangeClient.ClaimRewards)
claims referral and builder rewards, not staking rewards.)

## Read staking state

```go
summary, err := info.DelegatorSummary(ctx, hyperliquid.DelegatorSummaryRequest{User: user})
if err != nil {
	log.Fatal(err)
}
fmt.Println("delegated", summary.Delegated, "undelegated", summary.Undelegated,
	"pending withdrawal", summary.TotalPendingWithdrawal)

delegations, err := info.Delegations(ctx, hyperliquid.DelegationsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, d := range delegations {
	fmt.Println(d.Validator, d.Amount, "locked until", time.UnixMilli(d.LockedUntilTimestamp))
}

rewards, err := info.DelegatorRewards(ctx, hyperliquid.DelegatorRewardsRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, r := range rewards {
	fmt.Println(time.UnixMilli(r.Time), r.Source, r.TotalAmount)
}
```

[`DelegatorHistory`](/docs/reference#InfoClient.DelegatorHistory)
lists every deposit, delegation and withdrawal. Each event's `Delta` has
exactly one field set:

```go
history, err := info.DelegatorHistory(ctx, hyperliquid.DelegatorHistoryRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, e := range history {
	switch d := e.Delta; {
	case d.Delegate != nil:
		fmt.Println("delegate", d.Delegate.Validator, d.Delegate.Amount, "undelegate:", d.Delegate.IsUndelegate)
	case d.CDeposit != nil:
		fmt.Println("deposit", d.CDeposit.Amount)
	case d.Withdrawal != nil:
		fmt.Println("withdrawal", d.Withdrawal.Amount, d.Withdrawal.Phase)
	}
}
```

## Link a trading account to a staking account

[`LinkStakingUser`](/docs/reference#ExchangeClient.LinkStakingUser)
gives a trading account the fee discount of a separate staking account. It
takes two actions, each signed by its own account's key: the trading
account requests the link, then the staking account finalizes it.

```go
// Signed by the trading account.
err := tradingEx.LinkStakingUser(ctx, hyperliquid.LinkStakingUserAction{User: stakingUser})
if err != nil {
	log.Fatal(err)
}

// Signed by the staking account.
err = stakingEx.LinkStakingUser(ctx, hyperliquid.LinkStakingUserAction{
	User:       tradingUser,
	IsFinalize: true,
})
if err != nil {
	log.Fatal(err)
}
```

`InfoClient.UserFees` reports the link in `StakingLink`.

{{< callout type="error" >}}
[`StakingLinkDisableTradingUser`](/docs/reference#ExchangeClient.StakingLinkDisableTradingUser)
permanently disables a linked trading account and locks its funds, which
move to the staking account after one year. It cannot be undone.
{{< /callout >}}
