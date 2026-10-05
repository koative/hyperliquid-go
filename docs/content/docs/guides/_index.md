---
title: Guides
weight: 4
sidebar:
  open: true
---

Task-oriented walkthroughs. Snippets assume `ctx`, an
[`InfoClient`](/docs/reference#InfoClient) named `info` and,
for actions, an
[`ExchangeClient`](/docs/reference#ExchangeClient) named `ex`,
created as in [Getting started](/docs/getting-started).

{{< cards >}}
  {{< card link="market-data" title="Market data" icon="chart-bar" subtitle="Mids, metadata, order books, candles, trades, funding and the explorer." >}}
  {{< card link="account" title="Account state" icon="user-circle" subtitle="Positions, balances, open orders, fills, funding, fees and rate limits." >}}
  {{< card link="orders" title="Orders" icon="switch-horizontal" subtitle="Limit, IOC, TP/SL, TWAP and trailing orders; modify, cancel, leverage." >}}
  {{< card link="websocket" title="WebSocket" icon="lightning-bolt" subtitle="Subscriptions, reconnects and requests over the socket." >}}
  {{< card link="transfers" title="Transfers" icon="currency-dollar" subtitle="USDC and spot transfers, withdrawals, moving funds between dexes." >}}
  {{< card link="vaults-and-subaccounts" title="Vaults and sub-accounts" icon="collection" subtitle="Create, fund and trade vaults and sub-accounts." >}}
  {{< card link="staking" title="Staking" icon="lock-closed" subtitle="Deposit, delegate and withdraw HYPE." >}}
  {{< card link="multisig" title="Multi-sig" icon="user-group" subtitle="Convert accounts, collect signatures and submit multi-sig actions." >}}
  {{< card link="deployers" title="Deployers" icon="cube" subtitle="HIP-1 spot tokens, HIP-3 perp dexes and HIP-4 outcomes." >}}
  {{< card link="agents-and-builders" title="Agents, builders and referrals" icon="key" subtitle="API wallets, builder fees and referral codes." >}}
{{< /cards >}}
