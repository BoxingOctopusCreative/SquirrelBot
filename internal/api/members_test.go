package api

import "testing"

func TestMemberMatches(t *testing.T) {
	if !memberMatches("Ryan", "ryandraga", "") {
		t.Fatal("empty query matches")
	}
	if !memberMatches("Ryan", "ryandraga", "RY") {
		t.Fatal("display name")
	}
	if !memberMatches("Ryan", "ryandraga", "draga") {
		t.Fatal("username")
	}
	if memberMatches("Ryan", "ryandraga", "other") {
		t.Fatal("unrelated query")
	}
}
