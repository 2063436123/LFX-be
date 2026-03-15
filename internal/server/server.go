package server

import (
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/hook"
)

type Server struct {
	pb      *pocketbase.PocketBase
	service *SyncService
}

func New() *Server {
	pb := pocketbase.New()
	s := &Server{
		pb:      pb,
		service: NewSyncService(pb),
	}

	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{
		Automigrate: true,
	})

	pb.OnBootstrap().BindFunc(func(e *core.BootstrapEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		return s.service.EnsureCollections(e.App)
	})

	pb.OnServe().Bind(&hook.Handler[*core.ServeEvent]{
		Func: func(e *core.ServeEvent) error {
			s.registerRoutes(e)
			if !e.Router.HasRoute(http.MethodGet, "/") {
				e.Router.GET("/", func(re *core.RequestEvent) error {
					return re.JSON(http.StatusOK, map[string]any{
						"service": "lfx-be",
						"status":  "ok",
					})
				})
			}
			return e.Next()
		},
		Priority: 50,
	})

	if os.Getenv("PB_ENCRYPTION_ENV") == "" {
		_ = os.Setenv("PB_ENCRYPTION_ENV", "LFX_PB_ENCRYPTION_KEY")
	}

	return s
}

func (s *Server) Start() error {
	return s.pb.Start()
}
