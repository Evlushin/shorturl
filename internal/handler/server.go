package handler

import (
	"context"
	"errors"
	"github.com/Evlushin/shorturl/internal/handler/config"
	"github.com/Evlushin/shorturl/internal/logger"
	"github.com/Evlushin/shorturl/internal/observers"
	pb "github.com/Evlushin/shorturl/proto"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

const defaultShutdownCtxTimeout = 10 * time.Second

func Serve(ctx context.Context, cfg config.Config, shortener Shortener) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	auditManager := observers.InitAuditObservers(ctx, cfg)
	h := newHandlers(shortener, cfg, auditManager)
	router := newRouter(h)

	httpServer := &http.Server{
		Addr:         cfg.ServerAddr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,  // макс. время на чтение запроса
		WriteTimeout: cfg.WriteTimeout, // макс. время на запись ответа
		IdleTimeout:  cfg.IdleTimeout,  // макс. время бездействия соединения
	}

	var wg sync.WaitGroup

	// Запуск сервера
	wg.Add(1)
	go func() {
		defer wg.Done()
		logger.Log.Info("starting server", zap.String("addr", cfg.ServerAddr))
		if cfg.EnableHTTPS {
			// Проверка наличия сертификатов
			if cfg.TLSCertFile == "" || cfg.TLSKeyFile == "" {
				logger.Log.Error("TLS enabled but certificate or key file not provided")
				cancel()
				return
			}

			if !certFilesExist(cfg.TLSCertFile, cfg.TLSKeyFile) {
				logger.Log.Error("no certificates")
				cancel()
				return
			}

			// Запуск HTTPS-сервера
			err := httpServer.ListenAndServeTLS(cfg.TLSCertFile, cfg.TLSKeyFile)
			if err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Log.Error("error serving HTTPS", zap.Error(err))
				cancel()
			}
		} else {

			lis, err := net.Listen("tcp", cfg.ServerAddr)
			if err != nil {
				logger.Log.Error("failed to listen", zap.Error(err))
				cancel()
				return
			}
			if err := httpServer.Serve(lis); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Log.Error("error serving", zap.Error(err))
				cancel()
			}
		}
	}()

	// === ЗАПУСК GRPC-СЕРВЕРА ===
	if cfg.GRPCAddr != "" {
		var grpcServer *grpc.Server

		wg.Add(1)
		go func() {
			defer wg.Done()

			grpcServer = grpc.NewServer()

			pb.RegisterShortenerServiceServer(grpcServer, NewGRPCServer(cfg, shortener))
			healthServer := health.NewServer()
			grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

			lis, err := net.Listen("tcp", cfg.GRPCAddr)
			if err != nil {
				logger.Log.Error("failed to listen for gRPC", zap.Error(err), zap.String("addr", cfg.GRPCAddr))
				cancel()
				return
			}

			logger.Log.Info("starting gRPC server", zap.String("addr", cfg.GRPCAddr))
			err = grpcServer.Serve(lis)
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				logger.Log.Error("error serving gRPC", zap.Error(err))
				cancel()
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			<-ctx.Done()

			logger.Log.Info("stopping gRPC server", zap.String("addr", cfg.GRPCAddr))
			grpcServer.GracefulStop()
			logger.Log.Info("gRPC server stopped", zap.String("addr", cfg.GRPCAddr))
		}()
	}

	// Завершение
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()

		shutdownCtx, shutdownCancel := context.WithTimeout(ctx, defaultShutdownCtxTimeout)
		defer shutdownCancel()

		// Завершение http сервера
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Log.Error("error shutting down http server", zap.Error(err))
		}

		// Завершение auditManager
		auditManager.Stop()

		logger.Log.Info("server shutdown complete")
	}()

	wg.Wait()
}

func certFilesExist(certFile, keyFile string) bool {
	_, errCert := os.Stat(certFile)
	_, errKey := os.Stat(keyFile)
	return errCert == nil && errKey == nil
}
