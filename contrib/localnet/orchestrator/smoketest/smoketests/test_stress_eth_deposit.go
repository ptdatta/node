package smoketests

import (
	"fmt"
	"math/big"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/zeta-chain/zetacore/contrib/localnet/orchestrator/smoketest/runner"
	"github.com/zeta-chain/zetacore/contrib/localnet/orchestrator/smoketest/utils"
	crosschaintypes "github.com/zeta-chain/zetacore/x/crosschain/types"
	"golang.org/x/sync/errgroup"
)

// TestStressEtherDeposit tests the stressing deposit of ether
func TestStressEtherDeposit(sm *runner.SmokeTestRunner) {
	// number of deposits to perform
	numDeposits := 100

	sm.Logger.Print("starting stress test of %d deposits", numDeposits)

	// create a wait group to wait for all the deposits to complete
	var eg errgroup.Group

	// send the deposits
	for i := 0; i < numDeposits; i++ {
		i := i
		hash := sm.DepositERC20WithAmountAndMessage(big.NewInt(100000), []byte{})
		sm.Logger.Print("index %d: starting deposit, tx hash: %s", i, hash.Hex())

		eg.Go(func() error {
			return MonitorEtherDeposit(sm, hash, i, time.Now())
		})
	}

	// wait for all the deposits to complete
	if err := eg.Wait(); err != nil {
		panic(err)
	}

	sm.Logger.Print("all deposits completed")
}

// MonitorEtherDeposit monitors the deposit of ether, returns once the deposit is complete
func MonitorEtherDeposit(sm *runner.SmokeTestRunner, hash ethcommon.Hash, index int, startTime time.Time) error {
	cctx := utils.WaitCctxMinedByInTxHash(sm.Ctx, hash.Hex(), sm.CctxClient, sm.Logger, sm.ReceiptTimeout)
	if cctx.CctxStatus.Status != crosschaintypes.CctxStatus_OutboundMined {
		return fmt.Errorf(
			"index %d: deposit cctx failed with status %s, message %s, cctx index %s",
			index,
			cctx.CctxStatus.Status,
			cctx.CctxStatus.StatusMessage,
			cctx.Index,
		)
	}
	timeToComplete := time.Now().Sub(startTime)
	sm.Logger.Print("index %d: deposit cctx success in %s", index, timeToComplete.String())

	return nil
}
