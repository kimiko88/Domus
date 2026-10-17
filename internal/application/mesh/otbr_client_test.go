package mesh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOTBRClient_GetTopology_WithMockServer(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/node/state", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"leader"`))
	})
	mux.HandleFunc("/node/rloc16", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"0x2800"`))
	})
	mux.HandleFunc("/node/ext-address", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"f4ce3685e19a421b"`))
	})
	mux.HandleFunc("/node/network-name", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`"DomusThreadTest"`))
	})
	mux.HandleFunc("/node/neighbor-table", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"ext_address":"00124b0024a18b90","rloc16":"0x2801","link_quality_in":3,"average_rssi":-45,"is_child":false},
			{"ext_address":"00124b0024a19c11","rloc16":"0x2802","link_quality_in":2,"average_rssi":-65,"is_child":true}
		]`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := NewOTBRClient(server.URL)
	topo, err := client.GetTopology(context.Background())
	if err != nil {
		t.Fatalf("expected successful topology fetch, got error: %v", err)
	}

	if topo.NetworkName != "DomusThreadTest" {
		t.Errorf("expected network name 'DomusThreadTest', got '%s'", topo.NetworkName)
	}
	if topo.LocalRole != ThreadRoleLeader {
		t.Errorf("expected role 'leader', got '%s'", topo.LocalRole)
	}
	if topo.NodesCount != 3 { // 2 neighbors + 1 local node
		t.Errorf("expected 3 nodes, got %d", topo.NodesCount)
	}
	if topo.RoutersCount != 1 || topo.ChildrenCount != 1 {
		t.Errorf("expected 1 router and 1 child, got %d routers and %d children", topo.RoutersCount, topo.ChildrenCount)
	}
}

func TestOTBRClient_Fallback(t *testing.T) {
	// Point to unreachable server to test fallback
	client := NewOTBRClient("http://127.0.0.1:59999")
	topo, err := client.GetTopology(context.Background())
	if err != nil {
		t.Fatalf("expected fallback topology without error, got %v", err)
	}
	if topo.NetworkName == "" || len(topo.Neighbors) == 0 {
		t.Errorf("expected non-empty fallback topology, got %+v", topo)
	}
}
