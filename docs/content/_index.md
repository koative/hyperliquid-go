---
title: hyperliquid-go
layout: hextra-home
---

{{< hextra/hero-badge link="https://github.com/koative/hyperliquid-go" >}}
  <span>Open source · Apache-2.0</span>
  {{< icon name="arrow-circle-right" attributes="height=14" >}}
{{< /hextra/hero-badge >}}

<div class="hx:mt-6 hx:mb-6">
{{< hextra/hero-headline >}}
  The Go SDK for&nbsp;<br class="hx:sm:block hx:hidden" />Hyperliquid
{{< /hextra/hero-headline >}}
</div>

<div class="hx:mb-12">
{{< hextra/hero-subtitle >}}
  Market data, trading, transfers and real-time streams&nbsp;<br class="hx:sm:block hx:hidden" />with typed, lossless and verified signing.
{{< /hextra/hero-subtitle >}}
</div>

<div class="hx:mb-6">
{{< hextra/hero-button text="Get Started" link="docs/getting-started" >}}
{{< hextra/hero-button text="API Reference" link="docs/reference" style="background: transparent; color: inherit; border: 1px solid currentColor;" >}}
</div>

```sh
go get github.com/koative/hyperliquid-go
```

<div class="hx:mt-6"></div>

{{< hextra/feature-grid >}}
  {{< hextra/feature-card
    title="Complete API"
    icon="collection"
    subtitle="Every Info endpoint, every Exchange action and every WebSocket channel: trading, transfers, vaults, staking, multi-sig and HIP-1/3/4 deployer actions."
  >}}
  {{< hextra/feature-card
    title="Verified signing"
    icon="shield-check"
    subtitle="Signatures are tested byte for byte against the official Python SDK, and every action was accepted by the live exchange."
  >}}
  {{< hextra/feature-card
    title="Lossless numbers"
    icon="calculator"
    subtitle="Prices and sizes are decimal strings, exactly as Hyperliquid sends them. No float64 ever touches money on the wire."
  >}}
  {{< hextra/feature-card
    title="Resilient streaming"
    icon="lightning-bolt"
    subtitle="Typed subscriptions with keep-alive, automatic reconnect and re-subscribe, plus Info and Exchange requests over the socket."
  >}}
  {{< hextra/feature-card
    title="Idiomatic Go"
    icon="code"
    subtitle="context.Context everywhere, concurrency-safe clients, typed errors and a pluggable Signer for KMS and HSM keys."
  >}}
  {{< hextra/feature-card
    title="Small footprint"
    icon="cube"
    subtitle="No go-ethereum. Three dependencies: decred secp256k1, x/crypto and coder/websocket."
  >}}
{{< /hextra/feature-grid >}}
