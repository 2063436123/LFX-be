package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type SyncService struct {
	app core.App
}

func NewSyncService(app core.App) *SyncService {
	return &SyncService{app: app}
}

func (s *SyncService) PullChanges(e *core.RequestEvent) (*PullResponse, error) {
	res := &PullResponse{
		ServerTime: time.Now().UTC().Format(time.RFC3339),
	}

	var err error
	if res.Customers, err = s.listCustomersSince(queryTime(e.Request, "customersSince")); err != nil {
		return nil, err
	}
	if res.Recharges, err = s.listRecordsSince("recharge_records", queryTime(e.Request, "rechargesSince")); err != nil {
		return nil, err
	}
	if res.Consumes, err = s.listRecordsSince("consume_records", queryTime(e.Request, "consumesSince")); err != nil {
		return nil, err
	}
	if res.Logs, err = s.listRecordsSince("logs", queryTime(e.Request, "logsSince")); err != nil {
		return nil, err
	}
	if res.Conflicts, err = s.listConflictsSince(queryTime(e.Request, "conflictsSince")); err != nil {
		return nil, err
	}
	if res.Products, err = s.listProductsSince(queryTime(e.Request, "productsSince"), 1, 1000); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *SyncService) CreateCustomer(req CustomerCreateRequest) (*PushResult, error) {
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, errors.New("clientId is required")
	}

	record, err := s.findCustomer(req.RemoteID(), req.ClientID)
	if err == nil && record != nil {
		dto := customerDTO(record)
		return &PushResult{Status: "ok", Customer: &dto}, nil
	}

	var out *core.Record
	err = s.app.RunInTransaction(func(txApp core.App) error {
		col, err := txApp.FindCollectionByNameOrId("customers")
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		record := core.NewRecord(col)
		record.Set("clientId", req.ClientID)
		record.Set("name", strings.TrimSpace(req.Fields.Name))
		record.Set("birthYear", req.Fields.BirthYear)
		record.Set("gender", strings.TrimSpace(req.Fields.Gender))
		record.Set("phone", strings.TrimSpace(req.Fields.Phone))
		record.Set("remark", req.Fields.Remark)
		record.Set("balance", req.Balance)
		record.Set("serverVersion", 1)
		record.Set("deleted", false)
		record.Set("deletedAt", "")
		record.Set("createdAt", now)
		record.Set("changedAt", now)
		record.Set("updatedByDeviceId", req.DeviceID)
		record.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(record); err != nil {
			return err
		}
		out = record
		return nil
	})
	if err != nil {
		return nil, err
	}

	dto := customerDTO(out)
	return &PushResult{Status: "ok", Customer: &dto}, nil
}

func (s *SyncService) PatchCustomer(req CustomerPatchRequest) (*PushResult, error) {
	record, err := s.findCustomer(req.RemoteID, req.ClientID)
	if err != nil {
		return nil, err
	}
	if record.GetBool("deleted") {
		return nil, errors.New("customer already deleted")
	}

	var conflict *ConflictDTO
	err = s.app.RunInTransaction(func(txApp core.App) error {
		current, err := resolveCustomer(txApp, req.RemoteID, req.ClientID)
		if err != nil {
			return err
		}

		changedAny := false
		for field, rawNew := range req.Changes {
			baseValue, ok := baseFieldValue(req.BaseSnapshot, field)
			if !ok {
				continue
			}
			newValue := fmt.Sprintf("%v", rawNew)
			currentValue := current.GetString(field)
			if field == "birthYear" {
				currentValue = strconv.Itoa(current.GetInt(field))
			}
			if currentValue != baseValue && currentValue != newValue {
				conflictRecord, err := s.createConflict(txApp, current, req, field, baseValue, newValue, currentValue)
				if err != nil {
					return err
				}
				dto := conflictDTO(conflictRecord)
				conflict = &dto
				continue
			}

			switch field {
			case "name", "gender", "phone", "remark":
				current.Set(field, newValue)
				changedAny = true
			case "birthYear":
				current.Set(field, toInt(rawNew, current.GetInt(field)))
				changedAny = true
			}
		}

		if changedAny {
			now := time.Now().UTC().Format(time.RFC3339)
			current.Set("serverVersion", current.GetInt("serverVersion")+1)
			current.Set("changedAt", now)
			current.Set("updatedByDeviceId", req.DeviceID)
			current.Set("updatedByAdminId", req.AdminID)
			if err := txApp.Save(current); err != nil {
				return err
			}
		}

		record = current
		return nil
	})
	if err != nil {
		return nil, err
	}

	customer := customerDTO(record)
	if conflict != nil {
		return &PushResult{Status: "conflict", Customer: &customer, Conflict: conflict}, nil
	}
	return &PushResult{Status: "ok", Customer: &customer}, nil
}

func (s *SyncService) DeleteCustomer(req CustomerDeleteRequest) (*PushResult, error) {
	record, err := s.findCustomer(req.RemoteID, req.ClientID)
	if err != nil {
		return nil, err
	}

	err = s.app.RunInTransaction(func(txApp core.App) error {
		current, err := resolveCustomer(txApp, req.RemoteID, req.ClientID)
		if err != nil {
			return err
		}
		if current.GetBool("deleted") {
			record = current
			return nil
		}
		current.Set("deleted", true)
		now := time.Now().UTC().Format(time.RFC3339)
		current.Set("deletedAt", now)
		current.Set("changedAt", now)
		current.Set("serverVersion", current.GetInt("serverVersion")+1)
		current.Set("updatedByDeviceId", req.DeviceID)
		current.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(current); err != nil {
			return err
		}
		record = current
		return nil
	})
	if err != nil {
		return nil, err
	}

	dto := customerDTO(record)
	return &PushResult{Status: "ok", Customer: &dto}, nil
}

func (s *SyncService) CreateRecharge(req RechargeRequest) (*PushResult, error) {
	if strings.TrimSpace(req.EventID) == "" {
		return nil, errors.New("eventId is required")
	}
	if existing, _ := s.app.FindFirstRecordByData("recharge_records", "eventId", req.EventID); existing != nil {
		dto := recordDTO(existing, "recharge_records")
		return &PushResult{Status: "ok", Record: &dto}, nil
	}

	var out *core.Record
	err := s.app.RunInTransaction(func(txApp core.App) error {
		customer, err := resolveCustomer(txApp, req.RemoteCustomerID, req.CustomerID)
		if err != nil {
			return err
		}
		col, err := txApp.FindCollectionByNameOrId("recharge_records")
		if err != nil {
			return err
		}
		record := core.NewRecord(col)
		record.Set("eventId", req.EventID)
		record.Set("customerRef", customer.Id)
		record.Set("clientCustomerId", req.CustomerID)
		record.Set("amount", req.Amount)
		record.Set("adminId", req.AdminID)
		record.Set("adminUsername", req.AdminUsername)
		record.Set("deviceId", req.DeviceID)
		record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
		record.Set("clientCreatedAt", req.ClientCreatedAt)
		if err := txApp.Save(record); err != nil {
			return err
		}

		customer.Set("balance", customer.GetFloat("balance")+req.Amount)
		customer.Set("serverVersion", customer.GetInt("serverVersion")+1)
		customer.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
		customer.Set("updatedByDeviceId", req.DeviceID)
		customer.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(customer); err != nil {
			return err
		}

		out = record
		return nil
	})
	if err != nil {
		return nil, err
	}

	dto := recordDTO(out, "recharge_records")
	return &PushResult{Status: "ok", Record: &dto}, nil
}

func (s *SyncService) CreateConsume(req ConsumeRequest) (*PushResult, error) {
	if strings.TrimSpace(req.EventID) == "" {
		return nil, errors.New("eventId is required")
	}
	if existing, _ := s.app.FindFirstRecordByData("consume_records", "eventId", req.EventID); existing != nil {
		dto := recordDTO(existing, "consume_records")
		return &PushResult{Status: "ok", Record: &dto}, nil
	}

	var out *core.Record
	err := s.app.RunInTransaction(func(txApp core.App) error {
		customer, err := resolveCustomer(txApp, req.RemoteCustomerID, req.CustomerID)
		if err != nil {
			return err
		}
		if customer.GetFloat("balance") < req.TotalAmount {
			return errors.New("insufficient balance")
		}

		col, err := txApp.FindCollectionByNameOrId("consume_records")
		if err != nil {
			return err
		}
		record := core.NewRecord(col)
		record.Set("eventId", req.EventID)
		record.Set("customerRef", customer.Id)
		record.Set("clientCustomerId", req.CustomerID)
		record.Set("productId", req.ProductID)
		record.Set("productName", req.ProductName)
		record.Set("unitPrice", req.UnitPrice)
		record.Set("quantity", req.Quantity)
		record.Set("totalAmount", req.TotalAmount)
		record.Set("adminId", req.AdminID)
		record.Set("adminUsername", req.AdminUsername)
		record.Set("deviceId", req.DeviceID)
		record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
		record.Set("clientCreatedAt", req.ClientCreatedAt)
		if err := txApp.Save(record); err != nil {
			return err
		}

		customer.Set("balance", customer.GetFloat("balance")-req.TotalAmount)
		customer.Set("serverVersion", customer.GetInt("serverVersion")+1)
		customer.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
		customer.Set("updatedByDeviceId", req.DeviceID)
		customer.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(customer); err != nil {
			return err
		}

		out = record
		return nil
	})
	if err != nil {
		return nil, err
	}

	dto := recordDTO(out, "consume_records")
	return &PushResult{Status: "ok", Record: &dto}, nil
}

func (s *SyncService) CreateLog(req LogRequest) (*PushResult, error) {
	if strings.TrimSpace(req.EventID) == "" {
		return nil, errors.New("eventId is required")
	}
	if existing, _ := s.app.FindFirstRecordByData("logs", "eventId", req.EventID); existing != nil {
		dto := recordDTO(existing, "logs")
		return &PushResult{Status: "ok", Record: &dto}, nil
	}

	col, err := s.app.FindCollectionByNameOrId("logs")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	record.Set("eventId", req.EventID)
	record.Set("adminId", req.AdminID)
	record.Set("adminUsername", req.AdminUsername)
	record.Set("action", req.Action)
	record.Set("details", req.Details)
	record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
	record.Set("clientCreatedAt", req.ClientCreatedAt)
	if err := s.app.Save(record); err != nil {
		return nil, err
	}

	dto := recordDTO(record, "logs")
	return &PushResult{Status: "ok", Record: &dto}, nil
}

func (s *SyncService) ResolveConflict(req ResolveConflictRequest) error {
	if strings.TrimSpace(req.ConflictID) == "" {
		return errors.New("conflictId is required")
	}
	record, err := s.app.FindRecordById("sync_conflicts", req.ConflictID)
	if err != nil {
		return err
	}
	record.Set("status", strings.TrimSpace(req.Status))
	record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
	return s.app.Save(record)
}

func (s *SyncService) findCustomer(remoteID, clientID string) (*core.Record, error) {
	return resolveCustomer(s.app, remoteID, clientID)
}

func resolveCustomer(app core.App, remoteID, clientID string) (*core.Record, error) {
	if strings.TrimSpace(remoteID) != "" {
		record, err := app.FindRecordById("customers", remoteID)
		if err == nil {
			return record, nil
		}
	}
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("customer reference not found")
	}
	return app.FindFirstRecordByData("customers", "clientId", clientID)
}

func (s *SyncService) createConflict(app core.App, customer *core.Record, req CustomerPatchRequest, field, baseValue, localValue, remoteValue string) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("sync_conflicts")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	record.Set("customerRef", customer.Id)
	record.Set("clientId", req.ClientID)
	record.Set("fieldName", field)
	record.Set("baseValue", baseValue)
	record.Set("localValue", localValue)
	record.Set("remoteValue", remoteValue)
	record.Set("deviceId", req.DeviceID)
	record.Set("adminId", req.AdminID)
	record.Set("adminUsername", req.AdminUsername)
	record.Set("summary", fmt.Sprintf("conflict on %s for customer %s", field, customer.GetString("name")))
	record.Set("status", "open")
	record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *SyncService) listCustomersSince(since string) ([]CustomerDTO, error) {
	filter := ""
	args := dbx.Params{}
	if since != "" {
		filter = "changedAt >= {:since}"
		args["since"] = since
	}
	return collectCustomers(s.app, filter, args)
}

func (s *SyncService) listRecordsSince(collection, since string) ([]RecordDTO, error) {
	filter := ""
	args := dbx.Params{}
	if since != "" {
		filter = "changedAt >= {:since}"
		args["since"] = since
	}
	records, err := s.app.FindRecordsByFilter(collection, filter, "", 500, 0, args)
	if err != nil {
		return nil, err
	}
	out := make([]RecordDTO, 0, len(records))
	for _, record := range records {
		out = append(out, recordDTO(record, collection))
	}
	return out, nil
}

func (s *SyncService) listConflictsSince(since string) ([]ConflictDTO, error) {
	filter := ""
	args := dbx.Params{}
	if since != "" {
		filter = "changedAt >= {:since}"
		args["since"] = since
	}
	records, err := s.app.FindRecordsByFilter("sync_conflicts", filter, "", 500, 0, args)
	if err != nil {
		return nil, err
	}
	out := make([]ConflictDTO, 0, len(records))
	for _, record := range records {
		out = append(out, conflictDTO(record))
	}
	return out, nil
}

func collectCustomers(app core.App, filter string, args dbx.Params) ([]CustomerDTO, error) {
	records, err := app.FindRecordsByFilter("customers", filter, "", 500, 0, args)
	if err != nil {
		return nil, err
	}
	out := make([]CustomerDTO, 0, len(records))
	for _, record := range records {
		out = append(out, customerDTO(record))
	}
	return out, nil
}

func customerDTO(record *core.Record) CustomerDTO {
	return CustomerDTO{
		ID:            record.Id,
		ClientID:      record.GetString("clientId"),
		Name:          record.GetString("name"),
		BirthYear:     record.GetInt("birthYear"),
		Gender:        record.GetString("gender"),
		Phone:         record.GetString("phone"),
		Remark:        record.GetString("remark"),
		Balance:       record.GetFloat("balance"),
		ServerVersion: record.GetInt("serverVersion"),
		Deleted:       record.GetBool("deleted"),
		DeletedAt:     record.GetString("deletedAt"),
		CreatedAt:     record.GetString("createdAt"),
		UpdatedAt:     record.GetString("changedAt"),
		FieldValues: CustomerFields{
			Name:      record.GetString("name"),
			BirthYear: record.GetInt("birthYear"),
			Gender:    record.GetString("gender"),
			Phone:     record.GetString("phone"),
			Remark:    record.GetString("remark"),
		},
	}
}

func recordDTO(record *core.Record, collection string) RecordDTO {
	dto := RecordDTO{
		ID:              record.Id,
		EventID:         record.GetString("eventId"),
		AdminID:         record.GetString("adminId"),
		AdminUsername:   record.GetString("adminUsername"),
		ClientCreatedAt: record.GetString("clientCreatedAt"),
		CreatedAt:       record.GetString("changedAt"),
		UpdatedAt:       record.GetString("changedAt"),
	}
	switch collection {
	case "recharge_records":
		dto.CustomerID = record.GetString("customerRef")
		dto.ClientID = record.GetString("clientCustomerId")
		dto.Amount = record.GetFloat("amount")
	case "consume_records":
		dto.CustomerID = record.GetString("customerRef")
		dto.ClientID = record.GetString("clientCustomerId")
		dto.ProductID = record.GetString("productId")
		dto.ProductName = record.GetString("productName")
		dto.UnitPrice = record.GetFloat("unitPrice")
		dto.Quantity = record.GetInt("quantity")
		dto.TotalAmount = record.GetFloat("totalAmount")
	case "logs":
		dto.Action = record.GetString("action")
		dto.Details = record.GetString("details")
	}
	return dto
}

func conflictDTO(record *core.Record) ConflictDTO {
	return ConflictDTO{
		ID:          record.Id,
		CustomerID:  record.GetString("customerRef"),
		ClientID:    record.GetString("clientId"),
		FieldName:   record.GetString("fieldName"),
		BaseValue:   record.GetString("baseValue"),
		LocalValue:  record.GetString("localValue"),
		RemoteValue: record.GetString("remoteValue"),
		Summary:     record.GetString("summary"),
		Status:      record.GetString("status"),
		CreatedAt:   record.GetString("changedAt"),
		UpdatedAt:   record.GetString("changedAt"),
	}
}

func baseFieldValue(snapshot CustomerFields, field string) (string, bool) {
	switch field {
	case "name":
		return snapshot.Name, true
	case "birthYear":
		return strconv.Itoa(snapshot.BirthYear), true
	case "gender":
		return snapshot.Gender, true
	case "phone":
		return snapshot.Phone, true
	case "remark":
		return snapshot.Remark, true
	default:
		return "", false
	}
}

func toInt(v any, fallback int) int {
	switch value := v.(type) {
	case int:
		return value
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return fallback
}

func queryTime(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func (req CustomerCreateRequest) RemoteID() string {
	return ""
}

// ========== Product 相关方法 ==========

func (s *SyncService) CreateProduct(req ProductCreateRequest) (*PushResult, error) {
	if strings.TrimSpace(req.ClientID) == "" {
		return nil, errors.New("clientId is required")
	}
	if err := req.Fields.Validate(); err != nil {
		return nil, err
	}
	record, err := s.findProduct(req.ClientID, false)
	if err == nil && record != nil {
		dto := productDTO(record)
		return &PushResult{Status: "ok", Product: &dto}, nil
	}
	var out *core.Record
	err = s.app.RunInTransaction(func(txApp core.App) error {
		col, err := txApp.FindCollectionByNameOrId("products")
		if err != nil {
			return err
		}
		now := time.Now().UTC().Format(time.RFC3339)
		record := core.NewRecord(col)
		record.Set("clientId", req.ClientID)
		record.Set("name", strings.TrimSpace(req.Fields.Name))
		record.Set("price", req.Fields.Price)
		record.Set("serverVersion", 1)
		record.Set("changedAt", now)
		record.Set("createdAt", now)
		record.Set("updatedByDeviceId", req.DeviceID)
		record.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(record); err != nil {
			return err
		}
		out = record
		return nil
	})
	if err != nil {
		return nil, err
	}
	dto := productDTO(out)
	return &PushResult{Status: "ok", Product: &dto}, nil
}

func (s *SyncService) PatchProduct(req ProductPatchRequest) (*PushResult, error) {
	record, err := s.findProduct(req.ClientID, false)
	if err != nil {
		return nil, err
	}
	for field := range req.Changes {
		if !ValidateProductChangeField(field) {
			return nil, fmt.Errorf("cannot modify field: %s", field)
		}
	}
	var conflicts []*ConflictDTO
	err = s.app.RunInTransaction(func(txApp core.App) error {
		current, err := resolveProduct(txApp, req.RemoteID, req.ClientID, false)
		if err != nil {
			return err
		}
		changedAny := false
		hasConflict := false
		for field, rawNew := range req.Changes {
			baseValue, ok := baseFieldValueProduct(req.BaseSnapshot, field)
			if !ok {
				continue
			}
			newValue := fmt.Sprintf("%v", rawNew)
			var currentValue string
			if field == "price" {
				currentValue = strconv.FormatFloat(current.GetFloat(field), 'f', -1, 64)
			} else {
				currentValue = current.GetString(field)
			}
			if currentValue != baseValue && currentValue != newValue {
				conflictRecord, err := s.createProductConflict(txApp, current, req, field, baseValue, newValue, currentValue)
				if err != nil {
					return err
				}
				dto := conflictDTO(conflictRecord)
				conflicts = append(conflicts, &dto)
				hasConflict = true
			}
		}
		if hasConflict {
			return nil
		}
		for field, rawNew := range req.Changes {
			if field == "name" {
				current.Set(field, fmt.Sprintf("%v", rawNew))
				changedAny = true
			} else if field == "price" {
				current.Set(field, toFloat(rawNew, current.GetFloat(field)))
				changedAny = true
			}
		}
		if changedAny {
			now := time.Now().UTC().Format(time.RFC3339)
			current.Set("serverVersion", current.GetInt("serverVersion")+1)
			current.Set("changedAt", now)
			current.Set("updatedByDeviceId", req.DeviceID)
			current.Set("updatedByAdminId", req.AdminID)
			if err := txApp.Save(current); err != nil {
				return err
			}
		}
		record = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	product := productDTO(record)
	if len(conflicts) > 0 {
		return &PushResult{Status: "conflict", Product: &product, Conflict: conflicts[0]}, nil
	}
	return &PushResult{Status: "ok", Product: &product}, nil
}

func (s *SyncService) DeleteProduct(req ProductDeleteRequest) (*PushResult, error) {
	record, err := s.findProduct(req.ClientID, true)
	if err != nil {
		return nil, err
	}
	err = s.app.RunInTransaction(func(txApp core.App) error {
		current, err := resolveProduct(txApp, req.RemoteID, req.ClientID, true)
		if err != nil {
			return err
		}
		if current.GetBool("deleted") {
			record = current
			return nil
		}
		current.Set("deleted", true)
		now := time.Now().UTC().Format(time.RFC3339)
		current.Set("deletedAt", now)
		current.Set("changedAt", now)
		current.Set("serverVersion", current.GetInt("serverVersion")+1)
		current.Set("updatedByDeviceId", req.DeviceID)
		current.Set("updatedByAdminId", req.AdminID)
		if err := txApp.Save(current); err != nil {
			return err
		}
		record = current
		return nil
	})
	if err != nil {
		return nil, err
	}
	dto := productDTO(record)
	return &PushResult{Status: "ok", Product: &dto}, nil
}

func (s *SyncService) listProductsSince(since string, page, pageSize int) ([]ProductDTO, error) {
	filter := ""
	args := dbx.Params{}
	if since != "" {
		filter = "changedAt >= {:since}"
		args["since"] = since
	}
	return collectProducts(s.app, filter, args, page, pageSize)
}

func collectProducts(app core.App, filter string, args dbx.Params, page, pageSize int) ([]ProductDTO, error) {
	records, err := app.FindRecordsByFilter("products", filter, "", pageSize, (page-1)*pageSize, args)
	if err != nil {
		return nil, err
	}
	out := make([]ProductDTO, 0, len(records))
	for _, record := range records {
		out = append(out, productDTO(record))
	}
	return out, nil
}

func (s *SyncService) findProduct(clientID string, includeDeleted bool) (*core.Record, error) {
	if strings.TrimSpace(clientID) == "" {
		return nil, errors.New("clientId is required")
	}
	collection, err := s.app.FindCollectionByNameOrId("products")
	if err != nil {
		return nil, err
	}
	filter := "clientId={:clientId}"
	args := dbx.Params{"clientId": clientID}
	if !includeDeleted {
		filter += " && deleted=false"
	}
	return s.app.FindFirstRecordByFilter(collection, filter, args)
}

func resolveProduct(app core.App, remoteID, clientID string, includeDeleted bool) (*core.Record, error) {
	collection, err := app.FindCollectionByNameOrId("products")
	if err != nil {
		return nil, err
	}
	var record *core.Record
	if remoteID != "" {
		record, err = app.FindRecordById(collection, remoteID)
		if err == nil && record != nil {
			if !includeDeleted && record.GetBool("deleted") {
				return nil, errors.New("product not found")
			}
			return record, nil
		}
	}
	if clientID != "" {
		filter := "clientId={:clientId}"
		args := dbx.Params{"clientId": clientID}
		if !includeDeleted {
			filter += " && deleted=false"
		}
		record, err = app.FindFirstRecordByFilter(collection, filter, args)
		if err == nil && record != nil {
			return record, nil
		}
	}
	return nil, errors.New("product not found")
}

func productDTO(record *core.Record) ProductDTO {
	return ProductDTO{
		ID:            record.Id,
		ClientID:      record.GetString("clientId"),
		Name:          record.GetString("name"),
		Price:         record.GetFloat("price"),
		ServerVersion: record.GetInt("serverVersion"),
		ChangedAt:     record.GetString("changedAt"),
		CreatedAt:     record.GetString("createdAt"),
		UpdatedAt:     record.GetString("changedAt"),
		FieldValues: ProductFields{
			Name:  record.GetString("name"),
			Price: record.GetFloat("price"),
		},
	}
}

func baseFieldValueProduct(snapshot ProductFields, field string) (string, bool) {
	switch field {
	case "name":
		return snapshot.Name, true
	case "price":
		return strconv.FormatFloat(snapshot.Price, 'f', -1, 64), true
	default:
		return "", false
	}
}

func (s *SyncService) createProductConflict(app core.App, product *core.Record, req ProductPatchRequest, field, baseValue, localValue, remoteValue string) (*core.Record, error) {
	col, err := app.FindCollectionByNameOrId("sync_conflicts")
	if err != nil {
		return nil, err
	}
	record := core.NewRecord(col)
	record.Set("customerRef", "")
	record.Set("clientId", req.ClientID)
	record.Set("fieldName", field)
	record.Set("baseValue", baseValue)
	record.Set("localValue", localValue)
	record.Set("remoteValue", remoteValue)
	record.Set("deviceId", req.DeviceID)
	record.Set("adminId", req.AdminID)
	record.Set("adminUsername", req.AdminUsername)
	record.Set("summary", fmt.Sprintf("conflict on %s for product %s", field, product.GetString("name")))
	record.Set("status", "open")
	record.Set("changedAt", time.Now().UTC().Format(time.RFC3339))
	if err := app.Save(record); err != nil {
		return nil, err
	}
	return record, nil
}

func toFloat(value any, fallback float64) float64 {
	switch v := value.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}
	return fallback
}
