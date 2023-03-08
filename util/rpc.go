package util

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	"net/http"
)

func GetFullNodeAPIUsingCredentials(ctx context.Context, rpcUrl, token string) (api.FullNode, jsonrpc.ClientCloser, error) {
	return client.NewFullNodeRPCV1(ctx, rpcUrl, apiHeaders(token))
}

func apiHeaders(token string) http.Header {
	headers := http.Header{}
	//if token != "" {
	headers.Add("Authorization", "Bearer "+token)
	//}
	return headers
}
