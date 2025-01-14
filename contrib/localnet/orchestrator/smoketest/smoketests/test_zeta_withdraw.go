package smoketests

import (
	"fmt"
	"math/big"

	cctxtypes "github.com/zeta-chain/zetacore/x/crosschain/types"

	ethcommon "github.com/ethereum/go-ethereum/common"
	connectorzevm "github.com/zeta-chain/protocol-contracts/pkg/contracts/zevm/connectorzevm.sol"
	"github.com/zeta-chain/zetacore/common"
	"github.com/zeta-chain/zetacore/contrib/localnet/orchestrator/smoketest/runner"
	"github.com/zeta-chain/zetacore/contrib/localnet/orchestrator/smoketest/utils"
)

func TestZetaWithdraw(sm *runner.SmokeTestRunner) {
	amount := big.NewInt(0).Mul(big.NewInt(1e18), big.NewInt(10)) // 10 Zeta

	sm.ZevmAuth.Value = amount
	tx, err := sm.WZeta.Deposit(sm.ZevmAuth)
	if err != nil {
		panic(err)
	}
	sm.ZevmAuth.Value = big.NewInt(0)
	sm.Logger.Info("wzeta deposit tx hash: %s", tx.Hash().Hex())

	receipt := utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "wzeta deposit")
	if receipt.Status == 0 {
		panic("deposit failed")
	}

	chainID, err := sm.GoerliClient.ChainID(sm.Ctx)
	if err != nil {
		panic(err)
	}

	tx, err = sm.WZeta.Approve(sm.ZevmAuth, sm.ConnectorZEVMAddr, amount)
	if err != nil {
		panic(err)
	}
	sm.Logger.Info("wzeta approve tx hash: %s", tx.Hash().Hex())

	receipt = utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "wzeta approve")
	if receipt.Status == 0 {
		panic(fmt.Sprintf("approve failed, logs: %+v", receipt.Logs))
	}

	tx, err = sm.ConnectorZEVM.Send(sm.ZevmAuth, connectorzevm.ZetaInterfacesSendInput{
		DestinationChainId:  chainID,
		DestinationAddress:  sm.DeployerAddress.Bytes(),
		DestinationGasLimit: big.NewInt(400_000),
		Message:             nil,
		ZetaValueAndGas:     amount,
		ZetaParams:          nil,
	})
	if err != nil {
		panic(err)
	}
	sm.Logger.Info("send tx hash: %s", tx.Hash().Hex())
	receipt = utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "send")
	if receipt.Status == 0 {
		panic(fmt.Sprintf("send failed, logs: %+v", receipt.Logs))

	}

	sm.Logger.Info("  Logs:")
	for _, log := range receipt.Logs {
		sentLog, err := sm.ConnectorZEVM.ParseZetaSent(*log)
		if err == nil {
			sm.Logger.Info("    Dest Addr: %s", ethcommon.BytesToAddress(sentLog.DestinationAddress).Hex())
			sm.Logger.Info("    Dest Chain: %d", sentLog.DestinationChainId)
			sm.Logger.Info("    Dest Gas: %d", sentLog.DestinationGasLimit)
			sm.Logger.Info("    Zeta Value: %d", sentLog.ZetaValueAndGas)
		}
	}
	sm.Logger.Info("waiting for cctx status to change to final...")

	cctx := utils.WaitCctxMinedByInTxHash(sm.Ctx, tx.Hash().Hex(), sm.CctxClient, sm.Logger, sm.CctxTimeout)
	sm.Logger.CCTX(*cctx, "zeta withdraw")
	if cctx.CctxStatus.Status != cctxtypes.CctxStatus_OutboundMined {
		panic(fmt.Errorf(
			"expected cctx status to be %s; got %s, message %s",
			cctxtypes.CctxStatus_OutboundMined,
			cctx.CctxStatus.Status.String(),
			cctx.CctxStatus.StatusMessage,
		))
	}
}

func TestZetaWithdrawBTCRevert(sm *runner.SmokeTestRunner) {
	sm.ZevmAuth.Value = big.NewInt(1e18) // 1 Zeta
	tx, err := sm.WZeta.Deposit(sm.ZevmAuth)
	if err != nil {
		panic(err)
	}
	sm.ZevmAuth.Value = big.NewInt(0)
	sm.Logger.Info("Deposit tx hash: %s", tx.Hash().Hex())

	receipt := utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "Deposit")
	if receipt.Status != 1 {
		panic("Deposit failed")
	}

	tx, err = sm.WZeta.Approve(sm.ZevmAuth, sm.ConnectorZEVMAddr, big.NewInt(1e18))
	if err != nil {
		panic(err)
	}
	sm.Logger.Info("wzeta.approve tx hash: %s", tx.Hash().Hex())

	receipt = utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "Approve")
	if receipt.Status != 1 {
		panic("Approve failed")
	}

	tx, err = sm.ConnectorZEVM.Send(sm.ZevmAuth, connectorzevm.ZetaInterfacesSendInput{
		DestinationChainId:  big.NewInt(common.BtcRegtestChain().ChainId),
		DestinationAddress:  sm.DeployerAddress.Bytes(),
		DestinationGasLimit: big.NewInt(400_000),
		Message:             nil,
		ZetaValueAndGas:     big.NewInt(1e17),
		ZetaParams:          nil,
	})
	if err != nil {
		panic(err)
	}
	sm.Logger.Info("send tx hash: %s", tx.Hash().Hex())

	receipt = utils.MustWaitForTxReceipt(sm.Ctx, sm.ZevmClient, tx, sm.Logger, sm.ReceiptTimeout)
	sm.Logger.EVMReceipt(*receipt, "send")
	if receipt.Status != 0 {
		panic("Was able to send ZETA to BTC")
	}
}
