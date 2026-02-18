package results

import (
	"context"

	"github.com/yyhuni/lunafox/worker/internal/server"
)

const subdomainBatchSize = 5000

// WriteSubdomains sends subdomain results to the server in batches.
func WriteSubdomains(
	ctx context.Context,
	client server.ServerClient,
	scanID int,
	targetID int,
	in <-chan Subdomain,
) (items int, batches int, err error) {
	sender := server.NewBatchSender(ctx, client, scanID, targetID, "subdomain", subdomainBatchSize)

	for subdomain := range in {
		if err := sender.Add(subdomain); err != nil {
			return 0, 0, err
		}
	}

	if err := sender.Flush(); err != nil {
		return 0, 0, err
	}

	items, batches = sender.Stats()
	return items, batches, nil
}
