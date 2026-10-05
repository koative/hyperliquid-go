package hyperliquid

import (
	"context"
	"encoding/json"
	"fmt"
)

// OutcomeMeta describes the HIP-4 outcome (prediction) markets.
type OutcomeMeta struct {
	Outcomes  []OutcomeSpec     `json:"outcomes"`
	Questions []OutcomeQuestion `json:"questions"`
	Deployers []OutcomeDeployer `json:"deployers"`
	// FeeScale scales the trading fees of outcome markets.
	FeeScale Decimal `json:"feeScale"`
}

// OutcomeSpec describes an outcome. Side i of outcome n trades as asset
// 100000000 + 10*n + i.
type OutcomeSpec struct {
	Outcome     int               `json:"outcome"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	SideSpecs   []OutcomeSideSpec `json:"sideSpecs"`
	// QuoteToken is the quote token name, such as "USDC".
	QuoteToken string `json:"quoteToken"`
	// Venue is the deployer's venue name, if any.
	Venue            string  `json:"venue,omitempty"`
	DeployerFeeScale Decimal `json:"deployerFeeScale,omitempty"`
}

// OutcomeSideSpec describes a side of an outcome, such as "Yes".
type OutcomeSideSpec struct {
	Name string `json:"name"`
}

// OutcomeQuestion groups named outcomes, of which one settles to Yes.
type OutcomeQuestion struct {
	Question    int    `json:"question"`
	Name        string `json:"name"`
	Description string `json:"description"`
	// FallbackOutcome settles to Yes when no named outcome does.
	FallbackOutcome      int   `json:"fallbackOutcome"`
	NamedOutcomes        []int `json:"namedOutcomes"`
	SettledNamedOutcomes []int `json:"settledNamedOutcomes"`
}

// OutcomeDeployer is an outcome market deployer.
type OutcomeDeployer struct {
	Deployer Address `json:"deployer"`
	Venue    string  `json:"venue"`
	// SubDeployers maps function names, such as "settleOutcome", to the
	// addresses allowed to call them on the deployer's behalf.
	SubDeployers TupleMap[string, []Address] `json:"subDeployers"`
}

// OutcomeMeta returns outcome market metadata.
func (c *InfoClient) OutcomeMeta(ctx context.Context) (*OutcomeMeta, error) {
	return infoRequest[*OutcomeMeta](ctx, c, "outcomeMeta", noParams{})
}

// OutcomeTemplate is a template from which outcomes and questions are
// registered. Name and Description contain {keyword} placeholders.
type OutcomeTemplate struct {
	ID          string              `json:"id"`
	Role        OutcomeTemplateRole `json:"role"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	// Keywords maps placeholder keywords to their value kind, such as
	// "string", "dateTime", "uDecimal" or "hlPerp".
	Keywords TupleMap[string, string] `json:"keywords"`
}

// OutcomeTemplateRole is what a template registers: a question, a
// standalone outcome, or an outcome of a question. Exactly one field is set.
type OutcomeTemplateRole struct {
	Question          bool                   `json:"-"`
	StandaloneOutcome *StandaloneOutcomeRole `json:"standaloneOutcome,omitempty"`
	QuestionOutcome   *QuestionOutcomeRole   `json:"questionOutcome,omitempty"`
}

// StandaloneOutcomeRole is the role of a template for standalone outcomes.
type StandaloneOutcomeRole struct {
	// SideNames are the names of the two sides.
	SideNames [2]string `json:"sideNames"`
}

// QuestionOutcomeRole is the role of a template for outcomes of a question.
type QuestionOutcomeRole struct {
	// Parent is the ID of the question's template.
	Parent string `json:"parent"`
}

// UnmarshalJSON decodes the string "question" or an object with one key.
func (r *OutcomeTemplateRole) UnmarshalJSON(b []byte) error {
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		if s != "question" {
			return fmt.Errorf("hyperliquid: unknown outcome template role %q", s)
		}
		*r = OutcomeTemplateRole{Question: true}
		return nil
	}
	type plain OutcomeTemplateRole
	*r = OutcomeTemplateRole{}
	return json.Unmarshal(b, (*plain)(r))
}

// OutcomeTemplates returns the outcome templates.
func (c *InfoClient) OutcomeTemplates(ctx context.Context) ([]OutcomeTemplate, error) {
	return infoRequest[[]OutcomeTemplate](ctx, c, "outcomeTemplates", noParams{})
}

// SettledOutcomeRequest is the request for [InfoClient.SettledOutcome].
type SettledOutcomeRequest struct {
	Outcome int `json:"outcome"`
}

// SettledOutcome is the settlement of an outcome.
type SettledOutcome struct {
	Spec OutcomeSpec `json:"spec"`
	// SettleFraction is the share of the payout settled to side 0, between
	// 0 and 1.
	SettleFraction Decimal `json:"settleFraction"`
	Details        string  `json:"details"`
	// Question is the question the outcome belongs to, if any.
	Question *SettledOutcomeQuestion `json:"question,omitempty"`
}

// SettledOutcomeQuestion is the question of a [SettledOutcome].
type SettledOutcomeQuestion struct {
	Question    QuestionStatus `json:"question"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
}

// QuestionStatus is a question ID tagged with the question's state. Exactly
// one field is set.
type QuestionStatus struct {
	Active  *int `json:"active,omitempty"`
	Settled *int `json:"settled,omitempty"`
}

// SettledOutcome returns the settlement of an outcome, or nil if it has not
// settled.
func (c *InfoClient) SettledOutcome(ctx context.Context, req SettledOutcomeRequest) (*SettledOutcome, error) {
	return infoRequest[*SettledOutcome](ctx, c, "settledOutcome", req)
}
