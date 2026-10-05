package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"
)

// ExchangeClient signs and submits actions to the /exchange endpoint. It is
// safe for concurrent use.
//
// Methods return an [*APIError] when the exchange rejects the request, and
// a joined [*StatusError] for each failed item of a batch (orders, cancels).
type ExchangeClient struct {
	network      Network
	signer       Signer
	tr           transport
	vault        *Address
	expiresAfter *uint64
	nonces       *nonceClock
}

// NewExchangeClient returns a client that signs with signer on network.
// The signer may be the account's own key or an API (agent) wallet approved
// with [ExchangeClient.ApproveAgent]; user-signed actions such as transfers
// and withdrawals require the account's own key.
func NewExchangeClient(network Network, signer Signer, opts ...Option) *ExchangeClient {
	o := newOptions(opts)
	c := &ExchangeClient{network: network, signer: signer, tr: o.transport, nonces: new(nonceClock)}
	if c.tr == nil {
		c.tr = &httpTransport{baseURL: network.APIURL, client: o.httpClient}
	}
	return c
}

// Signer returns the client's signer.
func (c *ExchangeClient) Signer() Signer { return c.signer }

// WithVault returns a copy of c that trades on behalf of a vault or
// sub-account. It applies to L1 actions only; user-signed actions always act
// for the signer.
func (c *ExchangeClient) WithVault(vault Address) *ExchangeClient {
	cp := *c
	cp.vault = &vault
	return &cp
}

// WithExpiresAfter returns a copy of c whose L1 actions are rejected by the
// exchange if they arrive after t.
func (c *ExchangeClient) WithExpiresAfter(t time.Time) *ExchangeClient {
	cp := *c
	ms := uint64(t.UnixMilli())
	cp.expiresAfter = &ms
	return &cp
}

// nonceClock issues strictly increasing millisecond timestamps, so concurrent
// actions from one signer never reuse a nonce.
type nonceClock struct{ last atomic.Uint64 }

func (n *nonceClock) next() uint64 {
	now := uint64(time.Now().UnixMilli())
	for {
		last := n.last.Load()
		next := max(now, last+1)
		if n.last.CompareAndSwap(last, next) {
			return next
		}
	}
}

// Action is an exchange action. The *Action types in this package implement
// it; pass them to [ExchangeClient.SignMultiSig] and [ExchangeClient.MultiSig].
type Action interface {
	// actionType returns the wire "type" of the action.
	actionType() string
}

// userSignedAction is an action signed directly by the user's wallet with
// EIP-712 rather than through the L1 phantom agent.
type userSignedAction interface {
	Action
	userSignedSpec() *userSignedSpec
}

// multiSigPayloader is implemented by actions whose multi-sig payload
// differs from the form their signers sign.
type multiSigPayloader interface {
	multiSigPayload() Action
}

// encodeAction returns the wire JSON of a: {"type", …fields} for L1 actions,
// and {"type", "signatureChainId", "hyperliquidChain", …fields, nonce} for
// user-signed actions.
func encodeAction(a Action, nonce uint64, network Network) ([]byte, error) {
	typ := `"type":` + strconv.Quote(a.actionType())
	us, ok := a.(userSignedAction)
	if !ok {
		return typedJSON(typ, a, "")
	}
	chain := "Testnet"
	if network.Mainnet {
		chain = "Mainnet"
	}
	return typedJSON(
		typ+`,"signatureChainId":"`+signatureChainID+`","hyperliquidChain":"`+chain+`"`,
		a,
		strconv.Quote(us.userSignedSpec().NonceField)+":"+strconv.FormatUint(nonce, 10),
	)
}

// sign signs the wire JSON of a.
func (c *ExchangeClient) sign(ctx context.Context, a Action, wire []byte, nonce uint64) (Signature, error) {
	var digest [32]byte
	var err error
	if us, ok := a.(userSignedAction); ok {
		digest, err = userSignedDigest(us.userSignedSpec(), wire, false, nil)
	} else {
		var h [32]byte
		if h, err = l1ActionHash(wire, nonce, c.vault, c.expiresAfter); err == nil {
			digest, err = l1Digest(h, c.network.Mainnet)
		}
	}
	if err != nil {
		return Signature{}, err
	}
	return c.signer.SignHash(ctx, digest)
}

type exchangeRequest struct {
	Action       json.RawMessage `json:"action"`
	Nonce        uint64          `json:"nonce"`
	Signature    Signature       `json:"signature"`
	VaultAddress *Address        `json:"vaultAddress,omitempty"`
	ExpiresAfter *uint64         `json:"expiresAfter,omitempty"`
}

// do signs a with a fresh nonce, submits it and decodes the response data
// into out (which may be nil).
func (c *ExchangeClient) do(ctx context.Context, a Action, out any) error {
	return c.send(ctx, a, c.nonces.next(), out)
}

func (c *ExchangeClient) send(ctx context.Context, a Action, nonce uint64, out any) error {
	wire, err := encodeAction(a, nonce, c.network)
	if err != nil {
		return err
	}
	sig, err := c.sign(ctx, a, wire, nonce)
	if err != nil {
		return err
	}
	return c.post(ctx, wire, sig, nonce, a, out)
}

func (c *ExchangeClient) post(ctx context.Context, wire []byte, sig Signature, nonce uint64, a Action, out any) error {
	req := exchangeRequest{Action: wire, Nonce: nonce, Signature: sig}
	if _, ok := a.(userSignedAction); !ok {
		req.VaultAddress, req.ExpiresAfter = c.vault, c.expiresAfter
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	raw, err := c.tr.request(ctx, "exchange", body)
	if err != nil {
		return err
	}
	return decodeExchangeResponse(raw, out)
}

// decodeExchangeResponse unwraps {"status":"ok","response":{"type","data"}}
// into out, or returns the *APIError of {"status":"err","response":msg}.
func decodeExchangeResponse(raw json.RawMessage, out any) error {
	var r struct {
		Status   string          `json:"status"`
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return fmt.Errorf("hyperliquid: decode exchange response: %w", err)
	}
	if r.Status != "ok" {
		var msg string
		if json.Unmarshal(r.Response, &msg) != nil {
			msg = string(raw)
		}
		return &APIError{Message: msg}
	}
	if out == nil {
		return nil
	}
	var resp struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(r.Response, &resp); err != nil {
		return fmt.Errorf("hyperliquid: decode exchange response: %w", err)
	}
	if len(resp.Data) == 0 {
		return nil
	}
	if err := json.Unmarshal(resp.Data, out); err != nil {
		return fmt.Errorf("hyperliquid: decode exchange response: %w", err)
	}
	return nil
}

// StatusError reports that the exchange accepted a batch request but
// rejected one of its items, such as a single order of an order action.
type StatusError struct {
	// Index is the position of the failed item in the request.
	Index int
	// Message is the exchange's reason.
	Message string
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("hyperliquid: item %d: %s", e.Index, e.Message)
}

// statusErrors joins a *StatusError for every non-empty message in msgs.
func statusErrors(msgs []string) error {
	var errs []error
	for i, m := range msgs {
		if m != "" {
			errs = append(errs, &StatusError{Index: i, Message: m})
		}
	}
	return errors.Join(errs...)
}

// SignMultiSig returns c's signature approving action a for the multi-sig
// account multiSigUser. The action is submitted by the leader
// outerSigner with [ExchangeClient.MultiSig] using the same nonce; for L1
// actions the vault and expiry of both clients must match.
func (c *ExchangeClient) SignMultiSig(ctx context.Context, a Action, multiSigUser, outerSigner Address, nonce uint64) (Signature, error) {
	wire, err := encodeAction(a, nonce, c.network)
	if err != nil {
		return Signature{}, err
	}
	var digest [32]byte
	if us, ok := a.(userSignedAction); ok {
		digest, err = userSignedDigest(us.userSignedSpec(), wire, true, map[string]json.RawMessage{
			"payloadMultiSigUser": json.RawMessage(strconv.Quote(multiSigUser.String())),
			"outerSigner":         json.RawMessage(strconv.Quote(outerSigner.String())),
		})
	} else {
		envelope := fmt.Appendf(nil, `[%q,%q,%s]`, multiSigUser, outerSigner, wire)
		var h [32]byte
		if h, err = l1ActionHash(envelope, nonce, c.vault, c.expiresAfter); err == nil {
			digest, err = l1Digest(h, c.network.Mainnet)
		}
	}
	if err != nil {
		return Signature{}, err
	}
	return c.signer.SignHash(ctx, digest)
}

var sendMultiSigSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:SendMultiSig",
	Fields:      []typedField{{"multiSigActionHash", "bytes32"}},
	NonceField:  "nonce",
}

// MultiSig submits action a for the multi-sig account multiSigUser, with c's
// signer as the leader (outer signer). signatures are the authorized users'
// approvals from [ExchangeClient.SignMultiSig], made with the same nonce.
// It returns the raw response data of the inner action, if any.
func (c *ExchangeClient) MultiSig(ctx context.Context, multiSigUser Address, a Action, nonce uint64, signatures []Signature) (json.RawMessage, error) {
	payload := a
	if p, ok := a.(multiSigPayloader); ok {
		payload = p.multiSigPayload()
	}
	inner, err := encodeAction(payload, nonce, c.network)
	if err != nil {
		return nil, err
	}
	sigs, err := json.Marshal(signatures)
	if err != nil {
		return nil, err
	}
	body := fmt.Appendf(nil, `"signatureChainId":%q,"signatures":%s,"payload":{"multiSigUser":%q,"outerSigner":%q,"action":%s}`,
		signatureChainID, sigs, multiSigUser, c.signer.Address(), inner)
	h, err := l1ActionHash(append(append([]byte{'{'}, body...), '}'), nonce, c.vault, c.expiresAfter)
	if err != nil {
		return nil, err
	}
	chain := "Testnet"
	if c.network.Mainnet {
		chain = "Mainnet"
	}
	digest, err := eip712Digest(userSignedDomainName, signatureChainIDInt, sendMultiSigSpec.PrimaryType, sendMultiSigSpec.types(false),
		map[string]json.RawMessage{
			"hyperliquidChain":   json.RawMessage(strconv.Quote(chain)),
			"multiSigActionHash": json.RawMessage(fmt.Sprintf(`"0x%x"`, h)),
			"nonce":              json.RawMessage(strconv.FormatUint(nonce, 10)),
		})
	if err != nil {
		return nil, err
	}
	sig, err := c.signer.SignHash(ctx, digest)
	if err != nil {
		return nil, err
	}
	wire := append(append([]byte(`{"type":"multiSig",`), body...), '}')
	var data json.RawMessage
	err = c.post(ctx, wire, sig, nonce, multiSigAction{}, &data)
	return data, err
}

// multiSigAction marks the multiSig wrapper as an L1 action when posting.
type multiSigAction struct{}

func (multiSigAction) actionType() string { return "multiSig" }
