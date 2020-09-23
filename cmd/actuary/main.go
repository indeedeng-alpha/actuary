package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"

	"github.com/rs/cors"

	"github.com/sirupsen/logrus"

	"github.com/urfave/cli/v2"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"

	"google.golang.org/grpc"

	"indeed.com/mjpitz/actuary/internal/checks"
	"indeed.com/mjpitz/actuary/internal/service"
	"indeed.com/mjpitz/actuary/v1alpha"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type config struct {
	BindAddress string
}

func main() {
	cfg := &config{
		BindAddress: "localhost:8080",
	}

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "bind-address",
			EnvVars:     []string{"BIND_ADDRESS"},
			Value:       cfg.BindAddress,
			Destination: &(cfg.BindAddress),
		},
	}

	app := cli.App{
		Name:  "actuary",
		Flags: flags,
		Action: func(context *cli.Context) error {
			ctx := context.Context

			httpServer := runtime.NewServeMux()
			grpcServer := grpc.NewServer()
			fwdMux := http.NewServeMux()

			// setup the healthcheck
			allChecks := checks.Checks()
			checks.RegisterHealthCheck(ctx, fwdMux, grpcServer, allChecks)
			fwdMux.Handle("/metrics", promhttp.Handler())

			// setup actuary
			actuaryServer := service.NewActuaryServer()
			v1alpha.RegisterActuaryServiceServer(grpcServer, actuaryServer)
			_ = v1alpha.RegisterActuaryServiceHandlerServer(ctx, httpServer, actuaryServer)

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
