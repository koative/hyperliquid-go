---
title: Deployers
weight: 9
---

Actions for deploying assets: HIP-1 spot tokens, HIP-3 builder perp dexes
and HIP-4 outcome markets, plus the validator actions. These are L1 actions
signed by the deployer. Each action struct is a union: set exactly one
field per call, and the exchange enforces the rules described in the
Hyperliquid docs linked in each section.

## Read deployment state

```go
// Perp dexes. Element 0 is nil and stands for the main dex.
dexs, err := info.PerpDexs(ctx)
if err != nil {
	log.Fatal(err)
}
for _, d := range dexs[1:] {
	fmt.Println(d.Name, d.FullName, "deployer", d.Deployer)
}

// Open interest and transfer limits of one builder dex (nil for the main dex).
limits, err := info.PerpDexLimits(ctx, hyperliquid.PerpDexLimitsRequest{Dex: "xyz"})
if err != nil {
	log.Fatal(err)
}
fmt.Println("total OI cap", limits.TotalOICap)

// The Dutch auction for the next HIP-3 asset.
auction, err := info.PerpDeployAuctionStatus(ctx)
if err != nil {
	log.Fatal(err)
}
if auction.CurrentGas != nil {
	fmt.Println("current price", *auction.CurrentGas)
}

// A deployer's in-progress HIP-1 tokens and the token auction.
state, err := info.SpotDeployState(ctx, hyperliquid.SpotDeployStateRequest{User: user})
if err != nil {
	log.Fatal(err)
}
for _, t := range state.States {
	fmt.Println(t.Token, t.Spec.Name, "max supply", t.MaxSupply)
}
```

[`SpotPairDeployAuctionStatus`](/docs/reference#InfoClient.SpotPairDeployAuctionStatus)
reports the auction for registering a spot pair.

## HIP-1 spot tokens

[`SpotDeploy`](/docs/reference#ExchangeClient.SpotDeploy)
takes a [`SpotDeployAction`](/docs/reference#SpotDeployAction).
A token goes through these steps, one action each:

| Field | Step |
|---|---|
| `RegisterToken2` | Register the token, bidding in the deploy auction |
| `UserGenesis` | Assign genesis balances (repeatable) |
| `Genesis` | Finalize the supply |
| `RegisterSpot` | Register a trading pair |
| `RegisterHyperliquidity` | Seed the pair with the HIP-2 market maker |
| `SetDeployerTradingFeeShare` | Optional: the deployer's share of trading fees |

Other fields manage a live token: quote-token status, freeze privileges,
EVM contract links, annotations and the deployer label.

```go
err := ex.SpotDeploy(ctx, hyperliquid.SpotDeployAction{
	RegisterToken2: &hyperliquid.RegisterToken2{
		Spec:     hyperliquid.TokenSpec{Name: "TEST", SzDecimals: 2, WeiDecimals: 8},
		MaxGas:   500 * 100_000_000, // bid at most 500 HYPE (wei, 1e-8)
		FullName: "Test token",
	},
})
if err != nil {
	log.Fatal(err)
}
```

The new token's index, used by the later steps, appears in
`SpotDeployState`. See
[Deploying HIP-1 and HIP-2 assets](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/deploying-hip-1-and-hip-2-assets).

## HIP-3 perp dexes

[`PerpDeploy`](/docs/reference#ExchangeClient.PerpDeploy)
takes a [`PerpDeployAction`](/docs/reference#PerpDeployAction).
`RegisterAsset2` registers an asset; with `Schema` set it registers the dex
too, so the first call creates the dex:

```go
err := ex.PerpDeploy(ctx, hyperliquid.PerpDeployAction{
	RegisterAsset2: &hyperliquid.RegisterPerpAsset2{
		Dex: "mydex",
		AssetRequest: hyperliquid.PerpAssetRequest2{
			Coin:          "mydex:ABC",
			SzDecimals:    2,
			OraclePx:      "10.5",
			MarginTableID: 10,
			MarginMode:    hyperliquid.MarginModeNormal,
		},
		Schema: &hyperliquid.PerpDexSchema{
			FullName:        "My Dex",
			CollateralToken: 0, // USDC
		},
		// MaxGas nil pays the current auction price.
	},
})
if err != nil {
	log.Fatal(err)
}
```

The dex is then operated with the other fields, such as `SetOracle` (oracle
and mark prices, typically on a timer), the funding settings, margin tables,
open interest caps, `HaltTrading` and `SetSubDeployers` (delegating
variants to other addresses). Map-valued fields are
[`TupleMap`](/docs/reference#TupleMap)s keyed by full coin
name (`"mydex:ABC"`); the SDK sorts them as the exchange requires. See
[HIP-3 deployer actions](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-3-deployer-actions).

## HIP-4 outcomes

An outcome deployer first claims a venue with
[`ActivateOutcomeDeployer`](/docs/reference#ExchangeClient.ActivateOutcomeDeployer),
then registers and settles outcomes on it with
[`OutcomeDeploy`](/docs/reference#ExchangeClient.OutcomeDeploy).
Outcomes are instantiated from templates listed by
[`OutcomeTemplates`](/docs/reference#InfoClient.OutcomeTemplates);
each template's `Keywords` names the values to supply.

```go
err := ex.ActivateOutcomeDeployer(ctx, hyperliquid.ActivateOutcomeDeployerAction{
	Activate: &hyperliquid.OutcomeDeployerActivation{VenueName: "abc"},
})
if err != nil {
	log.Fatal(err)
}

err = ex.OutcomeDeploy(ctx, hyperliquid.OutcomeDeployAction{
	Venue: "abc",
	Operation: hyperliquid.OutcomeOperation{
		RegisterStandaloneOutcomeFromTemplate: &hyperliquid.OutcomeTemplateInstance{
			ID:               "aiModelHeadToHead",
			KeywordToValue:   values, // one entry per key of the template's Keywords
			DeployerFeeScale: "1",
		},
	},
})
if err != nil {
	log.Fatal(err)
}
```

`SettleOutcome` and `SettleQuestion2` settle markets; `SetSubDeployers`
delegates operations. See
[HIP-4 deployer actions](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-4-deployer-actions).

## Validators

Validator operators register and manage their validator with
[`CValidator`](/docs/reference#ExchangeClient.CValidator)
(`Register`, `ChangeProfile`, `Unregister`), and jail or unjail it with
[`CSigner`](/docs/reference#ExchangeClient.CSigner),
signed by the validator's signer key:

```go
commission := 300 // 3%
err := validatorEx.CValidator(ctx, hyperliquid.CValidatorAction{
	ChangeProfile: &hyperliquid.ValidatorProfileChange{CommissionBps: &commission},
})
if err != nil {
	log.Fatal(err)
}

err = signerEx.CSigner(ctx, hyperliquid.CSignerAction{UnjailSelf: true})
if err != nil {
	log.Fatal(err)
}
```

`ValidatorSummaries` and `ValidatorL1Votes` on `InfoClient` read the
validator set.
