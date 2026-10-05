package hyperliquid

import (
	"encoding/json"
	"testing"
)

func TestVaultDetailsResolvesLeaderFollower(t *testing.T) {
	const raw = `{"name":"V","vaultAddress":"0xdfc24b077bc1425ad1dea75bcb6f8158e10df303",
		"leader":"0x677d831aef5328190852e24f13c46cac05f984e7","description":"","portfolio":[],
		"apr":0.036,"followerState":null,"leaderFraction":0.0019,"leaderCommission":0,
		"followers":[
			{"user":"Leader","vaultEquity":"10.5","pnl":"1","allTimePnl":"2","daysFollowing":3,"vaultEntryTime":4,"lockupUntil":5},
			{"user":"0x03e161499870b0a37549b16a5d33e0582fe1255e","vaultEquity":"1","pnl":"0","allTimePnl":"0","daysFollowing":1,"vaultEntryTime":1,"lockupUntil":1}],
		"maxDistributable":41848682.429413,"maxWithdrawable":0.0,"isClosed":false,
		"relationship":{"type":"parent","data":{"childAddresses":["0x2e3d94f0562703b25c83308a05046ddaf9a8dd14"]}},
		"allowDeposits":true,"alwaysCloseOnWithdraw":false}`
	var d VaultDetails
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Followers) != 2 {
		t.Fatalf("got %d followers", len(d.Followers))
	}
	if d.Followers[0].User != d.Leader || d.Followers[0].VaultEquity != "10.5" || d.Followers[0].LockupUntil != 5 {
		t.Errorf("leader follower = %+v", d.Followers[0])
	}
	if d.Followers[1].User != MustParseAddress("0x03e161499870b0a37549b16a5d33e0582fe1255e") {
		t.Errorf("follower user = %v", d.Followers[1].User)
	}
	if d.MaxDistributable != "41848682.429413" || d.APR != "0.036" {
		t.Errorf("numeric fields = %q, %q", d.MaxDistributable, d.APR)
	}
	if d.Relationship.Data == nil || len(d.Relationship.Data.ChildAddresses) != 1 {
		t.Errorf("relationship = %+v", d.Relationship)
	}
}
