package hyperliquid

import (
	"context"
	"encoding/hex"
	"encoding/json"
)

// USDSendAction sends USDC from the perp balance to another address on
// Hyperliquid. It must be signed by the account's own key.
type USDSendAction struct {
	Destination Address `json:"destination"`
	Amount      Decimal `json:"amount"`
}

func (USDSendAction) actionType() string { return "usdSend" }

var usdSendSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:UsdSend",
	Fields:      []typedField{{"destination", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (USDSendAction) userSignedSpec() *userSignedSpec { return &usdSendSpec }

// USDSend sends USDC to another address.
func (c *ExchangeClient) USDSend(ctx context.Context, a USDSendAction) error {
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

// WithdrawAction withdraws USDC from the perp balance to Arbitrum through
// the bridge; a fixed fee is deducted. It must be signed by the account's
// own key.
type WithdrawAction struct {
	Destination Address `json:"destination"`
	Amount      Decimal `json:"amount"`
}

func (WithdrawAction) actionType() string { return "withdraw3" }

var withdrawSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:Withdraw",
	Fields:      []typedField{{"destination", "string"}, {"amount", "string"}},
	NonceField:  "time",
}

func (WithdrawAction) userSignedSpec() *userSignedSpec { return &withdrawSpec }

// Withdraw withdraws USDC to Arbitrum.
func (c *ExchangeClient) Withdraw(ctx context.Context, a WithdrawAction) error {
	return c.do(ctx, a, nil)
}

// USDClassTransferAction moves USDC between the spot and perp balances of
// the signer or, when SubAccount is set, of one of its sub-accounts. It must
// be signed by the account's own key.
type USDClassTransferAction struct {
	Amount Decimal
	// ToPerp moves spot to perp when true, perp to spot otherwise.
	ToPerp bool
	// SubAccount optionally selects a sub-account of the signer.
	SubAccount *Address
}

func (USDClassTransferAction) actionType() string { return "usdClassTransfer" }

// MarshalJSON encodes a in wire form, where a sub-account is appended to the
// amount as "<amount> subaccount:<address>".
func (a USDClassTransferAction) MarshalJSON() ([]byte, error) {
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

func (USDClassTransferAction) userSignedSpec() *userSignedSpec { return &usdClassTransferSpec }

// USDClassTransfer moves USDC between the spot and perp balances.
func (c *ExchangeClient) USDClassTransfer(ctx context.Context, a USDClassTransferAction) error {
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
	return json.Marshal(sendAssetWire{a.Destination, a.SourceDex, a.DestinationDex, a.Token, a.Amount, subAccountWire(a.FromSubAccount)})
}

// sendAssetWire is the wire form of [SendAssetAction] and
// [AgentSendAssetAction], in wire field order.
type sendAssetWire struct {
	Destination    Address `json:"destination"`
	SourceDex      string  `json:"sourceDex"`
	DestinationDex string  `json:"destinationDex"`
	Token          string  `json:"token"`
	Amount         Decimal `json:"amount"`
	FromSubAccount string  `json:"fromSubAccount"`
}

// subAccountWire encodes an optional sub-account, where nil (the main
// account) is "".
func subAccountWire(a *Address) string {
	if a == nil {
		return ""
	}
	return a.String()
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

// SendToEVMWithDataAction sends a token from HyperCore to a contract
// implementing ICoreReceiveWithData, passing it Data. It must be signed by
// the account's own key.
type SendToEVMWithDataAction struct {
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

func (SendToEVMWithDataAction) actionType() string { return "sendToEvmWithData" }

// MarshalJSON encodes a in wire form, where Data is 0x-prefixed hex.
func (a SendToEVMWithDataAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Token                string          `json:"token"`
		Amount               Decimal         `json:"amount"`
		SourceDex            string          `json:"sourceDex"`
		DestinationRecipient string          `json:"destinationRecipient"`
		AddressEncoding      AddressEncoding `json:"addressEncoding"`
		DestinationChainID   uint32          `json:"destinationChainId"`
		GasLimit             uint64          `json:"gasLimit"`
		Data                 string          `json:"data"`
	}{a.Token, a.Amount, a.SourceDex, a.DestinationRecipient, a.AddressEncoding, a.DestinationChainID, a.GasLimit, "0x" + hex.EncodeToString(a.Data)})
}

var sendToEVMWithDataSpec = userSignedSpec{
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

func (SendToEVMWithDataAction) userSignedSpec() *userSignedSpec { return &sendToEVMWithDataSpec }

// SendToEVMWithData sends a token to a contract with a data payload.
func (c *ExchangeClient) SendToEVMWithData(ctx context.Context, a SendToEVMWithDataAction) error {
	return c.do(ctx, a, nil)
}
