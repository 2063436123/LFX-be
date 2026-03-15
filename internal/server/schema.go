package server

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

func (s *SyncService) EnsureCollections(app core.App) error {
	if err := ensureAuthCollection(app, syncAuthCollectionName); err != nil {
		return fmt.Errorf("%s auth collection: %w", syncAuthCollectionName, err)
	}

	requiredAuth := types.Pointer("@request.auth.id != ''")

	if err := ensureCollection(app, buildCustomersCollection(requiredAuth)); err != nil {
		return fmt.Errorf("customers collection: %w", err)
	}
	if err := ensureCollection(app, buildProductsCollection(requiredAuth)); err != nil {
		return fmt.Errorf("products collection: %w", err)
	}
	if err := ensureCollection(app, buildRechargeCollection(requiredAuth)); err != nil {
		return fmt.Errorf("recharge_records collection: %w", err)
	}
	if err := ensureCollection(app, buildConsumeCollection(requiredAuth)); err != nil {
		return fmt.Errorf("consume_records collection: %w", err)
	}
	if err := ensureCollection(app, buildLogsCollection(requiredAuth)); err != nil {
		return fmt.Errorf("logs collection: %w", err)
	}
	if err := ensureCollection(app, buildConflictsCollection(requiredAuth)); err != nil {
		return fmt.Errorf("sync_conflicts collection: %w", err)
	}
	return nil
}

func ensureCollection(app core.App, col *core.Collection) error {
	existing, err := app.FindCollectionByNameOrId(col.Name)
	if err == nil && existing != nil {
		syncCollection(existing, col)
		return app.Save(existing)
	}
	return app.Save(col)
}

func ensureAuthCollection(app core.App, name string) error {
	col, err := app.FindCollectionByNameOrId(name)
	if err != nil {
		return err
	}
	if !col.IsAuth() {
		return fmt.Errorf("collection %q is not an auth collection", name)
	}
	return nil
}

func syncCollection(existing, target *core.Collection) {
	existing.ListRule = target.ListRule
	existing.ViewRule = target.ViewRule
	existing.CreateRule = target.CreateRule
	existing.UpdateRule = target.UpdateRule
	existing.DeleteRule = target.DeleteRule
	existing.Indexes = append([]string(nil), target.Indexes...)

	targetFields := map[string]struct{}{}
	for _, field := range target.Fields {
		targetFields[field.GetName()] = struct{}{}
		existing.Fields.Add(field)
	}
	for _, field := range existing.Fields {
		if field.GetSystem() {
			continue
		}
		if _, ok := targetFields[field.GetName()]; !ok {
			existing.Fields.RemoveByName(field.GetName())
		}
	}
}

func buildCustomersCollection(rule *string) *core.Collection {
	minBalance := 0.0
	minYear := 1900.0
	col := core.NewBaseCollection("customers")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "clientId", Required: true},
		&core.TextField{Name: "name", Required: true},
		&core.NumberField{Name: "birthYear", Required: true, OnlyInt: true, Min: &minYear},
		&core.TextField{Name: "gender", Required: true},
		&core.TextField{Name: "phone", Required: true},
		&core.TextField{Name: "remark"},
		&core.NumberField{Name: "balance", Min: &minBalance},
		&core.NumberField{Name: "serverVersion", Required: true, OnlyInt: true},
		&core.BoolField{Name: "deleted"},
		&core.TextField{Name: "deletedAt"},
		&core.TextField{Name: "createdAt", Required: true},
		&core.TextField{Name: "changedAt", Required: true},
		&core.TextField{Name: "updatedByDeviceId"},
		&core.TextField{Name: "updatedByAdminId"},
	)
	col.AddIndex("idx_customers_client_id_unique", true, "clientId", "")
	return col
}

func buildProductsCollection(rule *string) *core.Collection {
	minPrice := 0.0
	minVersion := 0.0
	col := core.NewBaseCollection("products")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "clientId", Required: true},
		&core.TextField{Name: "name", Required: true},
		&core.NumberField{Name: "price", Required: true, Min: &minPrice},
		&core.NumberField{Name: "serverVersion", Required: true, OnlyInt: true, Min: &minVersion},
		&core.BoolField{Name: "deleted"},
		&core.TextField{Name: "deletedAt"},
		&core.TextField{Name: "createdAt", Required: true},
		&core.TextField{Name: "changedAt", Required: true},
		&core.TextField{Name: "updatedByDeviceId"},
		&core.TextField{Name: "updatedByAdminId"},
	)
	col.AddIndex("idx_products_client_id_unique", true, "clientId", "")
	return col
}

func buildRechargeCollection(rule *string) *core.Collection {
	minAmount := 0.0
	col := core.NewBaseCollection("recharge_records")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "eventId", Required: true},
		&core.TextField{Name: "customerRef", Required: true},
		&core.TextField{Name: "clientCustomerId", Required: true},
		&core.NumberField{Name: "amount", Min: &minAmount},
		&core.TextField{Name: "adminId"},
		&core.TextField{Name: "adminUsername"},
		&core.TextField{Name: "deviceId"},
		&core.TextField{Name: "changedAt", Required: true},
		&core.TextField{Name: "clientCreatedAt"},
	)
	col.AddIndex("idx_recharge_event_id_unique", true, "eventId", "")
	return col
}

func buildConsumeCollection(rule *string) *core.Collection {
	minAmount := 0.0
	minQty := 1.0
	col := core.NewBaseCollection("consume_records")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "eventId", Required: true},
		&core.TextField{Name: "customerRef", Required: true},
		&core.TextField{Name: "clientCustomerId", Required: true},
		&core.TextField{Name: "productId"},
		&core.TextField{Name: "productName", Required: true},
		&core.NumberField{Name: "unitPrice", Min: &minAmount},
		&core.NumberField{Name: "quantity", Required: true, OnlyInt: true, Min: &minQty},
		&core.NumberField{Name: "totalAmount", Min: &minAmount},
		&core.TextField{Name: "adminId"},
		&core.TextField{Name: "adminUsername"},
		&core.TextField{Name: "deviceId"},
		&core.TextField{Name: "changedAt", Required: true},
		&core.TextField{Name: "clientCreatedAt"},
	)
	col.AddIndex("idx_consume_event_id_unique", true, "eventId", "")
	return col
}

func buildLogsCollection(rule *string) *core.Collection {
	col := core.NewBaseCollection("logs")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "eventId", Required: true},
		&core.TextField{Name: "adminId"},
		&core.TextField{Name: "adminUsername"},
		&core.TextField{Name: "action", Required: true},
		&core.TextField{Name: "details"},
		&core.TextField{Name: "changedAt", Required: true},
		&core.TextField{Name: "clientCreatedAt"},
	)
	col.AddIndex("idx_logs_event_id_unique", true, "eventId", "")
	return col
}

func buildConflictsCollection(rule *string) *core.Collection {
	col := core.NewBaseCollection("sync_conflicts")
	col.ListRule = rule
	col.ViewRule = rule
	col.CreateRule = rule
	col.UpdateRule = rule
	col.DeleteRule = rule
	col.Fields.Add(
		&core.TextField{Name: "customerRef", Required: true},
		&core.TextField{Name: "clientId", Required: true},
		&core.TextField{Name: "fieldName", Required: true},
		&core.TextField{Name: "baseValue"},
		&core.TextField{Name: "localValue"},
		&core.TextField{Name: "remoteValue"},
		&core.TextField{Name: "deviceId"},
		&core.TextField{Name: "adminId"},
		&core.TextField{Name: "adminUsername"},
		&core.TextField{Name: "summary", Required: true},
		&core.TextField{Name: "status", Required: true},
		&core.TextField{Name: "changedAt", Required: true},
	)
	return col
}
