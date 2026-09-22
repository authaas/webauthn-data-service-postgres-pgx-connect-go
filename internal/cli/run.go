//revive:disable:package-comments
package cli

import (
	"context"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"
	"uuid"

	"connectrpc.com/connect/v2"
	"github.com/caarlos0/env/v11"

	grpcdclient "github.com/grpcd/connect-client/client"
	connectserver "github.com/pbrpc/connect-server"
	"github.com/pbrpc/connect-service/diagnostics"
	"github.com/pbrpc/connect-service/health"
	service_lib "github.com/pbrpc/connect-service/service"
	transport "github.com/pbrpc/http-transport"
	"github.com/pbrpc/lifecycle"
	pbrpcotel "github.com/pbrpc/otel"
	svc "github.com/pbrpc/service"

	postgres "github.com/authaas/data-postgres-pgx-go"
	"github.com/authaas/webauthn-data-bindings-connect-go/webauthn/data/dataconnect"
	"github.com/authaas/webauthn-data-service-postgres-pgx-connect-go/internal/service"
	ops "github.com/authaas/webauthn-schema-postgres-bindings-pgx-go"
)

// cleanupTimeout bounds stopping the server and flushing telemetry, together
const cleanupTimeout = 5 * time.Second

// Run serves until a signal arrives or serving fails, and answers with the
// process exit code.
func Run() int {
	ctx := context.Background()

	serveCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	svcCfg := svc.Configuration{Name: "webauthn-data"}
	if err := env.Parse(&svcCfg); err != nil {
		slog.Default().Error("Failed to read configuration", slog.Any("error", err))
		return 1
	}

	stack := lifecycle.Stack{}

	log, flush, err := pbrpcotel.Init(ctx, svcCfg.Name, svcCfg.Version, uuid.New().String())
	if err != nil {
		slog.Default().Error("Failed to initialize telemetry", slog.Any("error", err))
		return 1
	}
	stack.Push(lifecycle.Logged(log, "telemetry", flush))

	host, err := connectserver.FromEnv(log)
	if err != nil {
		log.Error("Failed to create connect server", slog.Any("error", err))
		return 1
	}
	stack.Push(lifecycle.Logged(log, "server", host.HTTPHost.Server.Shutdown))

	defer lifecycle.HandleGracefulShutdown(ctx, log, &stack, cleanupTimeout)

	db, err := postgres.Resolve(ctx, log)
	if err != nil {
		log.Error("Failed to resolve the database", slog.Any("error", err))
		return 1
	}
	defer db.Close()

	server := service.New(ops.New(db), db, db.Address())

	grpcdConfig, err := env.ParseAs[grpcdclient.Configuration]()
	if err != nil {
		log.Error("Could not read configuration", slog.Any("error", err))
		return 1
	}

	base, err := transport.From(nil)
	if err != nil {
		log.Error("Could not build transport", slog.Any("error", err))
		return 1
	}

	conn := grpcdclient.Connect(grpcdConfig.GRPCDAddress, base)

	checks := diagnostics.Checks{
		grpcdclient.CheckName:    grpcdclient.Check(conn),
		service.StorageCheckName: server.StorageCheck,
	}

	methodList, err := service_lib.Register(
		host.Server,
		host.HTTPHost.Mux,
		health.NewServer(),
		checks,
		func(rpc *connect.Server) {
			dataconnect.RegisterServiceHandler(rpc, server)
		},
	)
	if err != nil {
		log.Error("Failed to register services", slog.Any("error", err))
		return 1
	}

	lis, err := net.Listen("tcp", svcCfg.Address)
	if err != nil {
		log.Error("Failed to create listener", slog.Any("error", err))
		return 1
	}

	log = log.With(slog.String("address", lis.Addr().String()))

	// Register holds the stream open; its ending is what removes the rows,
	// so there is no deregistration to wait for here.
	go grpcdclient.New(log, svcCfg.Name, lis.Addr(), methodList, conn).Register(serveCtx)

	serveErr := make(chan error, 1)
	go func() { serveErr <- host.Serve(lis) }()

	log.Info("Connect server listening")

	select {
	case err := <-serveErr:
		if err != nil {
			log.Error("Failed to serve", slog.Any("error", err))
			return 1
		}
	case <-serveCtx.Done():
	}

	return 0
}
