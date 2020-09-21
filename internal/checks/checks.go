package checks

import (
	"context"
	"net/http"
	"time"

	"github.com/mjpitz/go-gracefully/check"
	"github.com/mjpitz/go-gracefully/health"
	"github.com/mjpitz/go-gracefully/state"

	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func Checks() []check.Check {
	return []check.Check{
		&check.Periodic{
			Metadata: check.Metadata{
				Name:   "no-op",
				Weight: 1,
			},
			Interval: time.Second * 10,
			Timeout:  time.Second * 30,
			RunFunc: func(ctx context.Context) (state.State, error) {
				return state.OK, nil
			},
		},
	}
}

func RegisterHealthCheck(ctx context.Context, httpServer *http.ServeMux, grpcServer *grpc.Server, checks []check.Check) {
	monitor := health.NewMonitor(checks...)
	reports, unsubscribe := monitor.Subscribe()
	stopCh := ctx.Done()

	healthCheck := grpchealth.NewServer()

	go func() {
		defer unsubscribe()

		for {
			select {
			case <-stopCh:
				return
			case report := <-reports:
				if report.Check == nil {
					if report.Result.State == state.Outage {
						healthCheck.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
					} else {
						healthCheck.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
					}
				}
			}
		}
	}()

	httpServer.HandleFunc("/healthz", health.HandlerFunc(monitor))
	healthpb.RegisterHealthServer(grpcServer, healthCheck)
	_ = monitor.Start(ctx)
}
