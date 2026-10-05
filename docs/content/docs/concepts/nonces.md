---
title: Nonces, vaults and expiry
weight: 4
---

Every signed action carries a nonce that the exchange uses to reject
replays. The SDK draws nonces for you; this page explains the rules so you can
lay out keys and processes that never collide, and covers the two client
settings that are signed into each action: the vault and the expiry.

## Hyperliquid's nonce rules

Hyperliquid does not use Ethereum's sequential nonces. Per the
[Hyperliquid docs](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/nonces-and-api-wallets):

- The exchange stores the 100 highest nonces of each **signer**. A new action
  must have a nonce larger than the smallest stored one, and must not have
  been used before.
- The nonce must lie within `(T - 2 days, T + 1 day)`, where `T` is the
  timestamp of the block that includes the action, in milliseconds.
- Nonces are tracked per signer: the account address when the account key
  signs, or the agent address when an API wallet signs. A single agent that
  signs for an account, its sub-accounts and vaults shares one nonce set.

## How the SDK draws nonces

Each action is signed with a nonce from
[`NextNonce`](/docs/reference#NextNonce): the current Unix
time in milliseconds, or one more than the last nonce issued in this process
if that is later. The counter is a single atomic value shared by every
`ExchangeClient` in the process, so concurrent goroutines, and separate
clients built for the same key, never reuse a nonce. Bursts of more than one
action per millisecond simply run the counter slightly ahead of the clock,
well inside the allowed window.

Two things are outside the SDK's control:

- **Several processes signing with the same key.** Each process has its own
  counter, and two processes can draw the same millisecond; one of the two
  actions is then rejected. Give every trading process its own
  [API wallet](/docs/concepts/signing#api-wallets). If you trade several
  sub-accounts in parallel, use one API wallet per sub-account as well.
- **The system clock.** Nonces are timestamps, so a clock that is days off
  produces nonces outside the window. Keep the host synchronized with NTP.

## Explicit nonces

A few methods take the nonce as an argument, because several parties must
sign or reference the same value:
[`SignMultiSig`](/docs/reference#ExchangeClient.SignMultiSig),
[`MultiSig`](/docs/reference#ExchangeClient.MultiSig) and
[`Noop`](/docs/reference#ExchangeClient.Noop). Draw these
from `NextNonce` too, so they never collide with the nonces the client draws
for other actions:

```go
nonce := hyperliquid.NextNonce()
sig, err := ex.SignMultiSig(ctx, action, multiSigUser, leader, nonce)
if err != nil {
	log.Fatal(err)
}
```

`Noop` submits an action that does nothing with the given nonce, which uses
the nonce up. Use it to invalidate an action you signed with that nonce but
no longer want executed.

## Vaults and sub-accounts

[`WithVault`](/docs/reference#ExchangeClient.WithVault)
returns a copy of the client that acts for a vault you lead or one of your
sub-accounts. The vault address is signed into each L1 action, so the
exchange applies the order, cancel or leverage change to that account:

```go
sub := hyperliquid.MustParseAddress("0x1234567890abcdef1234567890abcdef12345678")
subEx := ex.WithVault(sub)

_, err := subEx.Order(ctx, hyperliquid.OrderAction{Orders: []hyperliquid.Order{order}})
if err != nil {
	log.Fatal(err)
}
```

The copy is cheap and the original client is unchanged, so keep one client
per account or call `ex.WithVault(addr)` inline. The signer is still the
master account or its API wallet, so the nonce set is still the signer's.

`WithVault` applies to L1 actions only. User-signed actions, such as
transfers, always act for the signer and ignore the vault. Move funds to and
from a sub-account from the master account with
[`SubAccountTransfer`](/docs/reference#ExchangeClient.SubAccountTransfer)
or [`SubAccountSpotTransfer`](/docs/reference#ExchangeClient.SubAccountSpotTransfer),
and to or from a vault with
[`VaultTransfer`](/docs/reference#ExchangeClient.VaultTransfer).
Query a sub-account's or vault's state with its own address.

## Expiring actions

[`WithExpiresAfter`](/docs/reference#ExchangeClient.WithExpiresAfter)
returns a copy of the client whose L1 actions the exchange rejects if they
arrive after a given time. Use it so that an order delayed by a slow network
or a reconnect cannot execute at a stale price:

```go
results, err := ex.WithExpiresAfter(time.Now().Add(3*time.Second)).Order(ctx, action)
```

The expiry is an absolute time fixed when you call `WithExpiresAfter`, so
create the copy per action rather than once at startup. Like the vault, the
expiry applies to L1 actions only; user-signed actions do not support it.
Hyperliquid charges five times the usual address-based rate limit for an
action rejected because it expired, so leave a reasonable margin.

An expiry also makes ambiguous failures decidable: once it has passed, an
action the exchange has not seen can no longer execute (see
[Did the action execute?](/docs/concepts/errors#did-the-action-execute)).
