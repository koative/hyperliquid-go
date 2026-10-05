package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

// ExplorerTx is a transaction as reported by the block explorer.
type ExplorerTx struct {
	// Action is the transaction's action JSON, such as {"type":"order",...}.
	// Some historical transactions encode it as a JSON array.
	Action json.RawMessage `json:"action"`
	Block  int64           `json:"block"`
	// Error is the failure reason, or nil if the transaction succeeded.
	Error *string `json:"error"`
	Hash  string  `json:"hash"`
	// Time is in milliseconds since the Unix epoch.
	Time int64   `json:"time"`
	User Address `json:"user"`
}

// BlockDetails is an L1 block with its transactions.
type BlockDetails struct {
	// BlockTime is in milliseconds since the Unix epoch.
	BlockTime int64        `json:"blockTime"`
	Hash      string       `json:"hash"`
	Height    int64        `json:"height"`
	NumTxs    int          `json:"numTxs"`
	Proposer  Address      `json:"proposer"`
	Txs       []ExplorerTx `json:"txs"`
}

// BlockDetailsRequest is the request for [InfoClient.BlockDetails].
type BlockDetailsRequest struct {
	Height int64 `json:"height"`
}

// BlockDetails returns a block from the explorer.
func (c *InfoClient) BlockDetails(ctx context.Context, req BlockDetailsRequest) (*BlockDetails, error) {
	resp, err := explorerRequest[struct {
		Type         string        `json:"type"`
		BlockDetails *BlockDetails `json:"blockDetails"`
	}](ctx, c, "blockDetails", req)
	return resp.BlockDetails, err
}

// TxDetailsRequest is the request for [InfoClient.TxDetails].
type TxDetailsRequest struct {
	// Hash is the 0x-prefixed transaction hash.
	Hash string `json:"hash"`
}

// TxDetails returns a transaction from the explorer.
func (c *InfoClient) TxDetails(ctx context.Context, req TxDetailsRequest) (*ExplorerTx, error) {
	resp, err := explorerRequest[struct {
		Type string      `json:"type"`
		Tx   *ExplorerTx `json:"tx"`
	}](ctx, c, "txDetails", req)
	return resp.Tx, err
}

// UserDetailsRequest is the request for [InfoClient.UserDetails].
type UserDetailsRequest struct {
	User Address `json:"user"`
}

// UserDetails returns a user's recent transactions from the explorer.
func (c *InfoClient) UserDetails(ctx context.Context, req UserDetailsRequest) ([]ExplorerTx, error) {
	resp, err := explorerRequest[struct {
		Type string       `json:"type"`
		Txs  []ExplorerTx `json:"txs"`
	}](ctx, c, "userDetails", req)
	return resp.Txs, err
}

// explorerRequest posts {"type": typ, ...req} to the explorer and decodes the
// response, turning {"type":"error","message":...} into an [*APIError].
func explorerRequest[T any](ctx context.Context, c *InfoClient, typ string, req any) (T, error) {
	var out T
	raw, err := postTyped[json.RawMessage](ctx, c.explorer, "explorer", typ, req)
	if err != nil {
		return out, err
	}
	var e struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	}
	if json.Unmarshal(raw, &e) == nil && e.Type == "error" {
		return out, &APIError{Message: e.Message}
	}
	if err := unmarshal(raw, &out); err != nil {
		return out, fmt.Errorf("hyperliquid: decode %s response: %w", typ, err)
	}
	return out, nil
}
