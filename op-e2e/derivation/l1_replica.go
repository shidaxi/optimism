package derivation

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rpc"
)

// L1CanonSrc is used to sync L1 from another node.
// The other node always has the canonical chain.
// May be nil if there is nothing to sync from
type L1CanonSrc func(num uint64) *types.Block

type L1Replica struct {
	log log.Logger

	node *node.Node
	eth  *eth.Ethereum

	// L1 evm / chain
	l1Chain    *core.BlockChain
	l1Database ethdb.Database
	l1Cfg      *core.Genesis
	l1Signer   types.Signer

	canonL1 L1CanonSrc

	failL1RPC error // mock error
}

func NewL1Replica(log log.Logger, genesis *core.Genesis, canonL1 L1CanonSrc) *L1Replica {
	ethCfg := &ethconfig.Config{
		NetworkId:                 genesis.Config.ChainID.Uint64(),
		Genesis:                   genesis,
		RollupDisableTxPoolGossip: true,
	}
	nodeCfg := &node.Config{
		Name:        "l1-geth",
		WSHost:      "127.0.0.1",
		WSPort:      0,
		WSModules:   []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		HTTPModules: []string{"debug", "admin", "eth", "txpool", "net", "rpc", "web3", "personal"},
		DataDir:     "", // in-memory
	}
	n, err := node.New(nodeCfg)
	if err != nil {
		panic(err)
	}

	backend, err := eth.New(n, ethCfg)
	if err != nil {
		panic(err)
	}

	n.RegisterAPIs(tracers.APIs(backend.APIBackend))

	return &L1Replica{
		log:        log,
		l1Chain:    backend.BlockChain(),
		l1Database: backend.ChainDb(),
		l1Cfg:      genesis,
		l1Signer:   types.LatestSigner(genesis.Config),
		canonL1:    canonL1,
		failL1RPC:  nil,
	}
}

var _ ActorL1Replica = (*L1Replica)(nil)

// rewind L1 chain to parent block of head
func (s *L1Replica) actL1RewindToParent(ctx context.Context) error {
	head := s.l1Chain.CurrentHeader().Number.Uint64()
	if head == 0 {
		return InvalidActionErr
	}
	if err := s.l1Chain.SetHead(head - 1); err != nil {
		return fmt.Errorf("failed to rewind L1 chain to nr %d: %v", head-1, err)
	}
	return nil
}

// process next canonical L1 block (may reorg)
func (s *L1Replica) actL1Sync(ctx context.Context) error {
	if s.canonL1 != nil {
		// TODO: implement basic sync
		return InvalidActionErr
	}
	return InvalidActionErr
}

// make next L1 request fail
func (s *L1Replica) actL1RPCFail(ctx context.Context) error {
	if s.failL1RPC != nil { // already set to fail?
		return InvalidActionErr
	}
	s.failL1RPC = errors.New("mock L1 RPC error")
	return nil
}

func (s *L1Replica) RPCClient() *rpc.Client {
	cl, _ := s.node.Attach() // never errors
	return cl
}
