package config

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"go.etcd.io/bbolt"
)

func TestStoreFirstRunInitializationAndPersistence(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "config.db")

	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}
	if store.IsInitialized() {
		t.Fatal("new config database must require administrator setup")
	}
	if got := store.Config().Web.Port; got != 8080 {
		t.Fatalf("default web port = %d, want 8080", got)
	}
	if err := store.Initialize("correct horse battery staple"); err != nil {
		t.Fatalf("initialize administrator: %v", err)
	}
	if !store.Authenticate("correct horse battery staple") {
		t.Fatal("initialized administrator password must authenticate")
	}
	if store.Authenticate("wrong password") {
		t.Fatal("incorrect administrator password must not authenticate")
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close config store: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen config store: %v", err)
	}
	defer reopened.Close()
	if !reopened.IsInitialized() {
		t.Fatal("administrator setup must persist in config.db")
	}
	if !reopened.Authenticate("correct horse battery staple") {
		t.Fatal("administrator password must persist in config.db")
	}
}

func TestStoreUpdatesSettingsAndAdministratorPasswordTogether(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "config.db"))
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}
	defer store.Close()
	if err := store.Initialize("old-password"); err != nil {
		t.Fatalf("initialize administrator: %v", err)
	}

	next := store.Config()
	next.Save.SourceMode = "agent"
	next.Save.Path = "http://game-host:8081/sync"
	next.Rcon.Address = "game-host:25575"
	if err := store.Update(next, "new-password"); err != nil {
		t.Fatalf("update settings: %v", err)
	}

	got := store.Config()
	if got.Save.SourceMode != "agent" || got.Save.Path != "http://game-host:8081/sync" {
		t.Fatalf("saved source = %#v, want agent URL", got.Save)
	}
	if got.Rcon.Address != "game-host:25575" {
		t.Fatalf("rcon address = %q, want game-host:25575", got.Rcon.Address)
	}
	if store.Authenticate("old-password") {
		t.Fatal("old administrator password must be invalidated")
	}
	if !store.Authenticate("new-password") {
		t.Fatal("new administrator password must authenticate")
	}
}

func TestNewStoreDefaultsRconBase64On(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "config.db"))
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}
	defer store.Close()

	if !store.Config().Rcon.UseBase64 {
		t.Fatal("new config database must default rcon.use_base64 to true")
	}
}

func TestStoreEnablesRconBase64ForLegacyDatabaseOnce(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "config.db")
	store, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}

	// Simulate a config.db written before base64 became the default.
	legacy := store.Config()
	legacy.Rcon.UseBase64 = false
	if err := store.Update(legacy, ""); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}
	store.DeleteKV(rconBase64MigrationKey)
	if err := store.Close(); err != nil {
		t.Fatalf("close config store: %v", err)
	}

	migrated, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen config store: %v", err)
	}
	if !migrated.Config().Rcon.UseBase64 {
		t.Fatal("legacy config database must be migrated to rcon.use_base64 = true")
	}

	// An explicit opt-out afterwards must survive the next restart.
	optOut := migrated.Config()
	optOut.Rcon.UseBase64 = false
	if err := migrated.Update(optOut, ""); err != nil {
		t.Fatalf("opt out of base64: %v", err)
	}
	if err := migrated.Close(); err != nil {
		t.Fatalf("close migrated store: %v", err)
	}

	reopened, err := Open(dbPath)
	if err != nil {
		t.Fatalf("reopen after opt-out: %v", err)
	}
	defer reopened.Close()
	if reopened.Config().Rcon.UseBase64 {
		t.Fatal("deliberate rcon.use_base64 = false must not be re-enabled")
	}
}

func TestStorePreservesLegacyWebPortWithoutPortSource(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "config.db"))
	if err != nil {
		t.Fatalf("open config store: %v", err)
	}
	defer store.Close()

	legacy := store.Config()
	legacy.Web.Port = 19090
	data, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("encode legacy settings: %v", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("decode legacy settings map: %v", err)
	}
	delete(raw["web"].(map[string]any), "port_source")
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatalf("encode settings without port_source: %v", err)
	}
	if err := store.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(configBucket).Put(configKey, data)
	}); err != nil {
		t.Fatalf("write legacy settings: %v", err)
	}

	loaded := store.Config()
	if loaded.Web.Port != 19090 {
		t.Fatalf("legacy web port = %d, want 19090", loaded.Web.Port)
	}
	if loaded.Web.PortSource != WebPortOverrideNone {
		t.Fatalf("legacy web port source = %q, want empty", loaded.Web.PortSource)
	}
}
