package app

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"os"
	"time"

	"github.com/aq2208/gorder-api/configs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// InitOrderGWConn dials the order-gw service and returns a conn + cleanup.
func InitOrderGWConn(ctx context.Context, cfg configs.Config) (*grpc.ClientConn, func(), error) {
	dialTimeout := cfg.GrpcServer.Timeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}

	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  200 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: dialTimeout,
		}),
	}

	// TLS vs. insecure
	if cfg.GrpcServer.UseTLS {
		var creds credentials.TransportCredentials
		if cfg.GrpcServer.CACertPath != "" {
			pem, err := os.ReadFile(cfg.GrpcServer.CACertPath)
			if err != nil {
				return nil, nil, err
			}
			pool := x509.NewCertPool()
			if ok := pool.AppendCertsFromPEM(pem); !ok {
				return nil, nil, ErrBadCACert
			}
			tlsCfg := &tls.Config{RootCAs: pool}
			if sn := cfg.GrpcServer.ServerName; sn != "" {
				tlsCfg.ServerName = sn
			}
			creds = credentials.NewTLS(tlsCfg)
		} else {
			creds = credentials.NewClientTLSFromCert(nil, cfg.GrpcServer.ServerName)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// (optional) message size knobs
	if n := cfg.GrpcServer.MaxRecvBytes; n > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(n)))
	}
	if n := cfg.GrpcServer.MaxSendBytes; n > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(n)))
	}

	_, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	conn, err := grpc.NewClient(
		cfg.GrpcServer.Target,
		opts...)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() { _ = conn.Close() }
	return conn, cleanup, nil
}

var ErrBadCACert = badCACert{"unable to parse CA cert"}

type badCACert struct{ msg string }

func (e badCACert) Error() string { return e.msg }
