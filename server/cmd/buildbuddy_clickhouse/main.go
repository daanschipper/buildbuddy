package main

import (
	"flag"
	"strconv"
	"strings"

	app_bundle "github.com/buildbuddy-io/buildbuddy/app"
	"github.com/buildbuddy-io/buildbuddy/server/config"
	"github.com/buildbuddy-io/buildbuddy/server/janitor"
	"github.com/buildbuddy-io/buildbuddy/server/libmain"
	"github.com/buildbuddy-io/buildbuddy/server/remote_cache/capabilities_server"
	"github.com/buildbuddy-io/buildbuddy/server/util/clickhouse"
	"github.com/buildbuddy-io/buildbuddy/server/util/grpc_server"
	"github.com/buildbuddy-io/buildbuddy/server/util/healthcheck"
	"github.com/buildbuddy-io/buildbuddy/server/util/log"
	"github.com/buildbuddy-io/buildbuddy/server/util/redact"
	"github.com/buildbuddy-io/buildbuddy/server/version"

	_ "google.golang.org/grpc/xds"
)

var serverType = flag.String("server_type", "buildbuddy-server", "The server type to match on health checks")

func main() {
	version.Print("BuildBuddy")

	flag.Parse()
	if err := config.Load(); err != nil {
		log.Fatalf("Error loading config from file: %s", err)
	}
	config.ReloadOnSIGHUP()

	healthChecker := healthcheck.NewHealthChecker(*serverType)
	appFS, err := app_bundle.GetAppFS()
	if err != nil {
		log.Fatalf("Error getting app FS from bundle: %s", err)
	}
	env := libmain.GetConfiguredEnvironmentOrDie(healthChecker, appFS)

	// Initialize OLAP before services can enqueue invocation statistics.
	if err := clickhouse.Register(env); err != nil {
		dataSource := flag.Lookup("olap_database.data_source").Value.String()
		redactedError := strings.ReplaceAll(err.Error(), strconv.Quote(dataSource), "<REDACTED>")
		redactedError = strings.ReplaceAll(redactedError, dataSource, "<REDACTED>")
		log.Fatalf("Error configuring ClickHouse: %s", redact.RedactText(redactedError))
	}

	cleanupService := janitor.NewInvocationJanitor(env)
	cleanupService.Start()
	defer cleanupService.Stop()

	libmain.StartMonitoringHandler(env)
	libmain.RegisterLocalServersAndClients(env)
	if err := capabilities_server.Register(env); err != nil {
		log.Fatalf("%v", err)
	}
	libmain.StartAndRunServices(env, grpc_server.GRPCServerConfig{})
}
