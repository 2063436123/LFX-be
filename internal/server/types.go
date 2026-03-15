package server
import (
	"errors"
	"strings"
)

type CustomerFields struct {
	Name      string `json:"name"`
	BirthYear int    `json:"birthYear"`
	Gender    string `json:"gender"`
	Phone     string `json:"phone"`
	Remark    string `json:"remark"`
}

type AuthLoginRequest struct {
	Identity string `json:"identity"`
	Password string `json:"password"`
}

type AuthUserDTO struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

type AuthSessionDTO struct {
	Token string      `json:"token"`
	User  AuthUserDTO `json:"user"`
}

type CustomerDTO struct {
	ID            string         `json:"id"`
	ClientID      string         `json:"clientId"`
	Name          string         `json:"name"`
	BirthYear     int            `json:"birthYear"`
	Gender        string         `json:"gender"`
	Phone         string         `json:"phone"`
	Remark        string         `json:"remark"`
	Balance       float64        `json:"balance"`
	ServerVersion int            `json:"serverVersion"`
	Deleted       bool           `json:"deleted"`
	DeletedAt     string         `json:"deletedAt"`
	CreatedAt     string         `json:"createdAt"`
	UpdatedAt     string         `json:"updatedAt"`
	FieldValues   CustomerFields `json:"fieldValues"`
}

type ConflictDTO struct {
	ID          string `json:"id"`
	CustomerID  string `json:"customerId"`
	ClientID    string `json:"clientId"`
	FieldName   string `json:"fieldName"`
	BaseValue   string `json:"baseValue"`
	LocalValue  string `json:"localValue"`
	RemoteValue string `json:"remoteValue"`
	Summary     string `json:"summary"`
	Status      string `json:"status"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type PullResponse struct {
	ServerTime string        `json:"serverTime"`
	Customers []CustomerDTO `json:"customers"`
	Products  []ProductDTO   `json:"products"`
	Recharges  []RecordDTO   `json:"recharges"`
	Consumes   []RecordDTO   `json:"consumes"`
	Logs       []RecordDTO   `json:"logs"`
	Conflicts  []ConflictDTO `json:"conflicts"`
}

type RecordDTO struct {
	ID              string  `json:"id"`
	EventID         string  `json:"eventId"`
	CustomerID      string  `json:"customerId"`
	ClientID        string  `json:"clientId"`
	ProductID       string  `json:"productId,omitempty"`
	ProductName     string  `json:"productName,omitempty"`
	UnitPrice       float64 `json:"unitPrice,omitempty"`
	Quantity        int     `json:"quantity,omitempty"`
	TotalAmount     float64 `json:"totalAmount,omitempty"`
	Amount          float64 `json:"amount,omitempty"`
	AdminID         string  `json:"adminId"`
	AdminUsername   string  `json:"adminUsername"`
	Action          string  `json:"action,omitempty"`
	Details         string  `json:"details,omitempty"`
	ClientCreatedAt string  `json:"clientCreatedAt"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

type CustomerCreateRequest struct {
	DeviceID      string         `json:"deviceId"`
	AdminID       string         `json:"adminId"`
	AdminUsername string         `json:"adminUsername"`
	ClientID      string         `json:"clientId"`
	Fields        CustomerFields `json:"fields"`
	Balance       float64        `json:"balance"`
}

type CustomerPatchRequest struct {
	DeviceID      string         `json:"deviceId"`
	AdminID       string         `json:"adminId"`
	AdminUsername string         `json:"adminUsername"`
	ClientID      string         `json:"clientId"`
	RemoteID      string         `json:"remoteId"`
	BaseVersion   int            `json:"baseVersion"`
	BaseSnapshot  CustomerFields `json:"baseSnapshot"`
	Changes       map[string]any `json:"changes"`
}

type CustomerDeleteRequest struct {
	DeviceID      string `json:"deviceId"`
	AdminID       string `json:"adminId"`
	AdminUsername string `json:"adminUsername"`
	ClientID      string `json:"clientId"`
	RemoteID      string `json:"remoteId"`
}

type RechargeRequest struct {
	DeviceID         string  `json:"deviceId"`
	EventID          string  `json:"eventId"`
	CustomerID       string  `json:"customerId"`
	RemoteCustomerID string  `json:"remoteCustomerId"`
	Amount           float64 `json:"amount"`
	AdminID          string  `json:"adminId"`
	AdminUsername    string  `json:"adminUsername"`
	ClientCreatedAt  string  `json:"clientCreatedAt"`
}

type ConsumeRequest struct {
	DeviceID         string  `json:"deviceId"`
	EventID          string  `json:"eventId"`
	CustomerID       string  `json:"customerId"`
	RemoteCustomerID string  `json:"remoteCustomerId"`
	ProductID        string  `json:"productId"`
	ProductName      string  `json:"productName"`
	UnitPrice        float64 `json:"unitPrice"`
	Quantity         int     `json:"quantity"`
	TotalAmount      float64 `json:"totalAmount"`
	AdminID          string  `json:"adminId"`
	AdminUsername    string  `json:"adminUsername"`
	ClientCreatedAt  string  `json:"clientCreatedAt"`
}

type LogRequest struct {
	EventID         string `json:"eventId"`
	AdminID         string `json:"adminId"`
	AdminUsername   string `json:"adminUsername"`
	Action          string `json:"action"`
	Details         string `json:"details"`
	ClientCreatedAt string `json:"clientCreatedAt"`
}

type PushResult struct {
	Status   string       `json:"status"`
	Customer *CustomerDTO `json:"customer,omitempty"`
	Conflict *ConflictDTO `json:"conflict,omitempty"`
        Product  *ProductDTO  `json:"product,omitempty"`
	Record   *RecordDTO   `json:"record,omitempty"`
}

type ResolveConflictRequest struct {
	ConflictID string `json:"conflictId"`
	Status     string `json:"status"`
}

// 允许的 Product 变更字段
var allowedProductChangeFields = map[string]bool{
	"name":  true,
	"price": true,
}

// ValidateProductChangeField 验证 Product 变更字段是否允许
func ValidateProductChangeField(field string) bool {
	return allowedProductChangeFields[field]
}

type ProductFields struct {
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

// Validate 验证 Product 字段
func (f ProductFields) Validate() error {
	name := strings.TrimSpace(f.Name)
	if name == "" {
		return errors.New("name is required")
	}
	if len(name) > 100 {
		return errors.New("name must be less than 100 characters")
	}
	if f.Price < 0 {
		return errors.New("price must be non-negative")
	}
	return nil
}

type ProductDTO struct {
	ID            string        `json:"id"`
	ClientID      string        `json:"clientId"`
	Name          string        `json:"name"`
	Price         float64       `json:"price"`
	ServerVersion int           `json:"serverVersion"`
	ChangedAt     string        `json:"changedAt"`
	CreatedAt     string        `json:"createdAt"`
	UpdatedAt     string        `json:"updatedAt"`
	FieldValues   ProductFields `json:"fieldValues"`
}

type ProductCreateRequest struct {
	DeviceID      string        `json:"deviceId"`
	AdminID       string        `json:"adminId"`
	AdminUsername string        `json:"adminUsername"`
	ClientID      string        `json:"clientId"`
	Fields        ProductFields `json:"fields"`
}

type ProductPatchRequest struct {
	DeviceID      string         `json:"deviceId"`
	AdminID       string         `json:"adminId"`
	AdminUsername string         `json:"adminUsername"`
	ClientID      string         `json:"clientId"`
	RemoteID      string         `json:"remoteId"`
	BaseVersion   int            `json:"baseVersion"`
	BaseSnapshot  ProductFields  `json:"baseSnapshot"`
	Changes       map[string]any `json:"changes"`
}

type ProductDeleteRequest struct {
	DeviceID      string `json:"deviceId"`
	AdminID       string `json:"adminId"`
	AdminUsername string `json:"adminUsername"`
	ClientID      string `json:"clientId"`
	RemoteID      string `json:"remoteId"`
}
