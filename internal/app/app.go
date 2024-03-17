package app

import (
	"context"
	"errors"
	"fmt"
	"github.com/Cyprinus12138/vectory/internal/config"
	"github.com/Cyprinus12138/vectory/internal/grpc_client"
	"github.com/Cyprinus12138/vectory/internal/grpc_handler"
	"github.com/Cyprinus12138/vectory/internal/http_handler"
	"github.com/Cyprinus12138/vectory/internal/utils/logger"
	"github.com/Cyprinus12138/vectory/pkg"
	"github.com/chilts/sid"
	"github.com/gin-gonic/gin"
	etcd "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/endpoints"
	"google.golang.org/grpc"
	"net"
	"net/http"
	"os"
	"time"
)

type App struct {
	ctx             context.Context
	id              string
	server          *http.Server
	router          *gin.Engine
	addr            string
	gRpcService     *grpc.Server
	conf            *config.ServiceMetaConfig
	sigChan         chan os.Signal
	etcd            *etcd.Client
	endpointManager endpoints.Manager
}

const defaultRegisterTimeoutMs = 5000

func (a *App) Setup(ctx context.Context) (err error) {

	// Setup instance id
	pkg.ClusterName = a.conf.ClusterName
	a.id = fmt.Sprintf("%s-%s", a.conf.ClusterName, sid.Id())
	logger.Info("setting up the service", logger.String("instanceId", a.id))

	a.sigChan = make(chan os.Signal, 1)
	a.conf = config.GetServiceMetaConfig()
	if a.conf == nil {
		logger.Fatal("meta service config missing")
	}
	a.ctx = ctx
	a.addr = fmt.Sprintf(config.FmtAddr, config.GetPodIp(), config.GetPort())
	a.router = http_handler.BuildRouter()
	a.server = &http.Server{
		Handler: a.router,
		Addr:    a.addr,
	}

	//
	if a.conf.GrpcEnabled {
		a.gRpcService = grpc.NewServer()
		if a.conf.ClusterMode.Enabled {
			logger.Info("etcd enabled", logger.Interface("setting", a.conf.ClusterMode))

			a.etcd, err = etcd.NewFromURLs(a.conf.ClusterMode.Endpoints)
			if err != nil {
				logger.Error("error when build etcd client", logger.Err(err))
				return err
			}
		}
		grpc_handler.SetupService(a.gRpcService)
		err = grpc_client.Init(a.etcd)
		if err != nil {
			logger.Error("error when connect grpc", logger.Err(err))
			return err
		}
	}
	return nil
}

func (a *App) Start() error {
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

		err = a.register(lis)
		if err != nil {
			logger.Error("register rpc failed", logger.Err(err))
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

func (a *App) register(lis net.Listener) (err error) {
	conf := a.conf.ClusterMode
	if conf.Enabled {
		logger.Info(
			"register the rpc service",
			logger.String("serviceName", a.conf.ClusterName),
			logger.String("instanceId", a.id),
		)

		a.endpointManager, err = endpoints.NewManager(
			a.etcd,
			fmt.Sprintf(config.FmtEtcdSvcPath, a.conf.ClusterName),
		)
		if err != nil {
			return err
		}
		var lease *etcd.LeaseGrantResponse

		ctx, cancel := context.WithTimeout(a.ctx, defaultRegisterTimeoutMs*time.Millisecond)
		defer cancel()
		lease, err = a.etcd.Grant(ctx, conf.Ttl)
		if err != nil {
			return err
		}

		var keepAliveChan <-chan *etcd.LeaseKeepAliveResponse
		keepAliveChan, err = a.etcd.Lease.KeepAlive(a.ctx, lease.ID) // Remember to use a.ctx, because the ctx with timeout will be canceled after the register method finished.
		if err != nil {
			return err
		}
		go func() {
			for {
				<-keepAliveChan
			}
		}()

		err = a.endpointManager.AddEndpoint(
			ctx,
			fmt.Sprintf(config.FmtEtcdSvcRegisterPath, a.conf.ClusterName, a.id),
			endpoints.Endpoint{
				Addr: lis.Addr().String(),
			},
			etcd.WithLease(lease.ID),
		)
		if err != nil {
			return err
		}
	} else {
		logger.Info(
			"discovery disabled, skip register the rpc service",
			logger.String("serviceName", a.conf.ClusterName),
			logger.String("instanceId", a.id),
		)
	}
	return nil
}

func (a *App) Stop(sig os.Signal) {

	if a.conf.ClusterMode.Enabled {
		logger.Info("unregistering rpc")
		err := a.endpointManager.DeleteEndpoint(a.ctx, fmt.Sprintf(config.FmtEtcdSvcRegisterPath, a.conf.ClusterName, a.id))
		if err != nil {
			logger.Error("unregister rpc failed", logger.Err(err))
		}
	}

	if a.conf.GrpcEnabled {
		logger.Info("stopping rpc")
		a.gRpcService.GracefulStop()
	}

	logger.Info("stopping http server")
	a.server.Shutdown(a.ctx)

	logger.Info("stopping server")
	a.sigChan <- sig
}
