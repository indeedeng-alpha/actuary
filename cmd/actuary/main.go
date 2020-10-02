package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/auth"
	"github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/indeedeng-alpha/actuary/internal/auth"
	"github.com/indeedeng-alpha/actuary/internal/checks"
	"github.com/indeedeng-alpha/actuary/internal/db"
	"github.com/indeedeng-alpha/actuary/internal/service"
	"github.com/indeedeng-alpha/actuary/v1alpha"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rs/cors"

	"github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/grpc"

	"gorm.io/driver/postgres"

	"gorm.io/gorm"
)

type config struct {
	BindAddress      string
	DatabaseUser     string
	DatabasePassword string
	DatabaseHost     string
	DatabasePort     int
	DatabaseName     string
}

func main() {
	cfg := &config{
		BindAddress: "localhost:8080",
	}

	// use database when not set, useful for testing
	authorizationsFile := ""

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "bind-address",
			EnvVars:     []string{"BIND_ADDRESS"},
			Value:       cfg.BindAddress,
			Destination: &(cfg.BindAddress),
		},
		&cli.StringFlag{
			Name:        "db-user",
			EnvVars:     []string{"DB_USER"},
			Value:       cfg.DatabaseUser,
			Destination: &(cfg.DatabaseUser),
		},
		&cli.StringFlag{
			Name:        "db-password",
			EnvVars:     []string{"DB_PASSWORD"},
			Value:       cfg.DatabasePassword,
			Destination: &(cfg.DatabasePassword),
		},
		&cli.StringFlag{
			Name:        "db-host",
			EnvVars:     []string{"DB_HOST"},
			Value:       cfg.DatabaseHost,
			Destination: &(cfg.DatabaseHost),
		},
		&cli.IntFlag{
			Name:        "db-port",
			EnvVars:     []string{"DB_PORT"},
			Value:       cfg.DatabasePort,
			Destination: &(cfg.DatabasePort),
		},
		&cli.StringFlag{
			Name:        "db-name",
			EnvVars:     []string{"DB_NAME"},
			Value:       cfg.DatabaseName,
			Destination: &(cfg.DatabaseName),
		},
		&cli.StringFlag{
			Name:        "authorizations-file",
			EnvVars:     []string{"AUTHORIZATIONS_FILE"},
			Value:       authorizationsFile,
			Destination: &authorizationsFile,
		},
	}

	app := cli.App{
		Name:  "actuary",
		Flags: flags,
		Action: func(context *cli.Context) error {
			ctx := context.Context

			var authMethod grpc_auth.AuthFunc

			if authorizationsFile != "" {
				authorizations := make(map[string]string)
				body, err := ioutil.ReadFile(authorizationsFile)
				if err != nil {
					return err
				}

				if err := json.Unmarshal(body, &authorizations); err != nil {
					return err
				}

				authMethod = auth.NewStatic(authorizations)
			}

			if authMethod == nil {
				return fmt.Errorf("database authorizations not yet supported")
			}

			opts := []grpc.ServerOption{
				grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
					grpc_prometheus.UnaryServerInterceptor,
					grpc_auth.UnaryServerInterceptor(authMethod),
				)),
				grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
					grpc_prometheus.StreamServerInterceptor,
					grpc_auth.StreamServerInterceptor(authMethod),
				)),
			}

			grpc_prometheus.EnableHandlingTimeHistogram()

			httpServer := runtime.NewServeMux()
			grpcServer := grpc.NewServer(opts...)
			fwdMux := http.NewServeMux()

			// setup actuary
			dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
				cfg.DatabaseUser, cfg.DatabasePassword, cfg.DatabaseHost, cfg.DatabasePort, cfg.DatabaseName)
			dialector := postgres.Open(dsn)
			gormDB, err := gorm.Open(dialector, &gorm.Config{})
			if err != nil {
				return err
			}

			err = gormDB.AutoMigrate(&db.LineItem{})
			if err != nil {
				return err
			}

			actuaryServer := service.NewActuaryServer(gormDB)
			v1alpha.RegisterActuaryServiceServer(grpcServer, actuaryServer)
			_ = v1alpha.RegisterActuaryServiceHandlerServer(ctx, httpServer, actuaryServer)

			grpc_prometheus.Register(grpcServer)

			// setup the /health and /metrics
			allChecks := checks.Checks(gormDB)
			checks.RegisterHealthCheck(ctx, fwdMux, grpcServer, allChecks)
			fwdMux.Handle("/metrics", promhttp.Handler())

			// setup rest of mux
			fwdMux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
				if request.ProtoMajor == 2 &&
					strings.HasPrefix(request.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(writer, request)
				} else {
					httpServer.ServeHTTP(writer, request)
				}
			})

			h2cMux := h2c.NewHandler(fwdMux, &http2.Server{})
			apiMux := cors.Default().Handler(h2cMux)

			logrus.Infof("[main] starting server on %s", cfg.BindAddress)
			return http.ListenAndServe(cfg.BindAddress, apiMux)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
