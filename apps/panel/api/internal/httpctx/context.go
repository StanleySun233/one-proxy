package httpctx

import (
	"context"

	"github.com/StanleySun233/python-proxy/apps/panel/api/internal/domain"
)

type contextKey string

const accountContextKey contextKey = "account"
const tenantContextKey contextKey = "tenant"
const nodeContextKey contextKey = "node"

func WithAccount(ctx context.Context, account domain.Account) context.Context {
	return context.WithValue(ctx, accountContextKey, account)
}

func Account(ctx context.Context) (domain.Account, bool) {
	account, ok := ctx.Value(accountContextKey).(domain.Account)
	return account, ok
}

func WithTenantAuth(ctx context.Context, tenantCtx domain.TenantAuthContext) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantCtx)
}

func TenantAuth(ctx context.Context) (domain.TenantAuthContext, bool) {
	tenantCtx, ok := ctx.Value(tenantContextKey).(domain.TenantAuthContext)
	return tenantCtx, ok
}

func WithNodeID(ctx context.Context, nodeID string) context.Context {
	return context.WithValue(ctx, nodeContextKey, nodeID)
}

func NodeID(ctx context.Context) (string, bool) {
	nodeID, ok := ctx.Value(nodeContextKey).(string)
	return nodeID, ok
}
