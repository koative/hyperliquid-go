---
title: Multi-sig
weight: 8
---

Hyperliquid has native multi-sig accounts: a set of authorized users and a
threshold of signatures every action needs. Once an account is converted,
all of its actions go through
[`ExchangeClient.MultiSig`](/docs/reference#ExchangeClient.MultiSig);
its own key can no longer act alone.

## Convert an account

The account to convert signs
[`ConvertToMultiSigUser`](/docs/reference#ExchangeClient.ConvertToMultiSigUser)
with its own key. Authorized users (at most 10) must be existing Hyperliquid
users; `Threshold` is the number of their signatures each action needs.

```go
err := accountEx.ConvertToMultiSigUser(ctx, hyperliquid.ConvertToMultiSigUserAction{
	Signers: &hyperliquid.MultiSigSigners{
		AuthorizedUsers: []hyperliquid.Address{aliceAddr, bobAddr, carolAddr},
		Threshold:       2,
	},
})
if err != nil {
	log.Fatal(err)
}
```

Read the current signer set with
[`UserToMultiSigSigners`](/docs/reference#InfoClient.UserToMultiSigSigners);
it returns `nil` for a normal account:

```go
signers, err := info.UserToMultiSigSigners(ctx, hyperliquid.UserToMultiSigSignersRequest{User: multiSigUser})
if err != nil {
	log.Fatal(err)
}
if signers == nil {
	fmt.Println("not a multi-sig account")
} else {
	fmt.Println(len(signers.AuthorizedUsers), "signers, threshold", signers.Threshold)
}
```

{{< callout type="warning" >}}
Conversion does not affect the account's HyperEVM side, which stays
controlled by the original key, and CoreWriter does not work for multi-sig
users. Hyperliquid recommends that multi-sig users do not use the HyperEVM.
{{< /callout >}}

## Send an action

One authorized user, the *leader*, submits the action. Every signature must
cover the same action, multi-sig account, leader address and nonce:

1. The leader picks a nonce with
   [`NextNonce`](/docs/reference#NextNonce) and shares it,
   with the action, with the other signers. Only the leader's nonce set is
   checked, so the nonce must come from the leader's side.
2. Each signer (the leader too, if its signature counts toward the
   threshold) calls
   [`SignMultiSig`](/docs/reference#ExchangeClient.SignMultiSig)
   with its own client.
3. The leader collects at least `Threshold` signatures and calls `MultiSig`.

```go
// Leader (alice): choose the action and nonce.
action := hyperliquid.USDSendAction{Destination: destination, Amount: "100"}
nonce := hyperliquid.NextNonce()
leaderAddr := alice.Signer().Address()

// Each signer, typically in its own process, with the same inputs.
aliceSig, err := alice.SignMultiSig(ctx, action, multiSigUser, leaderAddr, nonce)
if err != nil {
	log.Fatal(err)
}
bobSig, err := bob.SignMultiSig(ctx, action, multiSigUser, leaderAddr, nonce)
if err != nil {
	log.Fatal(err)
}

// Leader: submit.
_, err = alice.MultiSig(ctx, multiSigUser, action, nonce, []hyperliquid.Signature{aliceSig, bobSig})
if err != nil {
	log.Fatal(err)
}
```

[`Signature`](/docs/reference#Signature) marshals to and
from JSON, so signatures can travel between processes in whatever channel
you use to coordinate.

Any action works, user-signed (transfers, withdrawals) or L1 (orders,
cancels). `MultiSig` returns the inner action's raw response data; for
batch actions such as orders, rejected items also come back as joined
[`*StatusError`](/docs/reference#StatusError) values, like
`ExchangeClient.Order`:

```go
data, err := alice.MultiSig(ctx, multiSigUser, action, nonce, sigs)
if rejected, ok := errors.AsType[*hyperliquid.StatusError](err); ok {
	log.Printf("order %d rejected: %s", rejected.Index, rejected.Message)
} else if err != nil {
	log.Fatal(err)
}
var result struct {
	Statuses []hyperliquid.OrderResult `json:"statuses"`
}
if err := json.Unmarshal(data, &result); err != nil {
	log.Fatal(err)
}
```

{{< callout type="info" >}}
For L1 actions, the signers' and the leader's clients must use the same
[`WithVault`](/docs/reference#ExchangeClient.WithVault)
and [`WithExpiresAfter`](/docs/reference#ExchangeClient.WithExpiresAfter)
settings, and the same network: they are part of what is signed. The leader
may be an API wallet of an authorized user; pass that wallet's address as
the leader address when signing.
{{< /callout >}}

## Change signers or revert

Changing the signer set or threshold, and converting back to a normal
account, are themselves multi-sig actions: wrap a `ConvertToMultiSigUser`
in `MultiSig`. Nil `Signers` reverts to a normal account.

```go
revert := hyperliquid.ConvertToMultiSigUserAction{} // nil Signers
nonce := hyperliquid.NextNonce()
leaderAddr := alice.Signer().Address()

aliceSig, err := alice.SignMultiSig(ctx, revert, multiSigUser, leaderAddr, nonce)
if err != nil {
	log.Fatal(err)
}
bobSig, err := bob.SignMultiSig(ctx, revert, multiSigUser, leaderAddr, nonce)
if err != nil {
	log.Fatal(err)
}
if _, err := alice.MultiSig(ctx, multiSigUser, revert, nonce, []hyperliquid.Signature{aliceSig, bobSig}); err != nil {
	log.Fatal(err)
}
```

See Hyperliquid's [multi-sig docs](https://hyperliquid.gitbook.io/hyperliquid-docs/hypercore/multi-sig)
for the protocol rules.
