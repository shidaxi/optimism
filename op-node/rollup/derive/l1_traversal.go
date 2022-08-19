package derive

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/ethereum-optimism/optimism/op-node/eth"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/log"
)

// L1 Traversal fetches the next L1 block and exposes it through the progress API

type L1BlockRefByNumberFetcher interface {
	L1BlockRefByNumber(context.Context, uint64) (eth.L1BlockRef, error)
}

type L1Traversal struct {
	log      log.Logger
	l1Blocks L1BlockRefByNumberFetcher
	progress Progress
}

var _ PullStage = (*L1Traversal)(nil)

func NewL1Traversal(log log.Logger, l1Blocks L1BlockRefByNumberFetcher) *L1Traversal {
	return &L1Traversal{
		log:      log,
		l1Blocks: l1Blocks,
	}
}

func (l1t *L1Traversal) Progress() Progress {
	return l1t.progress
}

func (l1t *L1Traversal) NextL1Block(ctx context.Context) error {
	// close origin and do another pipeline sweep, before we try to move to the next origin
	if !l1t.progress.Closed {
		l1t.progress.Closed = true
		return nil
	}

	// If we reorg to a shorter chain, then we'll only derive new L2 data once the L1 reorg
	// becomes longer than the previous L1 chain.
	// This is fine, assuming the new L1 chain is live, but we may want to reconsider this.

	origin := l1t.progress.Origin
	nextL1Origin, err := l1t.l1Blocks.L1BlockRefByNumber(ctx, origin.Number+1)
	if errors.Is(err, ethereum.NotFound) {
		l1t.log.Debug("can't find next L1 block info (yet)", "number", origin.Number+1, "origin", origin)
		return io.EOF
	} else if err != nil {
		return NewTemporaryError(fmt.Errorf("failed to find L1 block info by number, at origin %s next %d: %w", origin, origin.Number+1, err))
	}
	if l1t.progress.Origin.Hash != nextL1Origin.ParentHash {
		return NewResetError(fmt.Errorf("detected L1 reorg from %s to %s with conflicting parent %s", l1t.progress.Origin, nextL1Origin, nextL1Origin.ParentID()))
	}
	l1t.progress.Origin = nextL1Origin
	l1t.progress.Closed = false
	return nil
}

func (l1t *L1Traversal) Reset(ctx context.Context, inner Progress) error {
	l1t.progress.Origin = inner.Origin
	l1t.progress.Closed = true
	l1t.log.Info("completed reset of derivation pipeline", "origin", l1t.progress.Origin)
	return io.EOF
}
