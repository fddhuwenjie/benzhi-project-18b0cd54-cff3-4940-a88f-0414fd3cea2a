package audit

import (
	"testing"
	"time"
)

func TestChainDetectsTampering(t *testing.T) {
	var chain Chain
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if _, err := chain.Append("b1", "batch.created", 1, at, map[string]string{"site": "A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := chain.Append("b1", "plan.frozen", 2, at, map[string]string{"seal": "double"}); err != nil {
		t.Fatal(err)
	}
	if err := chain.Verify(); err != nil {
		t.Fatalf("有效链未通过: %v", err)
	}
	chain.Events[0].PayloadHash = "tampered"
	if err := chain.Verify(); err == nil {
		t.Fatal("篡改后应校验失败")
	}
}
