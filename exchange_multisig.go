package hyperliquid

import (
	"bytes"
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
)

// MultiSigSigners is the signer set of a multi-sig account.
type MultiSigSigners struct {
	AuthorizedUsers []Address `json:"authorizedUsers"`
	// Threshold is the number of signatures an action needs, 1 to 10.
	Threshold int `json:"threshold"`
}

// ConvertToMultiSigUserAction turns the signer into a multi-sig account
// controlled by Signers, or, sent through [ExchangeClient.MultiSig] with
// nil Signers, back into a normal account. It must be signed by the
// account's own key.
type ConvertToMultiSigUserAction struct {
	Signers *MultiSigSigners
}

func (ConvertToMultiSigUserAction) actionType() string { return "convertToMultiSigUser" }

// MarshalJSON encodes a in wire form: signers is a JSON string holding the
// signer set (users sorted, formatted like Python's json.dumps) or "null".
func (a ConvertToMultiSigUserAction) MarshalJSON() ([]byte, error) {
	signers := "null"
	if s := a.Signers; s != nil {
		users := slices.Clone(s.AuthorizedUsers)
		slices.SortFunc(users, func(x, y Address) int { return bytes.Compare(x[:], y[:]) })
		quoted := make([]string, len(users))
		for i, u := range users {
			quoted[i] = `"` + u.String() + `"`
		}
		signers = `{"authorizedUsers": [` + strings.Join(quoted, ", ") + `], "threshold": ` + strconv.Itoa(s.Threshold) + `}`
	}
	return json.Marshal(struct {
		Signers string `json:"signers"`
	}{signers})
}

var convertToMultiSigUserSpec = userSignedSpec{
	PrimaryType: "HyperliquidTransaction:ConvertToMultiSigUser",
	Fields:      []typedField{{"signers", "string"}},
	NonceField:  "nonce",
}

func (ConvertToMultiSigUserAction) userSignedSpec() *userSignedSpec {
	return &convertToMultiSigUserSpec
}

// ConvertToMultiSigUser converts the signer into a multi-sig account. To
// convert a multi-sig account back, submit the action with nil Signers via
// [ExchangeClient.MultiSig].
func (c *ExchangeClient) ConvertToMultiSigUser(ctx context.Context, a ConvertToMultiSigUserAction) error {
	return c.do(ctx, a, nil)
}
