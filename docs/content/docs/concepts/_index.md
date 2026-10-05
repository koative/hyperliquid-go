---
title: Concepts
weight: 3
sidebar:
  open: true
---

Hyperliquid has a few rules that every integration runs into: actions are
signed, markets are addressed by numeric asset IDs, prices must fit a tick
grid, and every action carries a nonce. These pages explain how the SDK maps
them.

{{< cards >}}
  {{< card link="signing" title="Signing and keys" icon="key" subtitle="Signers, API wallets, L1 vs user-signed actions, KMS keys and networks." >}}
  {{< card link="markets" title="Markets and numbers" icon="chart-bar" subtitle="Asset IDs, spot and builder-dex names, Decimal, tick and lot sizes." >}}
  {{< card link="errors" title="Errors" icon="exclamation-circle" subtitle="APIError, per-item StatusError, WebSocket errors and ambiguous outcomes." >}}
  {{< card link="nonces" title="Nonces, vaults and expiry" icon="clock" subtitle="How nonces are drawn, trading for vaults and sub-accounts, expiring actions." >}}
{{< /cards >}}
