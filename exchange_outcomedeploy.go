package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
)

// ActivateOutcomeDeployerAction makes the signer a HIP-4 outcome deployer
// with its own venue, or permanently retires it. Set exactly one field.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-4-deployer-actions.
type ActivateOutcomeDeployerAction struct {
	Activate   *OutcomeDeployerActivation
	Deactivate bool
}

func (ActivateOutcomeDeployerAction) actionType() string { return "activateOutcomeDeployer" }

// MarshalJSON encodes a as {"activate":{…}} or {"deactivate":null}.
func (a ActivateOutcomeDeployerAction) MarshalJSON() ([]byte, error) {
	switch {
	case a.Activate != nil && !a.Deactivate:
		return json.Marshal(struct {
			Activate *OutcomeDeployerActivation `json:"activate"`
		}{a.Activate})
	case a.Deactivate && a.Activate == nil:
		return []byte(`{"deactivate":null}`), nil
	}
	return nil, errors.New("hyperliquid: ActivateOutcomeDeployerAction: set exactly one of Activate and Deactivate")
}

// ActivateOutcomeDeployer activates or deactivates the signer as an outcome
// deployer.
func (c *ExchangeClient) ActivateOutcomeDeployer(ctx context.Context, a ActivateOutcomeDeployerAction) error {
	return c.do(ctx, a, nil)
}

// OutcomeDeployerActivation claims an outcome deployer's venue.
type OutcomeDeployerActivation struct {
	// VenueName is 2-4 lowercase letters, unique across venues and perp
	// dexes.
	VenueName string `json:"venueName"`
}

// OutcomeDeployAction deploys or settles HIP-4 outcomes on a venue.
//
// See https://hyperliquid.gitbook.io/hyperliquid-docs/for-developers/api/hip-4-deployer-actions.
type OutcomeDeployAction struct {
	// Venue is the deployer's venue name; sub-deployers pass the venue they
	// act for.
	Venue     string           `json:"venue"`
	Operation OutcomeOperation `json:"operation"`
}

func (OutcomeDeployAction) actionType() string { return "outcomeDeploy" }

// OutcomeDeploy submits an outcome deployer action.
func (c *ExchangeClient) OutcomeDeploy(ctx context.Context, a OutcomeDeployAction) error {
	return c.do(ctx, a, nil)
}

// OutcomeOperation is the operation of an [OutcomeDeployAction]. Set exactly
// one field.
type OutcomeOperation struct {
	// RegisterStandaloneOutcomeFromTemplate deploys a standalone yes/no
	// outcome; DeployerFeeScale is required.
	RegisterStandaloneOutcomeFromTemplate *OutcomeTemplateInstance `json:"registerStandaloneOutcomeFromTemplate,omitempty"`
	// RegisterQuestionFromTemplate deploys a question and its named
	// outcomes.
	RegisterQuestionFromTemplate *RegisterQuestionFromTemplate `json:"registerQuestionFromTemplate,omitempty"`
	// RegisterAndAssociateNamedOutcomeFromTemplate adds a named outcome to a
	// live question.
	RegisterAndAssociateNamedOutcomeFromTemplate *RegisterNamedOutcome `json:"registerAndAssociateNamedOutcomeFromTemplate,omitempty"`
	// SettleOutcome settles one outcome.
	SettleOutcome *OutcomeSettlement `json:"settleOutcome,omitempty"`
	// SettleQuestion2 settles the remaining named outcomes of a question.
	SettleQuestion2 *SettleQuestion `json:"settleQuestion2,omitempty"`
	// SetSubDeployers grants or revokes sub-deployer permissions; Variant is
	// an operation name, with "settleQuestion" covering SettleQuestion2.
	SetSubDeployers []SubDeployer `json:"setSubDeployers,omitempty"`
}

// OutcomeTemplateInstance instantiates an outcome or question template.
type OutcomeTemplateInstance struct {
	ID string `json:"id"`
	// KeywordToValue maps each template keyword to its value.
	KeywordToValue TupleMap[string, string] `json:"keywordToValue"`
	// DeployerFeeScale, between 0 and 10, is required for standalone
	// outcomes and questions and must be empty for named outcomes. Users
	// pay the base fee times DeployerFeeScale + max(DeployerFeeScale, 1),
	// of which the deployer receives DeployerFeeScale.
	DeployerFeeScale Decimal `json:"deployerFeeScale,omitempty"`
}

// RegisterQuestionFromTemplate deploys a question and its named outcomes.
type RegisterQuestionFromTemplate struct {
	QuestionTemplateInstance OutcomeTemplateInstance `json:"questionTemplateInstance"`
	// NamedOutcomeTemplateInstances holds at most 100 outcomes.
	NamedOutcomeTemplateInstances []OutcomeTemplateInstance `json:"namedOutcomeTemplateInstances"`
}

// RegisterNamedOutcome adds a named outcome to a live question.
type RegisterNamedOutcome struct {
	Question                     int                     `json:"question"`
	NamedOutcomeTemplateInstance OutcomeTemplateInstance `json:"namedOutcomeTemplateInstance"`
}

// OutcomeSettlement settles an outcome.
type OutcomeSettlement struct {
	Outcome int `json:"outcome"`
	// SettleFraction is the payout fraction of the yes side, between 0 and
	// 1; outcomes of a question settle to exactly 0 or 1.
	SettleFraction Decimal `json:"settleFraction"`
	// Details must be empty.
	Details string `json:"details"`
	// NameAndDescription and SideNames must match the outcome exactly.
	NameAndDescription [2]string `json:"nameAndDescription"`
	SideNames          [2]string `json:"sideNames"`
}

// SettleQuestion settles every remaining named outcome of a question,
// exactly one of them to 1.
type SettleQuestion struct {
	Question           int                 `json:"question"`
	OutcomeSettlements []OutcomeSettlement `json:"outcomeSettlements"`
	NameAndDescription [2]string           `json:"nameAndDescription"`
}
