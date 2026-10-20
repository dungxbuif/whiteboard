package dns

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

func TestTXTResolverHonorsTimeout(t *testing.T) {
	r := NewTXTResolver(&net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}, 10*time.Millisecond)
	_, err := r.LookupTXT(context.Background(), "_whiteboard-verification.invalid.example")
	if err == nil || (!errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled)) {
		t.Fatalf("timeout error=%v", err)
	}
}
