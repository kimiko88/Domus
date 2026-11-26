package tuya_cloud

import (
	"context"
	"testing"
)

func TestAdapterImplementsProviderAndLifecycle(t *testing.T) {
	adapter, err := NewAdapter(PluginType, Config{
		ClientID: "test-client",
		Secret: "test-secret",
		BaseURL: "https://openapi.tuyaeu.com",
		UID: "test-user",
	})
	if err != nil {
		t.Fatalf("NewAdapter: %v", err)
	}
	if adapter.ID() != PluginType || adapter.Type() != PluginType {
		t.Fatalf("unexpected identity: id=%q type=%q", adapter.ID(), adapter.Type())
	}
	if err := adapter.HealthCheck(context.Background()); err == nil {
		t.Fatal("stopped adapter must not be healthy")
	}
	if err := adapter.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := adapter.HealthCheck(context.Background()); err != nil {
		t.Fatalf("started provider should be healthy: %v", err)
	}
	cfg := adapter.Config()
	if cfg.ClientID != "test-client" || cfg.UID != "test-user" {
		t.Fatalf("unexpected provider config: %+v", cfg)
	}

	adapter.SetCredentials("next-client", "next-secret", "https://openapi.tuyaeu.com", "next-user")
	cfg = adapter.Config()
	if cfg.ClientID != "next-client" || cfg.UID != "next-user" {
		t.Fatalf("credentials were not rotated: %+v", cfg)
	}
	if err := adapter.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if err := adapter.HealthCheck(context.Background()); err == nil {
		t.Fatal("stopped adapter must not be healthy")
	}
}

func TestAdapterRejectsUnsupportedRegionalEndpoint(t *testing.T) {
	_, err := NewAdapter(PluginType, Config{
		ClientID: "test-client",
		Secret: "test-secret",
		BaseURL: "https://attacker.example",
	})
	if err == nil {
		t.Fatal("unsupported regional endpoint must be rejected by the protocol client")
	}
}

func TestAdapterStartHonorsCancelledContext(t *testing.T) {
	adapter, err := NewAdapter(PluginType, Config{
		ClientID: "test-client",
		Secret: "test-secret",
		BaseURL: "https://openapi.tuyaeu.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := adapter.Start(ctx); err == nil {
		t.Fatal("Start must respect a cancelled context")
	}
}


func TestAdapterSupportsOAuthOnlyConfigWithoutReusingOAuthCredentialsAsOpenAPI(t *testing.T) {
	adapter, err := NewAdapter("tuya_oauth_only", Config{
		BaseURL: "https://openapi.tuyaeu.com",
	})
	if err != nil {
		t.Fatalf("NewAdapter should accept a provider instance used only for OAuth token storage: %v", err)
	}
	if got := adapter.Config(); got.ClientID != "" || got.Secret != "" {
		t.Fatalf("OAuth-only instance must not invent OpenAPI credentials: %+v", got)
	}
	if err := adapter.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := adapter.Devices(context.Background()); err == nil {
		t.Fatal("OpenAPI calls without OpenAPI credentials must fail explicitly")
	}

	adapter.SetCredentials("openapi-client", "openapi-secret", "https://openapi.tuyaeu.com", "smart-life-user")
	if got := adapter.Config(); got.ClientID != "openapi-client" || got.Secret != "openapi-secret" || got.UID != "smart-life-user" {
		t.Fatalf("OpenAPI credentials were not installed independently: %+v", got)
	}
	if adapter.Client() == nil {
		t.Fatal("valid OpenAPI credentials should initialize the provider client")
	}
}
