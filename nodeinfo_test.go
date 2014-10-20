package murcott

import (
	"sort"
	"testing"

	"github.com/vmihailenco/msgpack"
)

func TestNodeInfoMsgpack(t *testing.T) {
	info := nodeInfo{}
	data, err := msgpack.Marshal(info)
	if err != nil {
		t.Errorf("cannot marshal nodeInfo")
	}
	var info2 nodeInfo
	err = msgpack.Unmarshal(data, &info2)
	if err != nil {
		t.Errorf("cannot unmarshal nodeInfo")
	}
	if info.ID.cmp(info2.ID) != 0 {
		t.Errorf("node2.ID must be equal to node.ID")
	}
	if info.Addr.String() != info2.Addr.String() {
		t.Errorf("node2.Addr must be equal to node.Addr")
	}
}

func TestNodeInfoSort(t *testing.T) {
	ids := []string{
		"2R2eoXNPEhbmhx7aNqgY1e2SdKrJ",
		"4cLuxzdqZgKCatw2HJqEoZEAhkdD",
		"4fmJMvhoXrmBrHdeZnQ5iX5ropm3",
		"4fqqyXWVWmBRnLUVHfZgzjKdtFcd",
		"218GStqPqa7iLzLsAQBS9eZRrUik",
		"2vm8ByjrLATzFR6qqEHCdwua6eCf",
		"3nvgbcBzvt9y9Uvf1AbwVfnqV2RG",
		"33m8NJkskAUdCGw3uYAxeBD5jjBY",
		"3ru66Gjzx2cDuddRTzA47yMqEoLE",
		"2S68uiyhVt5c59zgXh1mj3v8vThp",
		"43eYKjPkMX3gqqWuzzBYvejLSQgJ",
		"2EhubMbxHHTHSdsLUuJmNpvRakt6",
		"hVrmqGmWDeRWcWVwTxMBEr1pszM",
		"3Fx6deQbP8arwtVAxbcts5d9KaTw",
		"36gDQzwABf2bscJwTjw9y2UU8Adg",
		"dT378JwadJ4h7HTgeh8UkMgAuVm",
		"37ZrjbsRymbaCD14mUu6FX3nHnPF",
		"3qtQPy3WCq3sx4vhGW1vR46aRRSo",
		"2Af1fPjeQ8jdtsHrrRxZCfJNBWGr",
		"3HJJyARx667UUrwoEDzCJAMx6tMg",
	}

	sorted := []string{
		"2R2eoXNPEhbmhx7aNqgY1e2SdKrJ",
		"2S68uiyhVt5c59zgXh1mj3v8vThp",
		"36gDQzwABf2bscJwTjw9y2UU8Adg",
		"33m8NJkskAUdCGw3uYAxeBD5jjBY",
		"37ZrjbsRymbaCD14mUu6FX3nHnPF",
		"3qtQPy3WCq3sx4vhGW1vR46aRRSo",
		"3ru66Gjzx2cDuddRTzA47yMqEoLE",
		"3nvgbcBzvt9y9Uvf1AbwVfnqV2RG",
		"3Fx6deQbP8arwtVAxbcts5d9KaTw",
		"3HJJyARx667UUrwoEDzCJAMx6tMg",
		"43eYKjPkMX3gqqWuzzBYvejLSQgJ",
		"4cLuxzdqZgKCatw2HJqEoZEAhkdD",
		"4fqqyXWVWmBRnLUVHfZgzjKdtFcd",
		"4fmJMvhoXrmBrHdeZnQ5iX5ropm3",
		"dT378JwadJ4h7HTgeh8UkMgAuVm",
		"hVrmqGmWDeRWcWVwTxMBEr1pszM",
		"218GStqPqa7iLzLsAQBS9eZRrUik",
		"2vm8ByjrLATzFR6qqEHCdwua6eCf",
		"2EhubMbxHHTHSdsLUuJmNpvRakt6",
		"2Af1fPjeQ8jdtsHrrRxZCfJNBWGr",
	}

	ary := make([]nodeInfo, len(ids))
	for i, _ := range ary {
		id, _ := NewNodeIDFromString(ids[i])
		ary[i] = nodeInfo{ID: id}
	}

	id, _ := NewNodeIDFromString(ids[0])
	sorter := nodeInfoSorter{nodes: ary, id: id}
	sort.Sort(sorter)

	for i, n := range sorter.nodes {
		id, _ := NewNodeIDFromString(sorted[i])
		if id.cmp(n.ID) != 0 {
			t.Errorf("sorter.nodes[%d] expects %s", i, sorted[i])
		}
	}
}
