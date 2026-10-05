package hyperliquid

import "testing"

// TestDeployActionEncoding checks deployer, validator and protocol actions
// against the JSON the Python SDK (or its examples) signs, and against the
// official API docs for actions Python lacks.
func TestDeployActionEncoding(t *testing.T) {
	user := MustParseAddress("0x5e9ee1089755c3435139848e47e6635505d5a13a")
	str := func(s string) *string { return &s }
	f, tr, oiCap := false, 3, uint64(1000000)
	for _, tc := range []struct {
		name string
		a    Action
		want string
	}{
		// Python SDK.
		{"spot registerToken2", SpotDeployAction{RegisterToken2: &RegisterToken2{
			Spec: TokenSpec{Name: "TEST", SzDecimals: 2, WeiDecimals: 8}, MaxGas: 100000000, FullName: "Test Token",
		}}, `{"type":"spotDeploy","registerToken2":{"spec":{"name":"TEST","szDecimals":2,"weiDecimals":8},"maxGas":100000000,"fullName":"Test Token"}}`},
		{"spot userGenesis", SpotDeployAction{UserGenesis: &UserGenesis{
			Token: 1234, UserAndWei: []UserWei{{user, "100000000"}}, ExistingTokenAndWei: []ExistingTokenWei{{1, "5000"}},
		}}, `{"type":"spotDeploy","userGenesis":{"token":1234,"userAndWei":[["0x5e9ee1089755c3435139848e47e6635505d5a13a","100000000"]],"existingTokenAndWei":[[1,"5000"]]}}`},
		{
			"spot enableFreezePrivilege",
			SpotDeployAction{EnableFreezePrivilege: &SpotDeployToken{1234}},
			`{"type":"spotDeploy","enableFreezePrivilege":{"token":1234}}`,
		},
		{
			"spot freezeUser",
			SpotDeployAction{FreezeUser: &FreezeUser{Token: 1234, User: user, Freeze: true}},
			`{"type":"spotDeploy","freezeUser":{"token":1234,"user":"0x5e9ee1089755c3435139848e47e6635505d5a13a","freeze":true}}`,
		},
		{
			"spot revokeFreezePrivilege",
			SpotDeployAction{RevokeFreezePrivilege: &SpotDeployToken{1234}},
			`{"type":"spotDeploy","revokeFreezePrivilege":{"token":1234}}`,
		},
		{
			"spot enableQuoteToken",
			SpotDeployAction{EnableQuoteToken: &SpotDeployToken{1234}},
			`{"type":"spotDeploy","enableQuoteToken":{"token":1234}}`,
		},
		{
			"spot genesis",
			SpotDeployAction{Genesis: &SpotGenesis{Token: 1234, MaxSupply: "10000000000", NoHyperliquidity: true}},
			`{"type":"spotDeploy","genesis":{"token":1234,"maxSupply":"10000000000","noHyperliquidity":true}}`,
		},
		{
			"spot registerSpot",
			SpotDeployAction{RegisterSpot: &RegisterSpot{Tokens: [2]int{1234, 0}}},
			`{"type":"spotDeploy","registerSpot":{"tokens":[1234,0]}}`,
		},
		{"spot registerHyperliquidity", SpotDeployAction{RegisterHyperliquidity: &RegisterHyperliquidity{
			Spot: 7, StartPx: "0.25", OrderSz: "12.5", NOrders: 40, NSeededLevels: &tr,
		}}, `{"type":"spotDeploy","registerHyperliquidity":{"spot":7,"startPx":"0.25","orderSz":"12.5","nOrders":40,"nSeededLevels":3}}`},
		{
			"spot setDeployerTradingFeeShare",
			SpotDeployAction{SetDeployerTradingFeeShare: &SetDeployerTradingFeeShare{Token: 1234, Share: "12.50"}},
			`{"type":"spotDeploy","setDeployerTradingFeeShare":{"token":1234,"share":"12.5%"}}`,
		},
		{"perp registerAsset", PerpDeployAction{RegisterAsset: &RegisterPerpAsset{
			AssetRequest: PerpAssetRequest{Coin: "test:ABC", SzDecimals: 2, OraclePx: "10.5", MarginTableID: 10},
			Dex:          "test",
			Schema:       &PerpDexSchema{FullName: "Test Dex", OracleUpdater: &user},
		}}, `{"type":"perpDeploy","registerAsset":{"maxGas":null,"assetRequest":{"coin":"test:ABC","szDecimals":2,"oraclePx":"10.5","marginTableId":10,"onlyIsolated":false},"dex":"test","schema":{"fullName":"Test Dex","collateralToken":0,"oracleUpdater":"0x5e9ee1089755c3435139848e47e6635505d5a13a"}}}`},
		{"perp setOracle", PerpDeployAction{SetOracle: &SetOracle{
			Dex:       "test",
			OraclePxs: TupleMap[string, Decimal]{"test:B": "2", "test:A": "1.5"},
			MarkPxs:   []TupleMap[string, Decimal]{{"test:B": "2.1", "test:A": "1.4"}},
		}}, `{"type":"perpDeploy","setOracle":{"dex":"test","oraclePxs":[["test:A","1.5"],["test:B","2"]],"markPxs":[[["test:A","1.4"],["test:B","2.1"]]],"externalPerpPxs":[]}}`},
		{"CSigner jailSelf", CSignerAction{JailSelf: true}, `{"type":"CSignerAction","jailSelf":null}`},
		{"CSigner unjailSelf", CSignerAction{UnjailSelf: true}, `{"type":"CSignerAction","unjailSelf":null}`},
		{"CValidator register", CValidatorAction{Register: &ValidatorRegistration{
			Profile:  ValidatorProfile{NodeIP: NodeIP{"1.2.3.4"}, Name: "val", Description: "desc", CommissionBps: 500, Signer: user},
			Unjailed: true, InitialWei: 1000000000,
		}}, `{"type":"CValidatorAction","register":{"profile":{"node_ip":{"Ip":"1.2.3.4"},"name":"val","description":"desc","delegations_disabled":false,"commission_bps":500,"signer":"0x5e9ee1089755c3435139848e47e6635505d5a13a"},"unjailed":true,"initial_wei":1000000000}}`},
		{"CValidator changeProfile", CValidatorAction{ChangeProfile: &ValidatorProfileChange{
			Name: str("val2"), Unjailed: true, DisableDelegations: &f, Signer: &user,
		}}, `{"type":"CValidatorAction","changeProfile":{"node_ip":null,"name":"val2","description":null,"unjailed":true,"disable_delegations":false,"commission_bps":null,"signer":"0x5e9ee1089755c3435139848e47e6635505d5a13a"}}`},
		{"CValidator unregister", CValidatorAction{Unregister: true}, `{"type":"CValidatorAction","unregister":null}`},
		{
			"gossipPriorityBid",
			GossipPriorityBidAction{SlotID: 1, IP: "1.2.3.4", MaxGas: 100000000},
			`{"type":"gossipPriorityBid","slotId":1,"ip":"1.2.3.4","maxGas":100000000}`,
		},
		// Python SDK examples/evm_erc20.py.
		{
			"spot requestEvmContract",
			SpotDeployAction{RequestEVMContract: &RequestEVMContract{Token: 1234, Address: user, EVMExtraWeiDecimals: 13}},
			`{"type":"spotDeploy","requestEvmContract":{"token":1234,"address":"0x5e9ee1089755c3435139848e47e6635505d5a13a","evmExtraWeiDecimals":13}}`,
		},
		{
			"finalizeEvmContract create",
			FinalizeEVMContractAction{Token: 1234, Input: FinalizeCreate(0)},
			`{"type":"finalizeEvmContract","token":1234,"input":{"create":{"nonce":0}}}`,
		},
		{
			"finalizeEvmContract slot",
			FinalizeEVMContractAction{Token: 1234, Input: FinalizeFirstStorageSlot},
			`{"type":"finalizeEvmContract","token":1234,"input":"firstStorageSlot"}`,
		},
		// Official API docs and nktkas schemas.
		{"spot userGenesis empty lists", SpotDeployAction{UserGenesis: &UserGenesis{
			Token: 1, BlacklistUsers: []BlacklistUser{{user, true}},
		}}, `{"type":"spotDeploy","userGenesis":{"token":1,"userAndWei":[],"existingTokenAndWei":[],"blacklistUsers":[["0x5e9ee1089755c3435139848e47e6635505d5a13a",true]]}}`},
		{
			"spot setTokenAnnotation",
			SpotDeployAction{SetTokenAnnotation: &SetTokenAnnotation{Token: 1, Annotation: TokenAnnotation{Category: "c", Description: "d"}}},
			`{"type":"spotDeploy","setTokenAnnotation":{"token":1,"annotation":{"category":"c","description":"d","displayName":null,"keywords":[]}}}`,
		},
		{
			"perp setMarginModes",
			PerpDeployAction{SetMarginModes: TupleMap[string, MarginMode]{"x:B": MarginModeNoCross, "x:A": MarginModeStrictIsolated}},
			`{"type":"perpDeploy","setMarginModes":[["x:A","strictIsolated"],["x:B","noCross"]]}`,
		},
		{
			"perp setOpenInterestCaps",
			PerpDeployAction{SetOpenInterestCaps: TupleMap[string, *uint64]{"x:B": nil, "x:A": &oiCap}},
			`{"type":"perpDeploy","setOpenInterestCaps":[["x:A",1000000],["x:B",null]]}`,
		},
		{
			"perp setDeployerFees",
			PerpDeployAction{SetDeployerFees: TupleMap[string, DeployerFee]{"x:A": {Scale: "1.50", GrowthMode: true}}},
			`{"type":"perpDeploy","setDeployerFees":[["x:A",{"scale":"1.5","growthMode":true}]]}`,
		},
		{
			"perp setPerpAnnotation",
			PerpDeployAction{SetPerpAnnotation: &SetPerpAnnotation{Coin: "x:A", Category: "c", Description: "d"}},
			`{"type":"perpDeploy","setPerpAnnotation":{"coin":"x:A","category":"c","description":"d","displayName":null,"keywords":[]}}`,
		},
		{
			"activateOutcomeDeployer activate",
			ActivateOutcomeDeployerAction{Activate: &OutcomeDeployerActivation{VenueName: "ab"}},
			`{"type":"activateOutcomeDeployer","activate":{"venueName":"ab"}}`,
		},
		{
			"activateOutcomeDeployer deactivate",
			ActivateOutcomeDeployerAction{Deactivate: true},
			`{"type":"activateOutcomeDeployer","deactivate":null}`,
		},
		{"outcomeDeploy registerQuestionFromTemplate", OutcomeDeployAction{Venue: "ab", Operation: OutcomeOperation{
			RegisterQuestionFromTemplate: &RegisterQuestionFromTemplate{
				QuestionTemplateInstance: OutcomeTemplateInstance{ID: "abc", KeywordToValue: TupleMap[string, string]{"expiry": "20260801-1830"}, DeployerFeeScale: "1"},
				NamedOutcomeTemplateInstances: []OutcomeTemplateInstance{
					{ID: "abc-outcome", KeywordToValue: TupleMap[string, string]{"choice": "A"}},
					{ID: "abc-other"},
				},
			},
		}}, `{"type":"outcomeDeploy","venue":"ab","operation":{"registerQuestionFromTemplate":{"questionTemplateInstance":{"id":"abc","keywordToValue":[["expiry","20260801-1830"]],"deployerFeeScale":"1"},"namedOutcomeTemplateInstances":[{"id":"abc-outcome","keywordToValue":[["choice","A"]]},{"id":"abc-other","keywordToValue":[]}]}}}`},
	} {
		got, err := encodeAction(tc.a, 1700000000000, Testnet)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if g := compact(t, got); g != tc.want {
			t.Errorf("%s:\n got %s\nwant %s", tc.name, g, tc.want)
		}
	}
	if _, err := encodeAction(CSignerAction{}, 1, Testnet); err == nil {
		t.Error("CSignerAction{}: want error")
	}
}
