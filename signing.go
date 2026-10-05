package hyperliquid

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/big"
	"strconv"
	"strings"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
	"golang.org/x/crypto/sha3"

	"github.com/koative/hyperliquid-go/internal/msgpack"
)

// Signer signs Hyperliquid actions.
//
// [PrivateKeySigner] covers local keys, including API (agent) wallets.
// Implement Signer to keep keys in a KMS, HSM or remote signing service.
type Signer interface {
	// Address returns the address whose key produces the signatures.
	Address() Address
	// SignHash signs a 32-byte EIP-712 digest and returns a recoverable
	// secp256k1 signature.
	SignHash(ctx context.Context, hash [32]byte) (Signature, error)
}

// Signature is a recoverable secp256k1 ECDSA signature.
type Signature struct {
	R, S [32]byte
	V    byte // 27 or 28
}

// MarshalJSON encodes s as {"r":"0x…","s":"0x…","v":27}. R and S are
// written without leading zeros, the canonical form required inside
// multi-sig actions and accepted everywhere else.
func (s Signature) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `{"r":%q,"s":%q,"v":%d}`, minimalHex(s.R), minimalHex(s.S), s.V), nil
}

// UnmarshalJSON decodes {"r":"0x…","s":"0x…","v":27}.
func (s *Signature) UnmarshalJSON(b []byte) error {
	var w struct {
		R string `json:"r"`
		S string `json:"s"`
		V byte   `json:"v"`
	}
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	for _, f := range []struct {
		dst *[32]byte
		hex string
	}{{&s.R, w.R}, {&s.S, w.S}} {
		h := strings.TrimPrefix(f.hex, "0x")
		if len(h) > 64 {
			return fmt.Errorf("hyperliquid: invalid signature component %q", f.hex)
		}
		v, ok := new(big.Int).SetString(h, 16)
		if !ok {
			return fmt.Errorf("hyperliquid: invalid signature component %q", f.hex)
		}
		v.FillBytes(f.dst[:])
	}
	s.V = w.V
	return nil
}

func minimalHex(b [32]byte) string {
	h := strings.TrimLeft(hex.EncodeToString(b[:]), "0")
	if h == "" {
		h = "0"
	}
	return "0x" + h
}

// PrivateKeySigner signs with an in-memory secp256k1 private key.
type PrivateKeySigner struct {
	key  *secp256k1.PrivateKey
	addr Address
}

// NewPrivateKeySigner returns a signer for a 0x-prefixed (or bare) 64-character
// hex private key.
func NewPrivateKeySigner(hexKey string) (*PrivateKeySigner, error) {
	b, err := hex.DecodeString(strings.TrimPrefix(hexKey, "0x"))
	if err != nil || len(b) != 32 {
		return nil, errors.New("hyperliquid: private key must be 32 hex-encoded bytes")
	}
	var k secp256k1.ModNScalar
	if overflow := k.SetByteSlice(b); overflow || k.IsZero() {
		return nil, errors.New("hyperliquid: private key out of range")
	}
	key := secp256k1.NewPrivateKey(&k)
	pub := key.PubKey().SerializeUncompressed()
	h := keccak256(pub[1:])
	s := &PrivateKeySigner{key: key}
	copy(s.addr[:], h[12:])
	return s, nil
}

// Address returns the signer's address.
func (s *PrivateKeySigner) Address() Address { return s.addr }

// SignHash signs hash with a deterministic (RFC 6979) signature.
func (s *PrivateKeySigner) SignHash(_ context.Context, hash [32]byte) (Signature, error) {
	compact := ecdsa.SignCompact(s.key, hash[:], false) // [27+recid | R | S]
	var sig Signature
	sig.V = compact[0]
	copy(sig.R[:], compact[1:33])
	copy(sig.S[:], compact[33:])
	return sig, nil
}

func keccak256(data ...[]byte) [32]byte {
	h := sha3.NewLegacyKeccak256()
	for _, d := range data {
		h.Write(d)
	}
	var out [32]byte
	h.Sum(out[:0])
	return out
}

// typedField is one member of an EIP-712 struct type.
type typedField struct{ Name, Type string }

const (
	l1DomainName         = "Exchange"
	l1ChainID            = 1337
	userSignedDomainName = "HyperliquidSignTransaction"
	// signatureChainID is the chain ID used in the user-signed EIP-712 domain.
	// Any value is accepted as long as it matches the action's
	// signatureChainId; 0x66eee (Arbitrum Sepolia) matches the official SDKs.
	signatureChainID    = "0x66eee"
	signatureChainIDInt = 0x66eee
)

var agentType = []typedField{{"source", "string"}, {"connectionId", "bytes32"}}

// eip712Digest computes keccak256(0x1901 ‖ domainSeparator ‖ hashStruct(message))
// for a domain {name, version "1", chainId, verifyingContract 0x0}. msg holds
// the message fields as JSON values; extra fields are ignored.
func eip712Digest(domainName string, chainID uint64, primaryType string, fields []typedField, msg map[string]json.RawMessage) ([32]byte, error) {
	var domain [5 * 32]byte
	dt := keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	copy(domain[0:], dt[:])
	dn := keccak256([]byte(domainName))
	copy(domain[32:], dn[:])
	dv := keccak256([]byte("1"))
	copy(domain[64:], dv[:])
	binary.BigEndian.PutUint64(domain[120:], chainID)
	// verifyingContract: zero address, already zero.
	ds := keccak256(domain[:])

	var sig strings.Builder
	sig.WriteString(primaryType + "(")
	for i, f := range fields {
		if i > 0 {
			sig.WriteByte(',')
		}
		sig.WriteString(f.Type + " " + f.Name)
	}
	sig.WriteByte(')')
	th := keccak256([]byte(sig.String()))

	enc := make([]byte, 0, 32*(len(fields)+1))
	enc = append(enc, th[:]...)
	for _, f := range fields {
		raw, ok := msg[f.Name]
		if !ok {
			return [32]byte{}, fmt.Errorf("hyperliquid: eip712: missing field %q", f.Name)
		}
		word, err := encodeTypedValue(f.Type, raw)
		if err != nil {
			return [32]byte{}, fmt.Errorf("hyperliquid: eip712: field %q: %w", f.Name, err)
		}
		enc = append(enc, word[:]...)
	}
	sh := keccak256(enc)
	return keccak256([]byte{0x19, 0x01}, ds[:], sh[:]), nil
}

// encodeTypedValue ABI-encodes one EIP-712 atomic value given as JSON.
func encodeTypedValue(typ string, raw json.RawMessage) ([32]byte, error) {
	var word [32]byte
	switch typ {
	case "string":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return word, err
		}
		return keccak256([]byte(s)), nil
	case "bool":
		var b bool
		if err := json.Unmarshal(raw, &b); err != nil {
			return word, err
		}
		if b {
			word[31] = 1
		}
		return word, nil
	case "uint32", "uint64", "uint256":
		n, err := strconv.ParseUint(string(raw), 10, 64)
		if err != nil {
			return word, err
		}
		binary.BigEndian.PutUint64(word[24:], n)
		return word, nil
	case "address", "bytes32", "bytes":
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return word, err
		}
		b, err := hex.DecodeString(strings.TrimPrefix(s, "0x"))
		if err != nil {
			return word, err
		}
		switch {
		case typ == "bytes":
			return keccak256(b), nil
		case typ == "address" && len(b) == 20:
			copy(word[12:], b)
			return word, nil
		case typ == "bytes32" && len(b) == 32:
			copy(word[:], b)
			return word, nil
		}
		return word, fmt.Errorf("invalid %s %q", typ, s)
	}
	return word, fmt.Errorf("unsupported type %q", typ)
}

// l1ActionHash returns keccak256(msgpack(action) ‖ nonce ‖ vault ‖ expiresAfter),
// the "connectionId" signed for L1 actions.
func l1ActionHash(action []byte, nonce uint64, vault *Address, expiresAfter *uint64) ([32]byte, error) {
	data, err := msgpack.FromJSON(action)
	if err != nil {
		return [32]byte{}, err
	}
	data = binary.BigEndian.AppendUint64(data, nonce)
	if vault == nil {
		data = append(data, 0)
	} else {
		data = append(append(data, 1), vault[:]...)
	}
	if expiresAfter != nil {
		data = binary.BigEndian.AppendUint64(append(data, 0), *expiresAfter)
	}
	return keccak256(data), nil
}

// l1Digest returns the EIP-712 digest of the phantom agent for an L1 action hash.
func l1Digest(hash [32]byte, mainnet bool) ([32]byte, error) {
	source := `"b"`
	if mainnet {
		source = `"a"`
	}
	return eip712Digest(l1DomainName, l1ChainID, "Agent", agentType, map[string]json.RawMessage{
		"source":       json.RawMessage(source),
		"connectionId": json.RawMessage(`"0x` + hex.EncodeToString(hash[:]) + `"`),
	})
}

// userSignedSpec describes the EIP-712 type of a user-signed action.
type userSignedSpec struct {
	// PrimaryType is e.g. "HyperliquidTransaction:UsdSend".
	PrimaryType string
	// Fields are the action fields between hyperliquidChain and the nonce,
	// in wire order.
	Fields []typedField
	// NonceField is "nonce" or "time".
	NonceField string
}

// types returns the full EIP-712 field list. Multi-sig signers sign two
// extra address fields after hyperliquidChain.
func (s *userSignedSpec) types(multiSig bool) []typedField {
	t := make([]typedField, 0, len(s.Fields)+4)
	t = append(t, typedField{"hyperliquidChain", "string"})
	if multiSig {
		t = append(t, typedField{"payloadMultiSigUser", "address"}, typedField{"outerSigner", "address"})
	}
	t = append(t, s.Fields...)
	return append(t, typedField{s.NonceField, "uint64"})
}

// userSignedDigest returns the EIP-712 digest of a user-signed action given
// its wire JSON. extra adds message fields (used by multi-sig signers).
func userSignedDigest(spec *userSignedSpec, action []byte, multiSig bool, extra map[string]json.RawMessage) ([32]byte, error) {
	var msg map[string]json.RawMessage
	if err := json.Unmarshal(action, &msg); err != nil {
		return [32]byte{}, err
	}
	maps.Copy(msg, extra)
	return eip712Digest(userSignedDomainName, signatureChainIDInt, spec.PrimaryType, spec.types(multiSig), msg)
}

// typedJSON returns the JSON object of v with prefix fields spliced in front
// and suffix fields appended, preserving v's field order:
// {prefix…, v's fields…, suffix…}. prefix and suffix are raw "key":value
// lists without surrounding braces.
func typedJSON(prefix string, v any, suffix string) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) < 2 || b[0] != '{' {
		return nil, fmt.Errorf("hyperliquid: action %T must encode as a JSON object", v)
	}
	var out bytes.Buffer
	out.Grow(len(prefix) + len(b) + len(suffix) + 2)
	out.WriteByte('{')
	parts := []string{prefix, string(b[1 : len(b)-1]), suffix}
	sep := false
	for _, p := range parts {
		if p == "" {
			continue
		}
		if sep {
			out.WriteByte(',')
		}
		out.WriteString(p)
		sep = true
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}
