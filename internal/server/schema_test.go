package server

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
	"github.com/pocketbase/pocketbase/tools/types"
)

func TestEnsureAuthCollectionUsesPocketBaseUsersCollection(t *testing.T) {
	t.Parallel()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}
	defer app.Cleanup()

	if err := ensureAuthCollection(app, syncAuthCollectionName); err != nil {
		t.Fatalf("ensure auth collection: %v", err)
	}

	col, err := app.FindCollectionByNameOrId(syncAuthCollectionName)
	if err != nil {
		t.Fatalf("find auth collection: %v", err)
	}
	if !col.IsAuth() {
		t.Fatalf("expected %q to be auth collection", syncAuthCollectionName)
	}
	if got := col.GetIndex("idx_users_username_unique"); got == "" {
		t.Fatal("expected unique username index to be configured on auth collection")
	}
	if !col.PasswordAuth.Enabled {
		t.Fatal("expected password auth to be enabled on auth collection")
	}
	if len(col.PasswordAuth.IdentityFields) != 2 ||
		col.PasswordAuth.IdentityFields[0] != "email" ||
		col.PasswordAuth.IdentityFields[1] != "username" {
		t.Fatalf("unexpected identity fields: %#v", col.PasswordAuth.IdentityFields)
	}
}

func TestEnsureCollectionSyncsIndexesForExistingCollection(t *testing.T) {
	t.Parallel()

	app, err := tests.NewTestApp()
	if err != nil {
		t.Fatalf("new test app: %v", err)
	}
	defer app.Cleanup()

	existing := core.NewBaseCollection("customers")
	existing.ListRule = types.Pointer("")
	existing.Fields.Add(&core.TextField{Name: "clientId", Required: true})
	if err := app.Save(existing); err != nil {
		t.Fatalf("save existing collection: %v", err)
	}

	requiredAuth := types.Pointer("@request.auth.id != ''")
	if err := ensureCollection(app, buildCustomersCollection(requiredAuth)); err != nil {
		t.Fatalf("ensure collection: %v", err)
	}

	col, err := app.FindCollectionByNameOrId("customers")
	if err != nil {
		t.Fatalf("find customers collection: %v", err)
	}

	if got := col.GetIndex("idx_customers_client_id_unique"); got == "" {
		t.Fatal("expected unique clientId index to be synced onto existing collection")
	}
}
