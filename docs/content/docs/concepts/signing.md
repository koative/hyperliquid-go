---
title: Signing and keys
weight: 1
---

Every action sent through an
[`ExchangeClient`](/docs/reference#ExchangeClient) is signed
by a [`Signer`](/docs/reference#Signer). Which key may sign
depends on the kind of action, so it is worth getting the key layout right
before writing any trading code.

## Signers

`Signer` is a two-method interface:

```go
type Signer interface {
	// Address returns the address whose key produces the signatures.
	Address() Address
	// SignHash signs a 32-byte EIP-712 digest and returns a recoverable
	// secp256k1 signature.
	SignHash(ctx context.Context, hash [32]byte) (Signature, error)
}
```

[`NewPrivateKeySigner`](/docs/reference#NewPrivateKeySigner)
covers keys held in memory. It takes a 64-character hex key, with or without
the `0x` prefix:

```go
signer, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_AGENT_KEY"))
if err != nil {
	log.Fatal(err)
}
fmt.Println("signing as", signer.Address())

ex := hyperliquid.NewExchangeClient(hyperliquid.Mainnet, signer)
```

The client computes the EIP-712 digest itself; a signer only ever sees a
32-byte hash. To keep keys in a KMS or HSM, implement `Signer` yourself
([see below](#signing-with-a-kms)).

## Networks

[`Mainnet`](/docs/reference#Mainnet) and
[`Testnet`](/docs/reference#Testnet) are
[`Network`](/docs/reference#Network) values holding the API,
WebSocket and explorer URLs of each network. `Network.Mainnet` also selects
the signature domain: a signature made for one network is rejected by the
other, so an order signed for Testnet can never execute on Mainnet.

Declare your own `Network` for other deployments, such as a gateway in front
of the public API. Keep `Mainnet` set to the network the gateway forwards to:

```go
gateway := hyperliquid.Mainnet
gateway.APIURL = "https://hl-gateway.internal.example.com"
gateway.WebSocketURL = "wss://hl-gateway.internal.example.com/ws"

info := hyperliquid.NewInfoClient(gateway)
ex := hyperliquid.NewExchangeClient(gateway, signer)
```

`RPCURL` is used only by the explorer requests
([`InfoClient.BlockDetails`](/docs/reference#InfoClient.BlockDetails),
`TxDetails`, `UserDetails`) and
[`DialExplorerWebSocket`](/docs/reference#DialExplorerWebSocket).

## L1 actions and user-signed actions

Hyperliquid signs actions in one of two ways, and the SDK picks the right one
for each action type:

- **L1 actions** are hashed together with the nonce, the vault address and
  the expiry, and the hash is signed through an EIP-712 "agent" message. Orders,
  cancels, modifies, leverage changes, TWAPs, vault and sub-account operations
  and the deployer actions are L1 actions. They may be signed by an
  [API wallet](#api-wallets), and they honor
  [`WithVault`](/docs/reference#ExchangeClient.WithVault) and
  [`WithExpiresAfter`](/docs/reference#ExchangeClient.WithExpiresAfter).
- **User-signed actions** are EIP-712 typed messages that a wallet can display
  field by field. They move funds or change who controls the account, so they
  **must be signed by the account's own key**. They always act for the signer:
  vault and expiry settings are not sent with them.

The user-signed actions are:

| Group | Methods |
|---|---|
| Transfers | [`USDSend`](/docs/reference#ExchangeClient.USDSend), [`SpotSend`](/docs/reference#ExchangeClient.SpotSend), [`Withdraw`](/docs/reference#ExchangeClient.Withdraw), [`USDClassTransfer`](/docs/reference#ExchangeClient.USDClassTransfer), [`SendAsset`](/docs/reference#ExchangeClient.SendAsset), [`SendToEVMWithData`](/docs/reference#ExchangeClient.SendToEVMWithData) |
| Approvals | [`ApproveAgent`](/docs/reference#ExchangeClient.ApproveAgent), [`ApproveBuilderFee`](/docs/reference#ExchangeClient.ApproveBuilderFee) |
| Staking | [`CDeposit`](/docs/reference#ExchangeClient.CDeposit), [`CWithdraw`](/docs/reference#ExchangeClient.CWithdraw), [`TokenDelegate`](/docs/reference#ExchangeClient.TokenDelegate), [`LinkStakingUser`](/docs/reference#ExchangeClient.LinkStakingUser), [`StakingLinkDisableTradingUser`](/docs/reference#ExchangeClient.StakingLinkDisableTradingUser) |
| Account | [`ConvertToMultiSigUser`](/docs/reference#ExchangeClient.ConvertToMultiSigUser), [`UserSetAbstraction`](/docs/reference#ExchangeClient.UserSetAbstraction), [`UserPortfolioMargin`](/docs/reference#ExchangeClient.UserPortfolioMargin) |

[`AgentSendAsset`](/docs/reference#ExchangeClient.AgentSendAsset)
and [`AgentSetAbstraction`](/docs/reference#ExchangeClient.AgentSetAbstraction)
are L1 counterparts of `SendAsset` and `UserSetAbstraction` that an API wallet
may sign.

## API wallets

An API wallet (also called an agent wallet) is a separate key that an account
authorizes to sign L1 actions for it. It can trade, but it cannot sign
user-signed actions, so it cannot withdraw or transfer funds. Run bots with an
API wallet and keep the account key offline.

You can create one in the Hyperliquid app under **More → API**, or from code
with the account key:

```go
// The account's own key approves the agent; it is needed only for this step.
owner, err := hyperliquid.NewPrivateKeySigner(os.Getenv("HL_ACCOUNT_KEY"))
if err != nil {
	log.Fatal(err)
}

// Generate a fresh agent key and store it in your secret manager.
var raw [32]byte
rand.Read(raw[:])
agentKey := hex.EncodeToString(raw[:])
agent, err := hyperliquid.NewPrivateKeySigner(agentKey)
if err != nil {
	log.Fatal(err)
}

err = hyperliquid.NewExchangeClient(hyperliquid.Testnet, owner).ApproveAgent(ctx, hyperliquid.ApproveAgentAction{
	AgentAddress: agent.Address(),
	AgentName:    "market-maker-1",
})
if err != nil {
	log.Fatal(err)
}

// From now on, trade with the agent key only.
ex := hyperliquid.NewExchangeClient(hyperliquid.Testnet, agent)
```

`AgentName` is at most 16 characters and may be followed by
`" valid_until <ms>"` to make the approval expire. An account has one unnamed
agent (empty name) and several named ones; approving a name again replaces
the agent that held it.
[`InfoClient.ExtraAgents`](/docs/reference#InfoClient.ExtraAgents)
lists the named agents of an account, and
[`InfoClient.UserRole`](/docs/reference#InfoClient.UserRole)
tells whether an address is an agent and for which account.

{{< callout type="warning" >}}
**Query with the account address, not the agent's.** An agent signs for the
account that approved it, so `ex.Signer().Address()` is the agent's address,
which holds no positions or orders. Pass the account (or sub-account, or
vault) address to every `InfoClient` and WebSocket request.
{{< /callout >}}

A few rules from the
[Hyperliquid docs](https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/nonces-and-api-wallets)
worth knowing:

- Nonces are tracked per signer, so an agent has its own nonce set. Use one
  agent per trading process (see [Nonces](/docs/concepts/nonces)).
- A replaced or expired agent, or one whose account has no funds left, may be
  pruned together with its nonce state. Generate a new agent key rather than
  re-approving an old one.
- An agent approved by the master account can also sign for its
  sub-accounts; select one with
  [`WithVault`](/docs/concepts/nonces#vaults-and-sub-accounts).

## Signing with a KMS

Implement `Signer` to sign with a key that never leaves a KMS, HSM or remote
signing service. `SignHash` must return a recoverable secp256k1 signature:
`R` and `S` plus `V` set to 27 or 28. Most KMS APIs return a DER-encoded
`(r, s)` pair instead, so the signer has to convert it, normalize `S` to the
lower half of the curve order (the canonical Ethereum form), and find the
recovery ID that yields its public key.

The example below uses the secp256k1 and Keccak packages that the SDK already
depends on. `sign` wraps the KMS call; with AWS KMS it is `Sign` with
`MessageType: DIGEST` and `SigningAlgorithm: ECDSA_SHA_256`, and the public key
comes from `GetPublicKey`.

```go
import (
	"context"
	"crypto/x509/pkix"
	"encoding/asn1"
	"errors"
	"math/big"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"github.com/koative/hyperliquid-go"
	"golang.org/x/crypto/sha3"
)

// KMSSigner signs with a secp256k1 key held in a KMS.
type KMSSigner struct {
	pub  *secp256k1.PublicKey
	addr hyperliquid.Address
	// sign returns the DER-encoded ECDSA signature of a 32-byte digest.
	sign func(ctx context.Context, digest []byte) ([]byte, error)
}

// NewKMSSigner returns a signer for the key whose public key is spki, a
// DER-encoded SubjectPublicKeyInfo such as AWS KMS GetPublicKey returns.
func NewKMSSigner(spki []byte, sign func(ctx context.Context, digest []byte) ([]byte, error)) (*KMSSigner, error) {
	var info struct {
		Algorithm pkix.AlgorithmIdentifier
		PublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(spki, &info); err != nil {
		return nil, err
	}
	pub, err := secp256k1.ParsePubKey(info.PublicKey.Bytes)
	if err != nil {
		return nil, err
	}
	// The address is the last 20 bytes of the Keccak-256 hash of the
	// uncompressed public key, without its 0x04 prefix.
	h := sha3.NewLegacyKeccak256()
	h.Write(pub.SerializeUncompressed()[1:])
	s := &KMSSigner{pub: pub, sign: sign}
	copy(s.addr[:], h.Sum(nil)[12:])
	return s, nil
}

func (s *KMSSigner) Address() hyperliquid.Address { return s.addr }

func (s *KMSSigner) SignHash(ctx context.Context, hash [32]byte) (hyperliquid.Signature, error) {
	der, err := s.sign(ctx, hash[:])
	if err != nil {
		return hyperliquid.Signature{}, err
	}
	var rs struct{ R, S *big.Int }
	if _, err := asn1.Unmarshal(der, &rs); err != nil {
		return hyperliquid.Signature{}, err
	}
	// Use the low-S form; a KMS may return either.
	n := secp256k1.S256().Params().N
	if rs.S.Cmp(new(big.Int).Rsh(n, 1)) > 0 {
		rs.S.Sub(n, rs.S)
	}
	var sig hyperliquid.Signature
	rs.R.FillBytes(sig.R[:])
	rs.S.FillBytes(sig.S[:])
	// Pick the recovery ID that recovers our own public key.
	compact := make([]byte, 65)
	copy(compact[1:33], sig.R[:])
	copy(compact[33:], sig.S[:])
	for _, v := range []byte{27, 28} {
		compact[0] = v
		if pub, _, err := ecdsa.RecoverCompact(compact, hash[:]); err == nil && pub.IsEqual(s.pub) {
			sig.V = v
			return sig, nil
		}
	}
	return hyperliquid.Signature{}, errors.New("kms: signature does not match the public key")
}
```

Pass it to `NewExchangeClient` like any other signer. `SignHash` receives the
request's context, so KMS calls are cancelled together with the request.
