package core

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/cluster"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/engine"
	"github.com/Cyprinus12138/vectory/internal/grpc_handler"
	"github.com/Cyprinus12138/vectory/internal/http_handler"
	"github.com/Cyprinus12138/vectory/internal/utils"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/gin-gonic/gin"
	etcd "go.etcd.io/etcd/client/v3"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"time"
)

type Vectory struct {
	ctx         context.Context
	id          string
	server      *http.Server
	router      *gin.Engine
	addr        string
	gRpcService *grpc.Server
	conf        *config.ClusterMetaConfig
	sigChan     chan os.Signal
	etcd        *etcd.Client

	traceProvider *trace.TracerProvider
	meterProvider *metric.MeterProvider
}

func (a *Vectory) Setup(ctx context.Context) (err error) {
	a.conf = config.GetClusterMetaConfig()
	if a.conf == nil {
		logger.Fatal("meta service config missing")
	}

	// Setup instance id
	pkg.ClusterName = a.conf.ClusterName
	a.id = utils.GenInstanceId(a.conf.ClusterName)
	logger.Info("setting up the service", logger.String("instanceId", a.id))

	a.sigChan = make(chan os.Signal, 1)
	a.ctx = ctx
	a.addr = fmt.Sprintf(config.FmtAddr, config.GetPodIp(), config.GetPort())
	a.router = http_handler.BuildRouter()
	a.server = &http.Server{
		Handler: a.router,
		Addr:    a.addr,
	}
	err = a.initOTel()
	if err != nil {
		logger.Error("init oTel providers failed", logger.Err(err))
		return err
	}

	//
	if a.conf.GrpcEnabled {
		a.gRpcService = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

		grpc_handler.SetupService(a.gRpcService)
	}

	if a.conf.ClusterMode.Enabled {
		logger.Info("etcd enabled", logger.Interface("setting", a.conf.ClusterMode))

		a.etcd, err = etcd.NewFromURLs(a.conf.ClusterMode.Endpoints)
		if err != nil {
			logger.Error("error when build etcd client", logger.Err(err))
			return err
		}
		grpc_handler.SetupClusterService(a.gRpcService)

	}
	return nil
}

func (a *Vectory) Start() error {
	logger.Info("starting the server")

	errChan := make(chan error, 2)
	lis, err := net.Listen(config.NetworkTCP, fmt.Sprintf(config.FmtAddr, config.GetPodIp(), config.GetRpcPort()))
	if err != nil {
		return err
	}

	go func() {
		logger.Info("http server start listening", logger.String("addr", a.addr))
		err = a.router.Run(a.addr)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
	}()

	if a.conf.GrpcEnabled {
		go func() {
			logger.Info("rpc server start listening", logger.String("addr", lis.Addr().String()))
			err = a.gRpcService.Serve(lis)
			if err != nil {
				errChan <- err
			}
		}()
	}

	if a.conf.ClusterMode.Enabled {
		cluster.InitEtcdManager(a.ctx, a.etcd, a.conf, a.id)
		manager := cluster.GetManger()
		err = manager.Register(lis)
		if err != nil {
			logger.Error("register node failed", logger.Err(err))
			return err
		}
		logger.Info("registered the node, now enter grace period",
			logger.String("serviceName", a.conf.ClusterName),
			logger.String("instanceId", a.id),
			logger.String("gracePeriodDuration", fmt.Sprintf("%d s", a.conf.ClusterMode.GracePeriod)),
		)
		manager.ReportLoad(a.conf.ClusterMode.LBMode)

		// For cluster mode, a grace period is introduced, to wait all instance come online, avoiding rebalancing too
		// frequently at the staging period.
		time.Sleep(time.Duration(a.conf.ClusterMode.GracePeriod) * time.Second)

		err = manager.SyncCluster()
		if err != nil {
			logger.Error("sync cluster nodes failed", logger.Err(err))
			return err
		}
	} else {
		// Dispose the status updating message because there is no consumer to handle them.
		// It will be consumed by cluster manager to make the node status sync with remote registry (eg. EtCD)
		go func() {
			for {
				<-pkg.StatusUpdating
			}
		}()
		logger.Info(
			"cluster mode disabled, skip register the node",
			logger.String("serviceName", a.conf.ClusterName),
			logger.String("instanceId", a.id),
		)
	}

	pkg.SetStatus(pkg.Init)
	err = engine.InitManager(a.ctx, a.conf.ClusterMode.Enabled, a.etcd)
	if err == nil {
		pkg.SetStatus(pkg.Healthy)
	} else {
		if errors.Is(err, config.ErrPartlySuccess) {
			pkg.SetStatus(pkg.Unhealthy)
		} else {
			pkg.SetStatus(pkg.Inactive)
			logger.Error("init engine failed", logger.Err(err))
			return err
		}
	}

	select {
	case err = <-errChan:
		return err
	case <-a.sigChan:
		return nil
	}
}

func (a *Vectory) Stop(sig os.Signal) {

	if a.conf.ClusterMode.Enabled {
		logger.Info("unregistering rpc")
		cluster.GetManger().Unregister()
	}

	if a.conf.GrpcEnabled {
		logger.Info("stopping rpc")
		a.gRpcService.GracefulStop()
	}

	logger.Info("stopping http server")
	a.server.Shutdown(a.ctx)

	logger.Info("shutting down tracerProvider")
	a.traceProvider.Shutdown(a.ctx)
	a.meterProvider.Shutdown(a.ctx)

	logger.Info("stopping server")
	a.sigChan <- sig
}
