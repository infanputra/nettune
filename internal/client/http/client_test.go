package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jtsang4/nettune/internal/shared/types"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:9876", "test-key", 30*time.Second)

	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.baseURL != "http://localhost:9876" {
		t.Errorf("baseURL = %s, want http://localhost:9876", client.baseURL)
	}
	if client.apiKey != "test-key" {
		t.Errorf("apiKey = %s, want test-key", client.apiKey)
	}
	if client.defaultTimeout != 30*time.Second {
		t.Errorf("defaultTimeout = %v, want 30s", client.defaultTimeout)
	}
}

func TestClient_CalculateProbeTimeout(t *testing.T) {
	client := NewClient("http://localhost:9876", "test-key", 30*time.Second)

	tests := []struct {
		name     string
		bytes    int64
		expected time.Duration
	}{
		{"small transfer", 1000, 30 * time.Second},      // min timeout
		{"medium transfer", 10000000, 30 * time.Second}, // ~10MB at 0.5Mbps = ~160s, but with 2x factor and min check
		{"large transfer", 500000000, 10 * time.Minute}, // ~500MB, hits max
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeout := client.calculateProbeTimeout(tt.bytes)
			if timeout < 30*time.Second {
				t.Errorf("timeout = %v, should be at least 30s", timeout)
			}
			if timeout > 10*time.Minute {
				t.Errorf("timeout = %v, should be at most 10m", timeout)
			}
		})
	}
}

func TestClient_ProbeEcho(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		if r.URL.Path != "/probe/echo" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"server_time": time.Now().Format(time.RFC3339),
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	result, err := client.ProbeEcho()

	if err != nil {
		t.Fatalf("ProbeEcho failed: %v", err)
	}
	if result == nil {
		t.Fatal("ProbeEcho returned nil result")
	}
}

func TestClient_ProbeEcho_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "UNAUTHORIZED",
				"message": "invalid api key",
			},
		}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "wrong-key", 5*time.Second)
	_, err := client.ProbeEcho()

	if err == nil {
		t.Error("Expected error for unauthorized request")
	}
}

func TestClient_ListProfiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profiles" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"profiles": []map[string]interface{}{
					{"id": "bbr-fq-default", "name": "BBR + FQ", "risk_level": "low"},
					{"id": "low-latency", "name": "Low Latency", "risk_level": "medium"},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	profiles, err := client.ListProfiles()

	if err != nil {
		t.Fatalf("ListProfiles failed: %v", err)
	}
	if len(profiles) != 2 {
		t.Errorf("Expected 2 profiles, got %d", len(profiles))
	}
}

func TestClient_GetProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/profiles/bbr-fq-default" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":          "bbr-fq-default",
				"name":        "BBR + FQ (Conservative)",
				"description": "Enable BBR with FQ",
				"risk_level":  "low",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	profile, err := client.GetProfile("bbr-fq-default")

	if err != nil {
		t.Fatalf("GetProfile failed: %v", err)
	}
	if profile.ID != "bbr-fq-default" {
		t.Errorf("Profile ID = %s, want bbr-fq-default", profile.ID)
	}
}

func TestClient_GetProfile_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"success": false,
			"error": map[string]interface{}{
				"code":    "NOT_FOUND",
				"message": "profile not found",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	_, err := client.GetProfile("nonexistent")

	if err == nil {
		t.Error("Expected error for nonexistent profile")
	}
}

func TestClient_CreateProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/profiles" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var profile types.Profile
		if err := json.NewDecoder(r.Body).Decode(&profile); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"id":         profile.ID,
				"name":       profile.Name,
				"risk_level": profile.RiskLevel,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	profile := &types.Profile{
		ID:        "custom-profile",
		Name:      "Custom Profile",
		RiskLevel: "low",
	}

	result, err := client.CreateProfile(profile)

	if err != nil {
		t.Fatalf("CreateProfile failed: %v", err)
	}
	if result.ID != "custom-profile" {
		t.Errorf("Result ID = %s, want custom-profile", result.ID)
	}
}

func TestClient_CreateSnapshot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/sys/snapshot" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"snapshot_id": "2024-01-01T00-00-00Z_abc123",
				"current_state": map[string]interface{}{
					"sysctl": map[string]string{
						"net.ipv4.tcp_congestion_control": "cubic",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	snapshot, err := client.CreateSnapshot()

	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}
	if snapshot.ID != "2024-01-01T00-00-00Z_abc123" {
		t.Errorf("Snapshot ID = %s, want 2024-01-01T00-00-00Z_abc123", snapshot.ID)
	}
}

func TestClient_Apply(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/sys/apply" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"status":      "success",
				"snapshot_id": "snapshot-123",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	req := &types.ApplyRequest{
		ProfileID: "bbr-fq-default",
		Mode:      "dry_run",
	}

	result, err := client.Apply(req)

	if err != nil {
		t.Fatalf("Apply failed: %v", err)
	}
	if result == nil {
		t.Fatal("Apply returned nil result")
	}
}

func TestClient_Rollback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/sys/rollback" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"status":      "success",
				"snapshot_id": "snapshot-123",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	req := &types.RollbackRequest{
		SnapshotID: "snapshot-123",
	}

	result, err := client.Rollback(req)

	if err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
	if result == nil {
		t.Fatal("Rollback returned nil result")
	}
}

func TestClient_GetStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sys/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"current_state": map[string]interface{}{
					"sysctl": map[string]string{
						"net.ipv4.tcp_congestion_control": "bbr",
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	status, err := client.GetStatus()

	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status == nil {
		t.Fatal("GetStatus returned nil")
	}
}

func TestClient_ConnectionError(t *testing.T) {
	client := NewClient("http://localhost:99999", "test-key", 1*time.Second)

	_, err := client.ProbeEcho()
	if err == nil {
		t.Error("Expected connection error")
	}
}

func TestClient_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	_, err := client.ProbeEcho()

	if err == nil {
		t.Error("Expected JSON parse error")
	}
}

func TestClient_ProbeDownload(t *testing.T) {
	testData := make([]byte, 1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/probe/download" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write(testData)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	received, duration, err := client.ProbeDownload(1024)

	if err != nil {
		t.Fatalf("ProbeDownload failed: %v", err)
	}
	if received != 1024 {
		t.Errorf("Received %d bytes, want 1024", received)
	}
	if duration <= 0 {
		t.Error("Duration should be positive")
	}
}

func TestClient_ProbeUpload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.Path != "/probe/upload" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"received_bytes": 1024,
				"duration_ms":    100,
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	data := make([]byte, 1024)
	result, err := client.ProbeUpload(data)

	if err != nil {
		t.Fatalf("ProbeUpload failed: %v", err)
	}
	if result.ReceivedBytes != 1024 {
		t.Errorf("ReceivedBytes = %d, want 1024", result.ReceivedBytes)
	}
}

func TestClient_ProbeInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/probe/info" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		resp := map[string]interface{}{
			"success": true,
			"data": map[string]interface{}{
				"os":       "linux",
				"arch":     "amd64",
				"hostname": "test-server",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key", 5*time.Second)
	info, err := client.ProbeInfo()

	if err != nil {
		t.Fatalf("ProbeInfo failed: %v", err)
	}
	if info == nil {
		t.Fatal("ProbeInfo returned nil")
	}
}
