package hyperliquid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"
)

// retryTransport retries transient failures: HTTP 429 and 5xx responses and
// network errors. The API's per-IP rate limit resets every minute, so 429s
// back off longer.
type retryTransport struct{ tr transport }

func (r retryTransport) request(ctx context.Context, endpoint string, body []byte) (json.RawMessage, error) {
	const attempts = 5
	for attempt := 1; ; attempt++ {
		raw, err := r.tr.request(ctx, endpoint, body)
		var apiErr *APIError
		wait := 3 * time.Second
		switch {
		case err == nil || attempt == attempts || ctx.Err() != nil:
			return raw, err
		case errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusTooManyRequests:
			wait = 10 * time.Second
		case errors.As(err, &apiErr) && apiErr.StatusCode < 500:
			return raw, err
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(attempt) * wait):
		}
	}
}

// richUser had sub-accounts, agents, approved builders and borrow/lend
// positions when the fixtures were recorded; if that changes, those
// endpoints still decode, just emptier.
var richUser = MustParseAddress("0x58859ef6937084f88f5d8bb087fdb1cc041ba60a")

// infoArgs are live Mainnet arguments that make the info endpoints return
// rich responses. The fixture test replays with the zero value.
type infoArgs struct {
	now      int64   // milliseconds since the Unix epoch
	trader   Address // buyer of the latest BTC trade
	leader   Address // HLP's leader
	follower Address // an HLP depositor
	hlpChild Address // an HLP child vault, a busy market maker
	staker   Address // a validator, which self-delegates
	deployer Address // a spot token deployer
	builder  Address // a builder richUser approved
	dex      string  // a HIP-3 perp dex
	coin     string  // an annotated perp
	tokenID  string  // HYPE
	margin   int     // BTC's margin table
	oid      int64   // one of the trader's orders
	hash     string  // one of the trader's transactions
	height   int64   // its block
}

// infoCase checks one InfoClient method. name is the wire type, which also
// names the fixture under testdata/fixtures/<kind>.
type infoCase struct {
	name string
	kind string // "info", "explorer", or "" for calls spanning several endpoints
	call func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error)
}

const msPerDay = int64(24 * time.Hour / time.Millisecond)

var infoCases = []infoCase{
	{"activeAssetData", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.ActiveAssetData(ctx, ActiveAssetDataRequest{User: a.trader, Coin: "BTC"})
	}},
	{"allBorrowLendReserveStates", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.AllBorrowLendReserveStates(ctx)
	}},
	{"allMids", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.AllMids(ctx, AllMidsRequest{})
	}},
	{"allPerpMetas", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.AllPerpMetas(ctx)
	}},
	{"approvedBuilders", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.ApprovedBuilders(ctx, ApprovedBuildersRequest{User: richUser})
	}},
	{"borrowLendReserveState", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.BorrowLendReserveState(ctx, BorrowLendReserveStateRequest{Token: 0})
	}},
	{"borrowLendUserState", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.BorrowLendUserState(ctx, BorrowLendUserStateRequest{User: richUser})
	}},
	{"candleSnapshot", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.CandleSnapshot(ctx, CandleSnapshotRequest{Coin: "BTC", Interval: Candle1h, StartTime: a.now - msPerDay})
	}},
	{"clearinghouseState", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.ClearinghouseState(ctx, ClearinghouseStateRequest{User: a.hlpChild})
	}},
	{"delegations", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.Delegations(ctx, DelegationsRequest{User: a.staker})
	}},
	{"delegatorHistory", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.DelegatorHistory(ctx, DelegatorHistoryRequest{User: a.staker})
	}},
	{"delegatorRewards", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.DelegatorRewards(ctx, DelegatorRewardsRequest{User: a.staker})
	}},
	{"delegatorSummary", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.DelegatorSummary(ctx, DelegatorSummaryRequest{User: a.staker})
	}},
	{"exchangeStatus", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.ExchangeStatus(ctx)
	}},
	{"extraAgents", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.ExtraAgents(ctx, ExtraAgentsRequest{User: richUser})
	}},
	{"frontendOpenOrders", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.FrontendOpenOrders(ctx, FrontendOpenOrdersRequest{User: a.hlpChild})
	}},
	{"fundingHistory", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.FundingHistory(ctx, FundingHistoryRequest{Coin: "ETH", StartTime: a.now - msPerDay})
	}},
	{"gossipPriorityAuctionStatus", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.GossipPriorityAuctionStatus(ctx)
	}},
	{"gossipRootIps", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.GossipRootIPs(ctx)
	}},
	{"historicalOrders", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.HistoricalOrders(ctx, HistoricalOrdersRequest{User: a.trader})
	}},
	{"isVip", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.IsVIP(ctx, IsVIPRequest{User: a.trader})
	}},
	{"l2Book", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.L2Book(ctx, L2BookRequest{Coin: "BTC"})
	}},
	{"leadingVaults", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.LeadingVaults(ctx, LeadingVaultsRequest{User: a.leader})
	}},
	{"legalCheck", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.LegalCheck(ctx, LegalCheckRequest{User: a.trader})
	}},
	{"liquidatable", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.Liquidatable(ctx)
	}},
	{"marginTable", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.MarginTable(ctx, MarginTableRequest{ID: a.margin})
	}},
	{"maxBuilderFee", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.MaxBuilderFee(ctx, MaxBuilderFeeRequest{User: richUser, Builder: a.builder})
	}},
	{"maxMarketOrderNtls", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.MaxMarketOrderNtls(ctx)
	}},
	{"meta", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.Meta(ctx, MetaRequest{Dex: a.dex})
	}},
	{"metaAndAssetCtxs", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.MetaAndAssetCtxs(ctx, MetaAndAssetCtxsRequest{})
	}},
	{"openOrders", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.OpenOrders(ctx, OpenOrdersRequest{User: a.hlpChild})
	}},
	{"orderStatus", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.OrderStatus(ctx, OrderStatusRequest{User: a.trader, Oid: OrderRef{Oid: a.oid}})
	}},
	{"outcomeMeta", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.OutcomeMeta(ctx)
	}},
	{"outcomeTemplates", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.OutcomeTemplates(ctx)
	}},
	{"perpAnnotation", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.PerpAnnotation(ctx, PerpAnnotationRequest{Coin: a.coin})
	}},
	{"perpCategories", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PerpCategories(ctx)
	}},
	{"perpConciseAnnotations", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PerpConciseAnnotations(ctx)
	}},
	{"perpDeployAuctionStatus", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PerpDeployAuctionStatus(ctx)
	}},
	{"perpDexLimits", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.PerpDexLimits(ctx, PerpDexLimitsRequest{Dex: a.dex})
	}},
	{"perpDexStatus", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.PerpDexStatus(ctx, PerpDexStatusRequest{Dex: a.dex})
	}},
	{"perpDexs", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PerpDexs(ctx)
	}},
	{"perpsAtOpenInterestCap", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PerpsAtOpenInterestCap(ctx, PerpsAtOpenInterestCapRequest{})
	}},
	{"portfolio", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.Portfolio(ctx, PortfolioRequest{User: a.trader})
	}},
	{"preTransferCheck", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.PreTransferCheck(ctx, PreTransferCheckRequest{User: a.trader, Source: hlpVault})
	}},
	{"predictedFundings", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.PredictedFundings(ctx)
	}},
	{"recentTrades", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.RecentTrades(ctx, RecentTradesRequest{Coin: "@107"})
	}},
	{"referral", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.Referral(ctx, ReferralRequest{User: a.trader})
	}},
	{"settledOutcome", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		// Outcome 100 settled a question, which fills every field.
		return c.SettledOutcome(ctx, SettledOutcomeRequest{Outcome: 100})
	}},
	{"spotClearinghouseState", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.SpotClearinghouseState(ctx, SpotClearinghouseStateRequest{User: a.trader})
	}},
	{"spotDeployState", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.SpotDeployState(ctx, SpotDeployStateRequest{User: a.deployer})
	}},
	{"spotMeta", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.SpotMeta(ctx)
	}},
	{"spotMetaAndAssetCtxs", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.SpotMetaAndAssetCtxs(ctx)
	}},
	{"spotPairDeployAuctionStatus", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.SpotPairDeployAuctionStatus(ctx)
	}},
	{"subAccounts", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.SubAccounts(ctx, SubAccountsRequest{User: richUser})
	}},
	{"subAccounts2", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.SubAccounts2(ctx, SubAccounts2Request{User: richUser})
	}},
	{"tokenDetails", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.TokenDetails(ctx, TokenDetailsRequest{TokenID: a.tokenID})
	}},
	{"twapHistory", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.TwapHistory(ctx, TwapHistoryRequest{User: a.trader})
	}},
	{"usdcRouting", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.USDCRouting(ctx)
	}},
	{"userAbstraction", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserAbstraction(ctx, UserAbstractionRequest{User: a.trader})
	}},
	{"userBorrowLendInterest", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserBorrowLendInterest(ctx, UserBorrowLendInterestRequest{User: richUser, StartTime: a.now - 30*msPerDay})
	}},
	{"userDexAbstraction", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserDexAbstraction(ctx, UserDexAbstractionRequest{User: a.trader})
	}},
	{"userFees", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserFees(ctx, UserFeesRequest{User: a.trader})
	}},
	{"userFills", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserFills(ctx, UserFillsRequest{User: a.trader})
	}},
	{"userFillsByTime", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserFillsByTime(ctx, UserFillsByTimeRequest{User: a.trader, StartTime: a.now - 30*msPerDay, Reversed: true})
	}},
	{"userFunding", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserFunding(ctx, UserFundingRequest{User: a.hlpChild, StartTime: a.now - msPerDay})
	}},
	{"userNonFundingLedgerUpdates", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserNonFundingLedgerUpdates(ctx, UserNonFundingLedgerUpdatesRequest{User: hlpVault, StartTime: a.now - 7*msPerDay})
	}},
	{"userRateLimit", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserRateLimit(ctx, UserRateLimitRequest{User: a.trader})
	}},
	{"userRole", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserRole(ctx, UserRoleRequest{User: a.hlpChild})
	}},
	{"userToMultiSigSigners", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserToMultiSigSigners(ctx, UserToMultiSigSignersRequest{User: a.trader})
	}},
	{"userTwapSliceFills", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserTwapSliceFills(ctx, UserTwapSliceFillsRequest{User: a.trader})
	}},
	{"userTwapSliceFillsByTime", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserTwapSliceFillsByTime(ctx, UserTwapSliceFillsByTimeRequest{User: a.trader, StartTime: a.now - 30*msPerDay})
	}},
	{"userVaultEquities", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserVaultEquities(ctx, UserVaultEquitiesRequest{User: a.follower})
	}},
	{"validatorL1Votes", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.ValidatorL1Votes(ctx)
	}},
	{"validatorSummaries", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.ValidatorSummaries(ctx)
	}},
	{"vaultDetails", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.VaultDetails(ctx, VaultDetailsRequest{VaultAddress: hlpVault, User: &a.follower})
	}},
	{"vaultSummaries", "info", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return c.VaultSummaries(ctx)
	}},
	{"webData2", "info", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.WebData2(ctx, WebData2Request{User: a.trader})
	}},
	{"blockDetails", "explorer", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.BlockDetails(ctx, BlockDetailsRequest{Height: a.height})
	}},
	{"txDetails", "explorer", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.TxDetails(ctx, TxDetailsRequest{Hash: a.hash})
	}},
	{"userDetails", "explorer", func(ctx context.Context, c *InfoClient, a *infoArgs) (any, error) {
		return c.UserDetails(ctx, UserDetailsRequest{User: a.trader})
	}},
	// LoadMarkets replays the perpDexs, allPerpMetas and spotMeta fixtures.
	{"LoadMarkets", "", func(ctx context.Context, c *InfoClient, _ *infoArgs) (any, error) {
		return LoadMarkets(ctx, c)
	}},
}

func TestLiveInfo(t *testing.T) {
	requireLive(t)
	c := NewInfoClient(Mainnet)
	rec := map[string]*recordingTransport{
		"info":     {tr: retryTransport{c.tr}},
		"explorer": {tr: retryTransport{c.explorer}},
	}
	c.tr, c.explorer = rec["info"], rec["explorer"]
	a := liveInfoArgs(t, c)
	for _, tc := range infoCases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.call(t.Context(), c, a); err != nil {
				t.Fatal(err)
			}
			if r := rec[tc.kind]; r != nil && recording() {
				writeFixture(t, tc.kind, tc.name, r.last)
			}
		})
	}
}

// liveInfoArgs picks arguments from the current Mainnet state.
func liveInfoArgs(t *testing.T, c *InfoClient) *infoArgs {
	t.Helper()
	ctx := t.Context()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	a := &infoArgs{now: time.Now().UnixMilli(), trader: activeTrader(t, c)}

	vault, err := c.VaultDetails(ctx, VaultDetailsRequest{VaultAddress: hlpVault})
	must(err)
	a.leader = vault.Leader
	for _, f := range vault.Followers {
		if f.User != vault.Leader {
			a.follower = f.User
			break
		}
	}
	if vault.Relationship.Data == nil || len(vault.Relationship.Data.ChildAddresses) == 0 {
		t.Fatal("HLP has no child vaults")
	}
	a.hlpChild = vault.Relationship.Data.ChildAddresses[0]

	validators, err := c.ValidatorSummaries(ctx)
	must(err)
	if len(validators) == 0 {
		t.Fatal("no validators")
	}
	a.staker = validators[0].Validator

	dexs, err := c.PerpDexs(ctx)
	must(err)
	for _, d := range dexs {
		if d != nil {
			a.dex = d.Name
			break
		}
	}
	annotations, err := c.PerpConciseAnnotations(ctx)
	must(err)
	for coin := range annotations {
		a.coin = coin
		break
	}
	meta, err := c.Meta(ctx, MetaRequest{})
	must(err)
	a.margin = meta.Universe[0].MarginTableID

	spot, err := c.SpotMeta(ctx)
	must(err)
	for _, tok := range spot.Tokens {
		if tok.Name == "HYPE" {
			a.tokenID = tok.TokenID
		}
	}
	latest, err := c.TokenDetails(ctx, TokenDetailsRequest{TokenID: spot.Tokens[len(spot.Tokens)-1].TokenID})
	must(err)
	if latest.Deployer != nil {
		a.deployer = *latest.Deployer
	}

	orders, err := c.HistoricalOrders(ctx, HistoricalOrdersRequest{User: a.trader})
	must(err)
	if len(orders) > 0 {
		a.oid = orders[0].Order.Oid
	}
	builders, err := c.ApprovedBuilders(ctx, ApprovedBuildersRequest{User: richUser})
	must(err)
	if len(builders) > 0 {
		a.builder = builders[0]
	}
	txs, err := c.UserDetails(ctx, UserDetailsRequest{User: a.trader})
	must(err)
	if len(txs) == 0 {
		t.Fatalf("no explorer transactions for %s", a.trader)
	}
	a.hash, a.height = txs[0].Hash, txs[0].Block
	return a
}

// TestInfoFixtures replays the recorded responses through the same calls.
func TestInfoFixtures(t *testing.T) {
	for _, tc := range infoCases {
		t.Run(tc.name, func(t *testing.T) {
			c := &InfoClient{tr: fixtureTransport{t, "info"}, explorer: fixtureTransport{t, "explorer"}}
			if _, err := tc.call(t.Context(), c, &infoArgs{}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

// fixtureTransport answers each request with the fixture named by its type,
// so calls spanning several endpoints, like LoadMarkets, replay too.
type fixtureTransport struct {
	t    *testing.T
	kind string
}

func (f fixtureTransport) request(_ context.Context, _ string, body []byte) (json.RawMessage, error) {
	var req struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, err
	}
	return readFixture(f.t, f.kind, req.Type), nil
}
