package dns

import (
	"context"
	"net"
	"time"
)

type TXTResolver struct {
	resolver *net.Resolver
	timeout  time.Duration
}

func NewTXTResolver(resolver *net.Resolver, timeout time.Duration) *TXTResolver {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &TXTResolver{resolver: resolver, timeout: timeout}
}

func (r *TXTResolver) LookupTXT(ctx context.Context, name string) ([]string, error) {
	lookupCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	return r.resolver.LookupTXT(lookupCtx, name)
}
