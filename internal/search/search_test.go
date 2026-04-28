package search

import "testing"

func TestSearchRanksTitleMatchesHigher(t *testing.T) {
	idx := NewIndex([]Document{
		{Route: "/a", Title: "Ledger API", Body: "protobuf status"},
		{Route: "/b", Title: "Other", Body: "ledger ledger ledger"},
	})
	results := idx.Search("ledger")
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Route != "/a" {
		t.Fatalf("first route = %q, want /a", results[0].Route)
	}
}

func TestSearchIgnoresPunctuationAndCase(t *testing.T) {
	idx := NewIndex([]Document{{Route: "/a", Title: "JSON-RPC", Body: "Wallet Gateway"}})
	results := idx.Search("json rpc")
	if len(results) != 1 || results[0].Route != "/a" {
		t.Fatalf("results = %+v", results)
	}
}
