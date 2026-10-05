package hyperliquid

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
)

// errOf drops the result of a method that returns one with its error.
func errOf[T any](_ T, err error) error { return err }

// TestLiveExchange submits every exchange action, and every variant of the
// variant actions, to Testnet signed by a fresh random key. The account does
// not exist, so the exchange rejects each action after recovering its signer
// from the signature: an error naming our address proves it re-derived the
// exact bytes we signed. Any other error (another address means the
// exchange hashed different bytes; HTTP 422 means it could not deserialize
// the action) or a success fails the case.
func TestLiveExchange(t *testing.T) {
	requireLive(t)
	var key [32]byte
	_, _ = rand.Read(key[:])
	signer, err := NewPrivateKeySigner(hex.EncodeToString(key[:]))
	if err != nil {
		t.Fatal(err)
	}
	c := NewExchangeClient(Testnet, signer)
	c.tr = retryTransport{tr: c.tr}
	me := signer.Address()
	t.Logf("signer %s", me)

	ctx := context.Background()
	other := MustParseAddress("0x5e9ee1089755c3435139848e47e6635505d5a13a")
	sub := MustParseAddress("0x1719884eb866cb12b2287399b15f7db5e7d775ea")
	cloid := MustParseCloid("0x00000000000000000000000000000001")
	usdc := "USDC:0xeb62eee3685fc4c43992febcd9e75443"
	str := func(s string) *string { return &s }
	yes, no := true, false
	gas, oiCap := uint64(100000000), uint64(1000000)
	three, bps := 3, 500
	pct, px := Decimal("1.5"), Decimal("100")
	ord := Order{Asset: 0, IsBuy: true, Price: "10000", Size: "0.001", Type: OrderType{Limit: &LimitOrder{Tif: TifGtc}}}
	tmpl := OutcomeTemplateInstance{ID: "t", KeywordToValue: TupleMap[string, string]{"b": "2", "a": "1"}, DeployerFeeScale: "1"}
	named := OutcomeTemplateInstance{ID: "t-o", KeywordToValue: TupleMap[string, string]{"choice": "A"}}
	settle := OutcomeSettlement{Outcome: 1, SettleFraction: "1", NameAndDescription: [2]string{"n", "d"}, SideNames: [2]string{"Yes", "No"}}
	spot := func(a SpotDeployAction) func() error { return func() error { return c.SpotDeploy(ctx, a) } }
	perp := func(a PerpDeployAction) func() error { return func() error { return c.PerpDeploy(ctx, a) } }
	outcome := func(op OutcomeOperation) func() error {
		return func() error { return c.OutcomeDeploy(ctx, OutcomeDeployAction{Venue: "zz", Operation: op}) }
	}
	userOutcome := func(a UserOutcomeAction) func() error { return func() error { return c.UserOutcome(ctx, a) } }
	multiSig := func(a Action) func() error {
		return func() error {
			nonce := NextNonce()
			sig, err := c.SignMultiSig(ctx, a, me, me, nonce)
			if err != nil {
				return err
			}
			return errOf(c.MultiSig(ctx, me, a, nonce, []Signature{sig}))
		}
	}

	cases := []struct {
		name string
		f    func() error
	}{
		// Trading.
		{"order", func() error { return errOf(c.Order(ctx, OrderAction{Orders: []Order{ord}})) }},
		{"order/trigger", func() error {
			tp := Order{
				Asset: 0, Size: "0.001", Price: "90000", ReduceOnly: true, Cloid: &cloid,
				Type: OrderType{Trigger: &TriggerOrder{IsMarket: true, TriggerPx: "90000", Tpsl: TpslTakeProfit}},
			}
			return errOf(c.Order(ctx, OrderAction{Orders: []Order{ord, tp}, Grouping: GroupingNormalTpsl}))
		}},
		{"order/builder", func() error {
			return errOf(c.Order(ctx, OrderAction{Orders: []Order{ord}, Builder: &Builder{Address: other, Fee: 10}}))
		}},
		{"order/priority", func() error {
			ioc := ord
			ioc.Type = OrderType{Limit: &LimitOrder{Tif: TifIoc}}
			return errOf(c.Order(ctx, OrderAction{Orders: []Order{ioc}, Grouping: GroupingPriority(100)}))
		}},
		{"order/expiresAfter", func() error {
			return errOf(c.WithExpiresAfter(time.Now().Add(time.Minute)).Order(ctx, OrderAction{Orders: []Order{ord}}))
		}},
		{"modify/oid", func() error { return c.Modify(ctx, ModifyAction{Oid: OrderRef{Oid: 1}, Order: ord, AlwaysPlace: true}) }},
		{"modify/cloid", func() error { return c.Modify(ctx, ModifyAction{Oid: OrderRef{Cloid: &cloid}, Order: ord}) }},
		{"modify/vault", func() error { return c.WithVault(other).Modify(ctx, ModifyAction{Oid: OrderRef{Oid: 1}, Order: ord}) }},
		{"batchModify", func() error {
			return errOf(c.BatchModify(ctx, BatchModifyAction{Modifies: []Modify{{Oid: OrderRef{Oid: 1}, Order: ord}}, AlwaysPlace: true}))
		}},
		{"cancel", func() error { return c.Cancel(ctx, CancelAction{Cancels: []Cancel{{Asset: 0, Oid: 1}}, Fast: true}) }},
		{"cancelByCloid", func() error {
			return c.CancelByCloid(ctx, CancelByCloidAction{Cancels: []CancelByCloid{{Asset: 0, Cloid: cloid}}, Fast: true})
		}},
		{"scheduleCancel/set", func() error {
			return c.ScheduleCancel(ctx, ScheduleCancelAction{Time: time.Now().Add(time.Minute).UnixMilli()})
		}},
		{"scheduleCancel/unset", func() error { return c.ScheduleCancel(ctx, ScheduleCancelAction{}) }},
		{"twapOrder", func() error {
			return errOf(c.TwapOrder(ctx, TwapOrderAction{Twap: Twap{Asset: 0, IsBuy: true, Size: "0.01", Minutes: 30}}))
		}},
		{"twapOrder/details", func() error {
			return errOf(c.TwapOrder(ctx, TwapOrderAction{
				Twap:    Twap{Asset: 0, IsBuy: true, Size: "0.01", Minutes: 30, Randomize: true},
				Details: &TwapDetails{Trigger: &TwapActivation{Price: "100000", Above: true}, StopPx: &px},
			}))
		}},
		{"twapCancel", func() error { return c.TwapCancel(ctx, TwapCancelAction{Asset: 0, TwapID: 1}) }},
		{"trailingStop/pct", func() error {
			return errOf(c.TrailingStop(ctx, TrailingStopAction{Asset: 0, Size: "0.001", ReduceOnly: true, Retracement: Retracement{Pct: &pct}}))
		}},
		{"trailingStop/px", func() error {
			return errOf(c.TrailingStop(ctx, TrailingStopAction{Asset: 0, IsBuy: true, Size: "0.001", Retracement: Retracement{Px: &px}, ActivationPx: &px}))
		}},
		{"updateLeverage", func() error { return c.UpdateLeverage(ctx, UpdateLeverageAction{Asset: 0, IsCross: true, Leverage: 5}) }},
		{"updateIsolatedMargin", func() error {
			return c.UpdateIsolatedMargin(ctx, UpdateIsolatedMarginAction{Asset: 0, IsBuy: true, Ntli: -1500000})
		}},
		{"topUpIsolatedOnlyMargin", func() error {
			return c.TopUpIsolatedOnlyMargin(ctx, TopUpIsolatedOnlyMarginAction{Asset: 0, Leverage: "3.5"})
		}},
		{"borrowLend", func() error { return c.BorrowLend(ctx, BorrowLendAction{Operation: BorrowLendRepay, Token: 0}) }},
		{"borrowLend/amount", func() error {
			return c.BorrowLend(ctx, BorrowLendAction{Operation: BorrowLendSupply, Token: 0, Amount: &px})
		}},
		{"noop", func() error { return c.Noop(ctx, NextNonce()) }},
		{"reserveRequestWeight", func() error { return c.ReserveRequestWeight(ctx, ReserveRequestWeightAction{Weight: 1}) }},
		{"reserveRequestWeight/destination", func() error {
			return c.ReserveRequestWeight(ctx, ReserveRequestWeightAction{Weight: 1, Destination: &other})
		}},

		// Account.
		{"evmUserModify", func() error { return c.EVMUserModify(ctx, EVMUserModifyAction{UsingBigBlocks: true}) }},
		{"setReferrer", func() error { return c.SetReferrer(ctx, SetReferrerAction{Code: "TESTCODE"}) }},
		{"registerReferrer", func() error { return c.RegisterReferrer(ctx, RegisterReferrerAction{Code: "TESTCODE"}) }},
		{"setDisplayName", func() error { return c.SetDisplayName(ctx, SetDisplayNameAction{DisplayName: "gotest"}) }},
		{"claimRewards", func() error { return c.ClaimRewards(ctx) }},
		{"createSubAccount", func() error { return errOf(c.CreateSubAccount(ctx, CreateSubAccountAction{Name: "sub1"})) }},
		{"subAccountModify", func() error { return c.SubAccountModify(ctx, SubAccountModifyAction{SubAccountUser: sub, Name: "x"}) }},
		{"subAccountTransfer", func() error {
			return c.SubAccountTransfer(ctx, SubAccountTransferAction{SubAccountUser: sub, IsDeposit: true, USD: 1000000})
		}},
		{"subAccountSpotTransfer", func() error {
			return c.SubAccountSpotTransfer(ctx, SubAccountSpotTransferAction{SubAccountUser: sub, IsDeposit: true, Token: usdc, Amount: "1"})
		}},
		{"createVault", func() error {
			return errOf(c.CreateVault(ctx, CreateVaultAction{Name: "govault", Description: "test description", InitialUSD: 100000000}))
		}},
		{"vaultModify", func() error {
			return c.VaultModify(ctx, VaultModifyAction{VaultAddress: other, AllowDeposits: &yes, AlwaysCloseOnWithdraw: &no})
		}},
		{"vaultDistribute", func() error { return c.VaultDistribute(ctx, VaultDistributeAction{VaultAddress: other, USD: 1000000}) }},
		{"vaultTransfer", func() error {
			return c.VaultTransfer(ctx, VaultTransferAction{VaultAddress: other, IsDeposit: true, USD: 1000000})
		}},
		{"agentSendAsset", func() error {
			return c.AgentSendAsset(ctx, AgentSendAssetAction{Destination: other, DestinationDex: "spot", Token: usdc, Amount: "1"})
		}},
		{"agentSendAsset/fromSubAccount", func() error {
			return c.AgentSendAsset(ctx, AgentSendAssetAction{Destination: other, DestinationDex: "spot", Token: usdc, Amount: "1", FromSubAccount: &sub})
		}},
		{"agentSetAbstraction", func() error {
			return c.AgentSetAbstraction(ctx, AgentSetAbstractionAction{Abstraction: AbstractionUnifiedAccount})
		}},
		{"spotUser", func() error {
			return c.SpotUser(ctx, SpotUserAction{ToggleSpotDusting: &ToggleSpotDusting{OptOut: true}})
		}},
		{"userOutcome/splitOutcome", userOutcome(UserOutcomeAction{SplitOutcome: &SplitOutcome{Outcome: 1, Amount: "1"}})},
		{"userOutcome/mergeOutcome", userOutcome(UserOutcomeAction{MergeOutcome: &MergeOutcome{Outcome: 1}})},
		{"userOutcome/mergeQuestion", userOutcome(UserOutcomeAction{MergeQuestion: &MergeQuestion{Question: 1, Amount: &px}})},
		{"userOutcome/negateOutcome", userOutcome(UserOutcomeAction{NegateOutcome: &NegateOutcome{Question: 1, Outcome: 1, Amount: "1"}})},

		// User-signed.
		{"usdSend", func() error { return c.USDSend(ctx, USDSendAction{Destination: other, Amount: "1"}) }},
		{"spotSend", func() error { return c.SpotSend(ctx, SpotSendAction{Destination: other, Token: usdc, Amount: "1"}) }},
		{"withdraw3", func() error { return c.Withdraw(ctx, WithdrawAction{Destination: other, Amount: "5"}) }},
		{"usdClassTransfer", func() error { return c.USDClassTransfer(ctx, USDClassTransferAction{Amount: "1", ToPerp: true}) }},
		{"usdClassTransfer/subAccount", func() error {
			return c.USDClassTransfer(ctx, USDClassTransferAction{Amount: "1", SubAccount: &sub})
		}},
		{"sendAsset", func() error {
			return c.SendAsset(ctx, SendAssetAction{Destination: other, DestinationDex: "spot", Token: usdc, Amount: "1"})
		}},
		{"sendAsset/fromSubAccount", func() error {
			return c.SendAsset(ctx, SendAssetAction{Destination: other, DestinationDex: "spot", Token: usdc, Amount: "1", FromSubAccount: &sub})
		}},
		{"sendToEvmWithData", func() error {
			return c.SendToEVMWithData(ctx, SendToEVMWithDataAction{
				Token: "USDC", Amount: "1", DestinationRecipient: other.String(),
				AddressEncoding: AddressEncodingHex, DestinationChainID: 998, GasLimit: 200000, Data: []byte{1, 2, 3},
			})
		}},
		{"approveAgent/named", func() error { return c.ApproveAgent(ctx, ApproveAgentAction{AgentAddress: other, AgentName: "bot"}) }},
		{"approveAgent/unnamed", func() error { return c.ApproveAgent(ctx, ApproveAgentAction{AgentAddress: other}) }},
		{"approveBuilderFee", func() error {
			return c.ApproveBuilderFee(ctx, ApproveBuilderFeeAction{MaxFeeRate: "0.001", Builder: other})
		}},
		{"cDeposit", func() error { return c.CDeposit(ctx, CDepositAction{Wei: 100000000}) }},
		{"cWithdraw", func() error { return c.CWithdraw(ctx, CWithdrawAction{Wei: 100000000}) }},
		{"tokenDelegate", func() error { return c.TokenDelegate(ctx, TokenDelegateAction{Validator: other, Wei: 100000000}) }},
		{"linkStakingUser", func() error { return c.LinkStakingUser(ctx, LinkStakingUserAction{User: other}) }},
		{"stakingLinkDisableTradingUser", func() error {
			return c.StakingLinkDisableTradingUser(ctx, StakingLinkDisableTradingUserAction{TradingUser: other})
		}},
		{"userPortfolioMargin", func() error { return c.UserPortfolioMargin(ctx, UserPortfolioMarginAction{User: me, Enabled: true}) }},
		{"userSetAbstraction", func() error {
			return c.UserSetAbstraction(ctx, UserSetAbstractionAction{User: me, Abstraction: AbstractionUnifiedAccount})
		}},
		{"convertToMultiSigUser/set", func() error {
			return c.ConvertToMultiSigUser(ctx, ConvertToMultiSigUserAction{Signers: &MultiSigSigners{AuthorizedUsers: []Address{other, sub}, Threshold: 1}})
		}},
		{"convertToMultiSigUser/null", func() error { return c.ConvertToMultiSigUser(ctx, ConvertToMultiSigUserAction{}) }},
		// The signer leads and authorizes its own account. A multi-sig
		// account needs funds to create, so the exchange stops at "Invalid
		// multi-sig user" (whatever the signatures): this only proves it
		// deserialized the multiSig wrapper and the inner action.
		{"multiSig/l1", multiSig(OrderAction{Orders: []Order{ord}})},
		{"multiSig/userSigned", multiSig(USDSendAction{Destination: other, Amount: "1"})},

		// Spot deploy.
		{"spotDeploy/registerToken2", spot(SpotDeployAction{RegisterToken2: &RegisterToken2{Spec: TokenSpec{Name: "ZZTEST", SzDecimals: 2, WeiDecimals: 8}, MaxGas: gas, FullName: "Z"}})},
		{"spotDeploy/userGenesis", spot(SpotDeployAction{UserGenesis: &UserGenesis{
			Token: 1, UserAndWei: []UserWei{{other, "1000"}},
			ExistingTokenAndWei: []ExistingTokenWei{{0, "10"}}, BlacklistUsers: []BlacklistUser{{other, false}},
		}})},
		{"spotDeploy/genesis", spot(SpotDeployAction{Genesis: &SpotGenesis{Token: 1, MaxSupply: "1000", NoHyperliquidity: true}})},
		{"spotDeploy/registerSpot", spot(SpotDeployAction{RegisterSpot: &RegisterSpot{Tokens: [2]int{1, 0}}})},
		{"spotDeploy/registerHyperliquidity", spot(SpotDeployAction{RegisterHyperliquidity: &RegisterHyperliquidity{Spot: 1, StartPx: "1", OrderSz: "1", NOrders: 1, NSeededLevels: &three}})},
		{"spotDeploy/setDeployerTradingFeeShare", spot(SpotDeployAction{SetDeployerTradingFeeShare: &SetDeployerTradingFeeShare{Token: 1, Share: "50"}})},
		{"spotDeploy/enableQuoteToken", spot(SpotDeployAction{EnableQuoteToken: &SpotDeployToken{Token: 1}})},
		{"spotDeploy/disableQuoteToken", spot(SpotDeployAction{DisableQuoteToken: &SpotDeployToken{Token: 1}})},
		{"spotDeploy/enableFreezePrivilege", spot(SpotDeployAction{EnableFreezePrivilege: &SpotDeployToken{Token: 1}})},
		{"spotDeploy/freezeUser", spot(SpotDeployAction{FreezeUser: &FreezeUser{Token: 1, User: other, Freeze: true}})},
		{"spotDeploy/revokeFreezePrivilege", spot(SpotDeployAction{RevokeFreezePrivilege: &SpotDeployToken{Token: 1}})},
		{"spotDeploy/requestEvmContract", spot(SpotDeployAction{RequestEVMContract: &RequestEVMContract{Token: 1, Address: other, EVMExtraWeiDecimals: -2}})},
		{"spotDeploy/setTokenAnnotation", spot(SpotDeployAction{SetTokenAnnotation: &SetTokenAnnotation{
			Token:      1,
			Annotation: TokenAnnotation{Category: "c", Description: "d", DisplayName: str("Z"), Keywords: []string{"k"}},
		}})},
		{"spotDeploy/setDeployerLabel", spot(SpotDeployAction{SetDeployerLabel: &SetDeployerLabel{Label: "zz"}})},
		{"finalizeEvmContract/create", func() error {
			return c.FinalizeEVMContract(ctx, FinalizeEVMContractAction{Token: 1, Input: FinalizeCreate(0)})
		}},
		{"finalizeEvmContract/firstStorageSlot", func() error {
			return c.FinalizeEVMContract(ctx, FinalizeEVMContractAction{Token: 1, Input: FinalizeFirstStorageSlot})
		}},
		{"finalizeEvmContract/customStorageSlot", func() error {
			return c.FinalizeEVMContract(ctx, FinalizeEVMContractAction{Token: 1, Input: FinalizeCustomStorageSlot})
		}},
		{"authorizeAqav2Role", func() error {
			return c.AuthorizeAQAv2Role(ctx, AuthorizeAQAv2RoleAction{Token: 1, Role: AQAv2RoleTechnical})
		}},

		// Perp deploy.
		{"perpDeploy/registerAsset", perp(PerpDeployAction{RegisterAsset: &RegisterPerpAsset{
			MaxGas:       &gas,
			AssetRequest: PerpAssetRequest{Coin: "zzt:ABC", SzDecimals: 2, OraclePx: "10", MarginTableID: 10},
			Dex:          "zzt", Schema: &PerpDexSchema{FullName: "zz", OracleUpdater: &other},
		}})},
		{"perpDeploy/registerAsset2", perp(PerpDeployAction{RegisterAsset2: &RegisterPerpAsset2{
			AssetRequest: PerpAssetRequest2{Coin: "zzt:ABC", SzDecimals: 2, OraclePx: "10", MarginTableID: 10, MarginMode: MarginModeNoCross}, Dex: "zzt",
		}})},
		{"perpDeploy/setOracle", perp(PerpDeployAction{SetOracle: &SetOracle{
			Dex: "zzt", OraclePxs: TupleMap[string, Decimal]{"zzt:B": "2", "zzt:A": "1"},
			MarkPxs: []TupleMap[string, Decimal]{{"zzt:A": "1"}}, ExternalPerpPxs: TupleMap[string, Decimal]{"zzt:A": "1"},
		}})},
		{"perpDeploy/setFundingMultipliers", perp(PerpDeployAction{SetFundingMultipliers: TupleMap[string, Decimal]{"zzt:A": "1"}})},
		{"perpDeploy/setFundingInterestRates", perp(PerpDeployAction{SetFundingInterestRates: TupleMap[string, Decimal]{"zzt:A": "-0.001"}})},
		{"perpDeploy/setFundingClamps", perp(PerpDeployAction{SetFundingClamps: TupleMap[string, Decimal]{"zzt:A": "0.001"}})},
		{"perpDeploy/haltTrading", perp(PerpDeployAction{HaltTrading: &HaltTrading{Coin: "zzt:A", IsHalted: true}})},
		{"perpDeploy/setMarginTableIds", perp(PerpDeployAction{SetMarginTableIDs: TupleMap[string, int]{"zzt:A": 10}})},
		{"perpDeploy/insertMarginTable", perp(PerpDeployAction{InsertMarginTable: &InsertMarginTable{
			Dex:         "zzt",
			MarginTable: MarginTableSpec{Description: "t", MarginTiers: []MarginTierSpec{{LowerBound: 0, MaxLeverage: 10}}},
		}})},
		{"perpDeploy/setFeeRecipient", perp(PerpDeployAction{SetFeeRecipient: &SetFeeRecipient{Dex: "zzt", FeeRecipient: other}})},
		{"perpDeploy/setOpenInterestCaps", perp(PerpDeployAction{SetOpenInterestCaps: TupleMap[string, *uint64]{"zzt:A": &oiCap, "zzt:B": nil}})},
		{"perpDeploy/setSubDeployers", perp(PerpDeployAction{SetSubDeployers: &SetSubDeployers{
			Dex:          "zzt",
			SubDeployers: []SubDeployer{{Variant: "setOracle", User: other, Allowed: true}},
		}})},
		{"perpDeploy/setMarginModes", perp(PerpDeployAction{SetMarginModes: TupleMap[string, MarginMode]{"zzt:A": MarginModeStrictIsolated}})},
		{"perpDeploy/setDeployerFees", perp(PerpDeployAction{SetDeployerFees: TupleMap[string, DeployerFee]{"zzt:A": {Scale: "1.5", GrowthMode: true}}})},
		{"perpDeploy/setPerpAnnotation", perp(PerpDeployAction{SetPerpAnnotation: &SetPerpAnnotation{
			Coin: "zzt:A", Category: "c",
			Description: "d", DisplayName: str("A"), Keywords: []string{"k"},
		}})},
		{"perpDeploy/disableDex", perp(PerpDeployAction{DisableDex: "zzt"})},
		{"hip3LiquidatorTransfer", func() error {
			return c.HIP3LiquidatorTransfer(ctx, HIP3LiquidatorTransferAction{Dex: "zzt", Ntl: 1000000000, IsDeposit: true})
		}},

		// Outcome deploy.
		{"activateOutcomeDeployer/activate", func() error {
			return c.ActivateOutcomeDeployer(ctx, ActivateOutcomeDeployerAction{Activate: &OutcomeDeployerActivation{VenueName: "zz"}})
		}},
		{"activateOutcomeDeployer/deactivate", func() error {
			return c.ActivateOutcomeDeployer(ctx, ActivateOutcomeDeployerAction{Deactivate: true})
		}},
		{"outcomeDeploy/registerStandaloneOutcomeFromTemplate", outcome(OutcomeOperation{RegisterStandaloneOutcomeFromTemplate: &tmpl})},
		{"outcomeDeploy/registerQuestionFromTemplate", outcome(OutcomeOperation{RegisterQuestionFromTemplate: &RegisterQuestionFromTemplate{
			QuestionTemplateInstance: tmpl, NamedOutcomeTemplateInstances: []OutcomeTemplateInstance{named, {ID: "o"}},
		}})},
		{"outcomeDeploy/registerAndAssociateNamedOutcomeFromTemplate", outcome(OutcomeOperation{
			RegisterAndAssociateNamedOutcomeFromTemplate: &RegisterNamedOutcome{Question: 3, NamedOutcomeTemplateInstance: named},
		})},
		{"outcomeDeploy/settleOutcome", outcome(OutcomeOperation{SettleOutcome: &settle})},
		{"outcomeDeploy/settleQuestion2", outcome(OutcomeOperation{SettleQuestion2: &SettleQuestion{
			Question:           1,
			OutcomeSettlements: []OutcomeSettlement{settle}, NameAndDescription: [2]string{"q", "d"},
		}})},
		{"outcomeDeploy/setSubDeployers", outcome(OutcomeOperation{SetSubDeployers: []SubDeployer{{Variant: "settleOutcome", User: other, Allowed: true}}})},

		// Validators.
		{"CSignerAction/jailSelf", func() error { return c.CSigner(ctx, CSignerAction{JailSelf: true}) }},
		{"CSignerAction/unjailSelf", func() error { return c.CSigner(ctx, CSignerAction{UnjailSelf: true}) }},
		{"CValidatorAction/register", func() error {
			return c.CValidator(ctx, CValidatorAction{Register: &ValidatorRegistration{Profile: ValidatorProfile{
				NodeIP: NodeIP{IP: "1.2.3.4"},
				Name:   "v", Description: "d", CommissionBps: bps, Signer: other,
			}, Unjailed: true, InitialWei: 1000}})
		}},
		{"CValidatorAction/changeProfile", func() error {
			return c.CValidator(ctx, CValidatorAction{ChangeProfile: &ValidatorProfileChange{
				NodeIP: &NodeIP{IP: "1.2.3.4"}, Name: str("v"),
				Unjailed: true, DisableDelegations: &no, CommissionBps: &bps,
			}})
		}},
		{"CValidatorAction/unregister", func() error { return c.CValidator(ctx, CValidatorAction{Unregister: true}) }},
		{"validatorL1Stream", func() error { return c.ValidatorL1Stream(ctx, ValidatorL1StreamAction{RiskFreeRate: "0.05"}) }},
		{"gossipPriorityBid", func() error {
			return c.GossipPriorityBid(ctx, GossipPriorityBidAction{SlotID: 0, IP: "1.2.3.4", MaxGas: gas})
		}},
	}

	addr := strings.ToLower(me.String())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			want := addr
			if strings.HasPrefix(tc.name, "multiSig/") {
				want = "invalid multi-sig user"
			}
			err := tc.f()
			var apiErr *APIError
			if !errors.As(err, &apiErr) || !strings.Contains(strings.ToLower(apiErr.Message), want) {
				t.Fatalf("want an API error containing %q, got %v", want, err)
			}
			t.Log(apiErr.Message)
		})
	}
}
