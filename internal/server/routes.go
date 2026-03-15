package server

import (
	"net/http"
	"strings"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

const syncAuthCollectionName = "users"

func (s *Server) registerRoutes(e *core.ServeEvent) {
	publicGroup := e.Router.Group("/api/lfx-sync")
	publicGroup.GET("/health", func(re *core.RequestEvent) error {
		return re.JSON(http.StatusOK, map[string]any{"status": "ok"})
	})
	publicGroup.POST("/auth/login", s.handleAuthLogin)

	authGroup := e.Router.Group("/api/lfx-sync")
	authGroup.Bind(apis.RequireAuth(syncAuthCollectionName))
	authGroup.GET("/auth/me", s.handleAuthMe)
	authGroup.POST("/auth/refresh", s.handleAuthRefresh)
	authGroup.POST("/auth/logout", s.handleAuthLogout)
	authGroup.GET("/pull", s.handlePull)
	authGroup.POST("/customers/create", s.handleCreateCustomer)
	authGroup.POST("/customers/patch", s.handlePatchCustomer)
	authGroup.POST("/customers/delete", s.handleDeleteCustomer)
	authGroup.POST("/recharges/create", s.handleCreateRecharge)
	authGroup.POST("/consumes/create", s.handleCreateConsume)
	authGroup.POST("/logs/create", s.handleCreateLog)
	authGroup.POST("/conflicts/resolve", s.handleResolveConflict)
}

func (s *Server) handlePull(e *core.RequestEvent) error {
	res, err := s.service.PullChanges(e)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleAuthLogin(e *core.RequestEvent) error {
	var req AuthLoginRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	user, err := s.findUserByIdentity(req.Identity)
	if err != nil || user == nil || !user.ValidatePassword(req.Password) {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid credentials"})
	}
	token, err := user.NewAuthToken()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, AuthSessionDTO{
		Token: token,
		User:  authUserDTO(user),
	})
}

func (s *Server) handleAuthMe(e *core.RequestEvent) error {
	return e.JSON(http.StatusOK, authUserDTO(e.Auth))
}

func (s *Server) handleAuthRefresh(e *core.RequestEvent) error {
	token, err := e.Auth.NewAuthToken()
	if err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, AuthSessionDTO{
		Token: token,
		User:  authUserDTO(e.Auth),
	})
}

func (s *Server) handleAuthLogout(e *core.RequestEvent) error {
	e.Auth.RefreshTokenKey()
	if err := e.App.Save(e.Auth); err != nil {
		return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateCustomer(e *core.RequestEvent) error {
	var req CustomerCreateRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.CreateCustomer(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handlePatchCustomer(e *core.RequestEvent) error {
	var req CustomerPatchRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.PatchCustomer(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleDeleteCustomer(e *core.RequestEvent) error {
	var req CustomerDeleteRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.DeleteCustomer(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleCreateRecharge(e *core.RequestEvent) error {
	var req RechargeRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.CreateRecharge(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleCreateConsume(e *core.RequestEvent) error {
	var req ConsumeRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.CreateConsume(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleCreateLog(e *core.RequestEvent) error {
	var req LogRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	s.applyActorFromAuth(e, &req.AdminID, &req.AdminUsername)
	res, err := s.service.CreateLog(req)
	if err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, res)
}

func (s *Server) handleResolveConflict(e *core.RequestEvent) error {
	var req ResolveConflictRequest
	if err := e.BindBody(&req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	if err := s.service.ResolveConflict(req); err != nil {
		return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) applyActorFromAuth(e *core.RequestEvent, adminID, adminUsername *string) {
	if e.Auth == nil {
		return
	}
	username := strings.TrimSpace(e.Auth.GetString("username"))
	if username == "" {
		username = strings.TrimSpace(e.Auth.Email())
	}
	if username == "" {
		username = e.Auth.Id
	}
	*adminID = e.Auth.Id
	*adminUsername = username
}

func (s *Server) findUserByIdentity(identity string) (*core.Record, error) {
	trimmed := strings.TrimSpace(identity)
	if trimmed == "" {
		return nil, nil
	}
	if strings.Contains(trimmed, "@") {
		record, err := s.pb.FindAuthRecordByEmail(syncAuthCollectionName, trimmed)
		if err == nil {
			return record, nil
		}
	}
	record, err := s.pb.FindFirstRecordByData(syncAuthCollectionName, "username", trimmed)
	if err == nil {
		return record, nil
	}
	return s.pb.FindAuthRecordByEmail(syncAuthCollectionName, trimmed)
}

func authUserDTO(record *core.Record) AuthUserDTO {
	if record == nil {
		return AuthUserDTO{}
	}
	username := strings.TrimSpace(record.GetString("username"))
	if username == "" {
		username = strings.TrimSpace(record.Email())
	}
	name := strings.TrimSpace(record.GetString("name"))
	if name == "" {
		name = username
	}
	return AuthUserDTO{
		ID:       record.Id,
		Username: username,
		Name:     name,
		Email:    record.Email(),
		Avatar:   strings.TrimSpace(record.GetString("avatar")),
	}
}
