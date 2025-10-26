package grpc

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

type GrpcClient struct {
	cfg configs.Config
}

func NewGrpcClient(cfg configs.Config) *GrpcClient {
	return &GrpcClient{cfg: cfg}
}

func (c *GrpcClient) Close() error {
	return nil
}

func (c *GrpcClient) Dial(ctx context.Context) (*grpc.ClientConn, error) {
	if c.cfg.GrpcServer.Timeout <= 0 {
		c.cfg.GrpcServer.Timeout = 5 * time.Second
	}

	// Base options
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  200 * time.Millisecond,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   5 * time.Second,
			},
			MinConnectTimeout: c.cfg.GrpcServer.Timeout,
		}),
		// Enable gRPC retries (idempotent/unary) if server exposes it via service config.
		grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
	}

	// Credentials
	if c.cfg.GrpcServer.UseTLS {
		var creds credentials.TransportCredentials
		if c.cfg.GrpcServer.CACertPath != "" {
			pem, err := os.ReadFile(c.cfg.GrpcServer.CACertPath)
			if err != nil {
				return nil, err
			}
			pool := x509.NewCertPool()
			if ok := pool.AppendCertsFromPEM(pem); !ok {
				return nil, ErrBadCACert
			}
			tlsCfg := &tls.Config{RootCAs: pool}
			if c.cfg.GrpcServer.ServerName != "" {
				tlsCfg.ServerName = c.cfg.GrpcServer.ServerName
			}
			creds = credentials.NewTLS(tlsCfg)
		} else {
			// System CA
			creds = credentials.NewClientTLSFromCert(nil, c.cfg.GrpcServer.ServerName)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Size limits (optional)
	if c.cfg.GrpcServer.MaxRecvBytes > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(c.cfg.GrpcServer.MaxRecvBytes)))
	}
	if c.cfg.GrpcServer.MaxSendBytes > 0 {
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.MaxCallSendMsgSize(c.cfg.GrpcServer.MaxSendBytes)))
	}

	// Deadline for the DIAL itself
	dialCtx, cancel := context.WithTimeout(ctx, c.cfg.GrpcServer.Timeout)
	defer cancel()

	return grpc.DialContext(dialCtx, c.cfg.GrpcServer.Target, opts...)
}

var ErrBadCACert = &badCACert{"unable to parse CA cert"}

type badCACert struct{ msg string }

func (e *badCACert) Error() string { return e.msg }
