package hyperliquid

import (
	"context"
	"encoding/hex"
	"encoding/json"
)

// UsdSendAction sends USDC from the perp balance to another address on
// Hyperliquid. It must be signed by the account's own key.
type UsdSendAction struct {
	Destination Address `json:"destination"`
	Amount      Decimal `json:"amount"`
}

func (UsdSendAction) actionType() string { return "usdSend" }

var usdSendSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UsdSend",
	Fields:      []typedField{{"destination", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (UsdSendAction) userSignedSpec() *userSignedSpec { return &usdSendSpec }

// UsdSend sends USDC to another address.
func (c *ExchangeClient) UsdSend(ctx context.Context, a UsdSendAction) error {
	return c.do(ctx, a, nil)
}

// SpotSendAction sends a spot token to another address on Hyperliquid. It
// must be signed by the account's own key.
type SpotSendAction struct {
	Destination Address `json:"destination"`
	// Token is the token's "NAME:0x<tokenId>" identifier, e.g.
	// "PURR:0xc4bf3f870c0e9465323c0b6ed28096c2".
	Token  string  `json:"token"`
	Amount Decimal `json:"amount"`
}

func (SpotSendAction) actionType() string { return "spotSend" }

var spotSendSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:SpotSend",
	Fields:      []typedField{{"destination", "string"}, {"token", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (SpotSendAction) userSignedSpec() *userSignedSpec { return &spotSendSpec }

// SpotSend sends a spot token to another address.
func (c *ExchangeClient) SpotSend(ctx context.Context, a SpotSendAction) error {
	return c.do(ctx, a, nil)
}

// Withdraw3Action withdraws USDC from the perp balance to Arbitrum through
// the bridge; a fixed fee is deducted. It must be signed by the account's
// own key.
type Withdraw3Action struct {
	Destination Address `json:"destination"`
	Amount      Decimal `json:"amount"`
}

func (Withdraw3Action) actionType() string { return "withdraw3" }

var withdraw3Spec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:Withdraw",
	Fields:      []typedField{{"destination", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (Withdraw3Action) userSignedSpec() *userSignedSpec { return &withdraw3Spec }

// Withdraw3 withdraws USDC to Arbitrum.
func (c *ExchangeClient) Withdraw3(ctx context.Context, a Withdraw3Action) error {
	return c.do(ctx, a, nil)
}

// UsdClassTransferAction moves USDC between the spot and perp balances of
// the signer or, when SubAccount is set, of one of its sub-accounts. It must
// be signed by the account's own key.
type UsdClassTransferAction struct {
	Amount Decimal
	// ToPerp moves spot to perp when true, perp to spot otherwise.
	ToPerp bool
	// SubAccount optionally selects a sub-account of the signer.
	SubAccount *Address
}

func (UsdClassTransferAction) actionType() string { return "usdClassTransfer" }

// MarshalJSON encodes a in wire form, where a sub-account is appended to the
// amount as "<amount> subaccount:<address>".
func (a UsdClassTransferAction) MarshalJSON() ([]byte, error) {
	amount, err := ParseDecimal(string(a.Amount))
	if err != nil {
		return nil, err
	}
	s := string(amount)
	if a.SubAccount != nil {
		s += " subaccount:" + a.SubAccount.String()
	}
	return json.Marshal(struct {
		Amount string `json:"amount"`
		ToPerp bool   `json:"toPerp"`
	}{s, a.ToPerp})
}

var usdClassTransferSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UsdClassTransfer",
	Fields:      []typedField{{"amount", "string"}, {"toPerp", "bool"}},
	NonceField:  "nonce",
}

func (UsdClassTransferAction) userSignedSpec() *userSignedSpec { return &usdClassTransferSpec }

// UsdClassTransfer moves USDC between the spot and perp balances.
func (c *ExchangeClient) UsdClassTransfer(ctx context.Context, a UsdClassTransferAction) error {
	return c.do(ctx, a, nil)
}

// SendAssetAction moves a token between addresses and between the balances
// of perp dexes and spot. It must be signed by the account's own key.
type SendAssetAction struct {
	Destination Address `json:"destination"`
	// SourceDex and DestinationDex name the balances: "" is the default perp
	// dex, "spot" is spot, anything else a builder-deployed perp dex.
	SourceDex      string `json:"sourceDex"`
	DestinationDex string `json:"destinationDex"`
	// Token is the token's "NAME:0x<tokenId>" identifier. It must be the
	// collateral token when a perp dex is involved.
	Token  string  `json:"token"`
	Amount Decimal `json:"amount"`
	// FromSubAccount optionally sends from a sub-account of the signer.
	FromSubAccount *Address `json:"fromSubAccount"`
}

func (SendAssetAction) actionType() string { return "sendAsset" }

// MarshalJSON encodes a in wire form, where the main account is
// fromSubAccount "".
func (a SendAssetAction) MarshalJSON() ([]byte, error) {
	type plain SendAssetAction
	from := ""
	if a.FromSubAccount != nil {
		from = a.FromSubAccount.String()
	}
	// The outer field shadows plain's and, being declared after it, is
	// encoded last, as on the wire.
	return json.Marshal(struct {
		plain
		FromSubAccount string `json:"fromSubAccount"`
	}{plain(a), from})
}

var sendAssetSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:SendAsset",
	Fields: []typedField{
		{"destination", "string"},
		{"sourceDex", "string"},
		{"destinationDex", "string"},
		{"token", "string"},
		{"amount", "string"},
		{"fromSubAccount", "string"},
	},
	NonceField: "nonce",
}

func (SendAssetAction) userSignedSpec() *userSignedSpec { return &sendAssetSpec }

// SendAsset moves a token between addresses, perp dexes and spot.
func (c *ExchangeClient) SendAsset(ctx context.Context, a SendAssetAction) error {
	return c.do(ctx, a, nil)
}

// AddressEncoding is the text encoding of a recipient address on another
// chain.
type AddressEncoding string

// Address encodings.
const (
	AddressEncodingHex    AddressEncoding = "hex"
	AddressEncodingBase58 AddressEncoding = "base58"
)

// SendToEvmWithDataAction sends a token from HyperCore to a contract
// implementing ICoreReceiveWithData, passing it Data. It must be signed by
// the account's own key.
type SendToEvmWithDataAction struct {
	// Token is the token name, e.g. "USDC".
	Token     string  `json:"token"`
	Amount    Decimal `json:"amount"`
	SourceDex string  `json:"sourceDex"`
	// DestinationRecipient is the recipient address in AddressEncoding.
	DestinationRecipient string          `json:"destinationRecipient"`
	AddressEncoding      AddressEncoding `json:"addressEncoding"`
	DestinationChainID   uint32          `json:"destinationChainId"`
	GasLimit             uint64          `json:"gasLimit"`
	// Data is the payload passed to the receiver; it may be empty.
	Data []byte `json:"data"`
}

func (SendToEvmWithDataAction) actionType() string { return "sendToEvmWithData" }

// MarshalJSON encodes a in wire form, where Data is 0x-prefixed hex.
func (a SendToEvmWithDataAction) MarshalJSON() ([]byte, error) {
	type plain SendToEvmWithDataAction
	// The outer field shadows plain's and, being declared after it, is
	// encoded last, as on the wire.
	return json.Marshal(struct {
		plain
		Data string `json:"data"`
	}{plain(a), "0x" + hex.EncodeToString(a.Data)})
}

var sendToEvmWithDataSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:SendToEvmWithData",
	Fields: []typedField{
		{"token", "string"},
		{"amount", "string"},
		{"sourceDex", "string"},
		{"destinationRecipient", "string"},
		{"addressEncoding", "string"},
		{"destinationChainId", "uint32"},
		{"gasLimit", "uint64"},
		{"data", "bytes"},
	},
	NonceField: "nonce",
}

func (SendToEvmWithDataAction) userSignedSpec() *userSignedSpec { return &sendToEvmWithDataSpec }

// SendToEvmWithData sends a token to a contract with a data payload.
func (c *ExchangeClient) SendToEvmWithData(ctx context.Context, a SendToEvmWithDataAction) error {
	return c.do(ctx, a, nil)
}
