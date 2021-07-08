package util

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/client"
	"github.com/filecoin-project/lotus/api/v0api"
	"net/http"
)

func GetFullNodeAPIUsingCredentials(ctx context.Context, rpcUrl, token string) (v0api.FullNode, jsonrpc.ClientCloser, error) {
	return client.NewFullNodeRPCV0(ctx, rpcUrl, apiHeaders(token))
}

func apiHeaders(token string) http.Header {
	headers := http.Header{}
	headers.Add("Authorization", "Bearer "+token)
	return headers
}
