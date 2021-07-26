package pdex

import (
	"encoding/json"
	"strconv"

	"errors"

	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/dataaccessobject/statedb"
	metadataCommon "github.com/incognitochain/incognito-chain/metadata/common"

	instruction "github.com/incognitochain/incognito-chain/instruction/pdexv3"
	"github.com/incognitochain/incognito-chain/metadata"
	metadataPdexv3 "github.com/incognitochain/incognito-chain/metadata/pdexv3"
	"github.com/incognitochain/incognito-chain/utils"
)

type stateProducerV2 struct {
	stateProducerBase
}

func buildModifyParamsInst(
	params metadataPdexv3.Pdexv3Params,
	shardID byte,
	reqTxID common.Hash,
	status string,
) []string {
	modifyingParamsReqContent := metadataPdexv3.ParamsModifyingContent{
		Content: params,
		TxReqID: reqTxID,
		ShardID: shardID,
	}
	modifyingParamsReqContentBytes, _ := json.Marshal(modifyingParamsReqContent)
	return []string{
		strconv.Itoa(metadataCommon.Pdexv3ModifyParamsMeta),
		strconv.Itoa(int(shardID)),
		status,
		string(modifyingParamsReqContentBytes),
	}
}

func isValidPdexv3Params(params Params) bool {
	if params.DefaultFeeRateBPS > MaxFeeRateBPS {
		return false
	}
	for _, feeRate := range params.FeeRateBPS {
		if feeRate > MaxFeeRateBPS {
			return false
		}
	}
	if params.PRVDiscountPercent > MaxPRVDiscountPercent {
		return false
	}
	if params.TradingStakingPoolRewardPercent+params.TradingProtocolFeePercent > 100 {
		return false
	}
	if params.LimitProtocolFeePercent+params.LimitStakingPoolRewardPercent > 100 {
		return false
	}
	return true
}

func (sp *stateProducerV2) addLiquidity(
	txs []metadata.Transaction,
	beaconHeight uint64,
	poolPairs map[string]PoolPairState,
	waitingContributions map[string]statedb.Pdexv3ContributionState,
) (
	[][]string,
	map[string]PoolPairState,
	map[string]statedb.Pdexv3ContributionState,
	error,
) {
	res := [][]string{}
	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		txReqID := tx.Hash().String()
		metaData, ok := tx.GetMetadata().(*metadataPdexv3.AddLiquidity)
		if !ok {
			return res, poolPairs, waitingContributions, errors.New("Can not parse add liquidity metadata")
		}
		waitingContribution, found := waitingContributions[metaData.PairHash()]
		if !found {
			waitingContributions[metaData.PairHash()] = *NewContributionWithMetaData(
				*metaData, txReqID, shardID,
			)
			inst := instruction.NewWaitingAddLiquidityFromMetadata(*metaData, txReqID, shardID).StringSlice()
			res = append(res, inst)
			continue
		}
		delete(waitingContributions, metaData.PairHash())
		waitingContributionMetaData := metadataPdexv3.NewAddLiquidityWithValue(
			waitingContribution.PoolPairID(), metaData.PairHash(),
			waitingContribution.ReceiveAddress(), waitingContribution.RefundAddress(),
			waitingContribution.TokenID(), waitingContribution.Amount(),
			waitingContribution.Amplifier(),
		)
		if waitingContribution.TokenID() == metaData.TokenID() ||
			waitingContribution.Amplifier() != metaData.Amplifier() ||
			waitingContribution.PoolPairID() != metaData.PoolPairID() {
			refundInst0 := instruction.NewRefundAddLiquidityFromMetadata(
				*waitingContributionMetaData, waitingContribution.TxReqID(), waitingContribution.ShardID(),
			).StringSlice()
			res = append(res, refundInst0)
			refundInst1 := instruction.NewRefundAddLiquidityFromMetadata(
				*metaData, txReqID, shardID,
			).StringSlice()
			res = append(res, refundInst1)
			continue
		}

		poolPairID := utils.EmptyString
		if waitingContribution.PoolPairID() == utils.EmptyString {
			poolPairID = generatePoolPairKey(waitingContribution.TokenID(), metaData.TokenID(), waitingContribution.TxReqID())
		} else {
			poolPairID = waitingContribution.PoolPairID()
		}
		incomingWaitingContribution := *statedb.NewPdexv3ContributionStateWithValue(
			poolPairID, metaData.ReceiveAddress(), metaData.RefundAddress(),
			metaData.TokenID(), txReqID, metaData.TokenAmount(),
			metaData.Amplifier(), shardID,
		)
		poolPair, found := poolPairs[poolPairID]
		if !found {
			poolPairs[poolPairID] = *initPoolPairState(waitingContribution, incomingWaitingContribution)
			poolPair := poolPairs[poolPairID]
			nfctID := poolPair.addShare(poolPairID, poolPair.state.Token0RealAmount(), beaconHeight)
			inst := instruction.NewMatchAddLiquidityFromMetadata(
				*metaData, txReqID, shardID, poolPairID, nfctID,
			).StringSlice()
			res = append(res, inst)
			continue
		}
		token0Contribution, token1Contribution, token0Metadata, token1Metadata := poolPair.getContributionsByOrder(
			&waitingContribution,
			&incomingWaitingContribution,
			waitingContributionMetaData,
			metaData,
		)
		actualToken0ContributionAmount,
			returnedToken0ContributionAmount,
			actualToken1ContributionAmount,
			returnedToken1ContributionAmount := poolPair.
			computeActualContributedAmounts(&token0Contribution, &token1Contribution)

		if actualToken0ContributionAmount == 0 || actualToken1ContributionAmount == 0 {
			refundInst0 := instruction.NewRefundAddLiquidityFromMetadata(
				token0Metadata, token0Contribution.TxReqID(), token0Contribution.ShardID(),
			).StringSlice()
			res = append(res, refundInst0)
			refundInst1 := instruction.NewRefundAddLiquidityFromMetadata(
				token1Metadata, token1Contribution.TxReqID(), token1Contribution.ShardID(),
			).StringSlice()
			res = append(res, refundInst1)
			continue
		}

		// change token amount
		token0Contribution.SetAmount(actualToken0ContributionAmount)
		token1Contribution.SetAmount(actualToken1ContributionAmount)

		shareAmount := poolPair.updateReserveAndShares(
			token0Contribution.TokenID(), token1Contribution.TokenID(),
			token0Contribution.Amount(), token1Contribution.Amount(),
		)
		nfctID := poolPair.addShare(poolPairID, shareAmount, beaconHeight)
		matchAndReturnInst0 := instruction.NewMatchAndReturnAddLiquidityFromMetadata(
			token0Metadata, token0Contribution.TxReqID(), token0Contribution.ShardID(),
			returnedToken0ContributionAmount, actualToken1ContributionAmount,
			returnedToken1ContributionAmount, token1Contribution.TokenID(), nfctID,
		).StringSlice()
		res = append(res, matchAndReturnInst0)
		matchAndReturnInst1 := instruction.NewMatchAndReturnAddLiquidityFromMetadata(
			token1Metadata, token1Contribution.TxReqID(), token1Contribution.ShardID(),
			returnedToken1ContributionAmount, actualToken0ContributionAmount,
			returnedToken0ContributionAmount, token0Contribution.TokenID(), nfctID,
		).StringSlice()
		res = append(res, matchAndReturnInst1)
	}

	return res, poolPairs, waitingContributions, nil
}

func (sp *stateProducerV2) modifyParams(
	txs []metadata.Transaction,
	beaconHeight uint64,
	params Params,
) ([][]string, Params, error) {
	instructions := [][]string{}

	for _, tx := range txs {
		shardID := byte(tx.GetValidationEnv().ShardID())
		txReqID := *tx.Hash()
		metaData, ok := tx.GetMetadata().(*metadataPdexv3.ParamsModifyingRequest)
		if !ok {
			return instructions, params, errors.New("Can not parse params modifying metadata")
		}

		// check conditions
		metadataParams := metaData.Pdexv3Params
		newParams := Params(metadataParams)
		isValidParams := isValidPdexv3Params(newParams)

		status := ""
		if isValidParams {
			status = metadataPdexv3.RequestAcceptedChainStatus
			params = newParams
		} else {
			status = metadataPdexv3.RequestRejectedChainStatus
		}

		inst := buildModifyParamsInst(
			metadataParams,
			shardID,
			txReqID,
			status,
		)
		instructions = append(instructions, inst)
	}

	return instructions, params, nil
}
