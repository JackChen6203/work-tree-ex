package config

import "testing"

func TestLoadDatabaseAppRoleDefaultsToServiceRole(t *testing.T) {
	t.Setenv("DB_APP_ROLE", "")

	cfg := Load()
	if cfg.Database.AppRole != "service_role" {
		t.Fatalf("expected default DB_APP_ROLE to be service_role, got %q", cfg.Database.AppRole)
	}
}

func TestLoadDatabaseAppRoleFromEnv(t *testing.T) {
	t.Setenv("DB_APP_ROLE", "anon")

	cfg := Load()
	if cfg.Database.AppRole != "anon" {
		t.Fatalf("expected DB_APP_ROLE from env, got %q", cfg.Database.AppRole)
	}
}

func TestLoadRedisPoolDefaults(t *testing.T) {
	t.Setenv("REDIS_POOL_SIZE", "")
	t.Setenv("REDIS_MIN_IDLE_CONNS", "")
	t.Setenv("REDIS_CONN_MAX_LIFETIME_MIN", "")
	t.Setenv("REDIS_CONN_MAX_IDLE_MIN", "")

	cfg := Load()
	if cfg.Redis.PoolSize != 50 {
		t.Fatalf("expected REDIS_POOL_SIZE default 50, got %d", cfg.Redis.PoolSize)
	}
	if cfg.Redis.MinIdleConns != 10 {
		t.Fatalf("expected REDIS_MIN_IDLE_CONNS default 10, got %d", cfg.Redis.MinIdleConns)
	}
}

func TestRuntimeModeDefaultsToSingle(t *testing.T) {
	t.Setenv("RUNTIME_MODE", "")

	cfg := Load()
	if cfg.RuntimeMode != "single" {
		t.Fatalf("expected RUNTIME_MODE default single, got %q", cfg.RuntimeMode)
	}
	if cfg.DistributedModeEnabled() {
		t.Fatalf("expected distributed mode disabled by default")
	}
}

func TestRuntimeModeDistributedEnablesDistributedFeatures(t *testing.T) {
	t.Setenv("RUNTIME_MODE", "distributed")

	cfg := Load()
	if !cfg.DistributedModeEnabled() {
		t.Fatalf("expected distributed mode to be enabled")
	}
}
