# Changelog

## [0.4.0](https://github.com/koative/hyperliquid-go/compare/v0.3.0...v0.4.0) (2026-10-28)


### Features

* add Asset.MarketOrder, builder fee splits and AgentCDeposit ([#24](https://github.com/koative/hyperliquid-go/issues/24)) ([778be7d](https://github.com/koative/hyperliquid-go/commit/778be7ddc80771b130564e946f6c429601ec0ab0))


### Bug Fixes

* fail WebSocket subscriptions the server drops and bound WebSocket requests ([#19](https://github.com/koative/hyperliquid-go/issues/19)) ([8ab367d](https://github.com/koative/hyperliquid-go/commit/8ab367debcf1882ecfb5e9506baa6a38e42fbd0c))
* send zero StartTime, harden FormatSize, WithHTTPClient and hex errors ([#20](https://github.com/koative/hyperliquid-go/issues/20)) ([418e66a](https://github.com/koative/hyperliquid-go/commit/418e66aba80a376f1d12e0dc14e3fcda3903060a))
* validate orders before signing and surface multi-sig status errors ([#21](https://github.com/koative/hyperliquid-go/issues/21)) ([d1a0f8e](https://github.com/koative/hyperliquid-go/commit/d1a0f8e36ccac4ea07b7c6156239e85d6d4f25be))
* write WebSocket subscribe and unsubscribe frames in order ([#23](https://github.com/koative/hyperliquid-go/issues/23)) ([ef56be2](https://github.com/koative/hyperliquid-go/commit/ef56be26783d3387520c983e218d7e49eddef6d7))

## [0.3.0](https://github.com/koative/hyperliquid-go/compare/v0.2.0...v0.3.0) (2026-10-26)


### ⚠ BREAKING CHANGES

* remove InfoClient.Liquidatable ([#15](https://github.com/koative/hyperliquid-go/issues/15))

### Bug Fixes

* do not send WebSocket requests whose context is already done ([#14](https://github.com/koative/hyperliquid-go/issues/14)) ([cce6ac8](https://github.com/koative/hyperliquid-go/commit/cce6ac81ad8e875bb88e92110d0a41ebfc9e910e))
* remove InfoClient.Liquidatable ([#15](https://github.com/koative/hyperliquid-go/issues/15)) ([f197869](https://github.com/koative/hyperliquid-go/commit/f19786974d2c981a04d2c556be8b858379c2632e))

## [0.2.0](https://github.com/koative/hyperliquid-go/compare/v0.1.0...v0.2.0) (2026-10-05)


### ⚠ BREAKING CHANGES

* decode outcomeMetaUpdates messages as a list of updates ([#4](https://github.com/koative/hyperliquid-go/issues/4))

### Bug Fixes

* decode outcomeMetaUpdates messages as a list of updates ([#4](https://github.com/koative/hyperliquid-go/issues/4)) ([7071cae](https://github.com/koative/hyperliquid-go/commit/7071cae853d84562d43c9ac9ff6183424b945d21))

## 0.1.0 (2026-10-05)


### Features

* complete info, exchange and websocket API surface ([1189a75](https://github.com/koative/hyperliquid-go/commit/1189a758e945eb3445d35abb6db8acef96db1182))
* core signing, transport and client foundation ([7e3f88f](https://github.com/koative/hyperliquid-go/commit/7e3f88fa32099c5310c439c15f139cb438bcaa1e))


### Bug Fixes

* report per-order failures from Modify ([1be1779](https://github.com/koative/hyperliquid-go/commit/1be1779f0a9b3bb027fb1ce97119a08701bab4d4))
