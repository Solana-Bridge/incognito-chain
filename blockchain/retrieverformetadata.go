package blockchain

import (
	"bytes"
	"fmt"

	"github.com/ninjadotorg/constant/blockchain/params"
	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
	"github.com/ninjadotorg/constant/metadata"
	privacy "github.com/ninjadotorg/constant/privacy"
	"github.com/pkg/errors"
)

func (self *BlockChain) GetDatabase() database.DatabaseInterface {
	return self.config.DataBase
}

func (self *BlockChain) GetTxChainHeight(tx metadata.Transaction) (uint64, error) {
	shardID := common.GetShardIDFromLastByte(tx.GetSenderAddrLastByte())
	return self.GetChainHeight(shardID), nil
}

func (self *BlockChain) GetChainHeight(shardID byte) uint64 {
	return self.BestState.Shard[shardID].ShardHeight
}

func (self *BlockChain) GetBeaconHeight() uint64 {
	return self.BestState.Beacon.BeaconHeight
}

func (self *BlockChain) GetBoardPubKeys(boardType byte) [][]byte {
	if boardType == common.DCBBoard {
		return self.GetDCBBoardPubKeys()
	} else {
		return self.GetGOVBoardPubKeys()
	}
}

func (self *BlockChain) GetDCBBoardPubKeys() [][]byte {
	pubkeys := [][]byte{}
	for _, addr := range self.BestState.Beacon.StabilityInfo.DCBGovernor.BoardPaymentAddress {
		pubkeys = append(pubkeys, addr.Pk[:])
	}
	return pubkeys
}

func (self *BlockChain) GetGOVBoardPubKeys() [][]byte {
	pubkeys := [][]byte{}
	for _, addr := range self.BestState.Beacon.StabilityInfo.GOVGovernor.BoardPaymentAddress {
		pubkeys = append(pubkeys, addr.Pk[:])
	}
	return pubkeys
}
func (self *BlockChain) GetBoardPaymentAddress(boardType byte) []privacy.PaymentAddress {
	if boardType == common.DCBBoard {
		return self.BestState.Beacon.StabilityInfo.DCBGovernor.BoardPaymentAddress
	}
	return self.BestState.Beacon.StabilityInfo.GOVGovernor.BoardPaymentAddress
}

func ListPubKeyFromListPayment(listPaymentAddresses []privacy.PaymentAddress) [][]byte {
	pubKeys := make([][]byte, 0)
	for _, i := range listPaymentAddresses {
		pubKeys = append(pubKeys, i.Pk)
	}
	return pubKeys
}

func (self *BlockChain) GetDCBParams() params.DCBParams {
	return self.BestState.Beacon.StabilityInfo.DCBConstitution.DCBParams
}

func (self *BlockChain) GetGOVParams() params.GOVParams {
	return self.BestState.Beacon.StabilityInfo.GOVConstitution.GOVParams
}

func (self *BlockChain) GetLoanReq(loanID []byte) (*common.Hash, error) {
	key := getLoanRequestKeyBeacon(loanID)
	reqHash, ok := self.BestState.Beacon.Params[key]
	if !ok {
		return nil, errors.Errorf("Loan request with ID %x not found", loanID)
	}
	resp, err := common.NewHashFromStr(reqHash)
	return resp, err
}

// GetLoanResps returns all responses of a given loanID
func (self *BlockChain) GetLoanResps(loanID []byte) ([][]byte, []metadata.ValidLoanResponse, error) {
	key := getLoanResponseKeyBeacon(loanID)
	senders := [][]byte{}
	responses := []metadata.ValidLoanResponse{}
	if data, ok := self.BestState.Beacon.Params[key]; ok {
		lrds, err := parseLoanResponseValueBeacon(data)
		if err != nil {
			return nil, nil, err
		}
		for _, lrd := range lrds {
			senders = append(senders, lrd.SenderPubkey)
			responses = append(responses, lrd.Response)
		}
	}
	return senders, responses, nil
}

func (self *BlockChain) GetLoanPayment(loanID []byte) (uint64, uint64, uint64, error) {
	return self.config.DataBase.GetLoanPayment(loanID)
}

func (self *BlockChain) GetLoanRequestMeta(loanID []byte) (*metadata.LoanRequest, error) {
	reqHash, err := self.GetLoanReq(loanID)
	if err != nil {
		return nil, err
	}
	_, _, _, txReq, err := self.GetTransactionByHash(reqHash)
	if err != nil {
		return nil, err
	}
	requestMeta := txReq.GetMetadata().(*metadata.LoanRequest)
	return requestMeta, nil
}

func (self *BlockChain) parseProposalCrowdsaleData(proposalTxHash *common.Hash, saleID []byte) *params.SaleData {
	var saleData *params.SaleData
	_, _, _, proposalTx, err := self.GetTransactionByHash(proposalTxHash)
	if err == nil {
		proposalMeta := proposalTx.GetMetadata().(*metadata.SubmitDCBProposalMetadata)
		fmt.Printf("[db] proposal cs data: %+v\n", proposalMeta)
		for _, data := range proposalMeta.DCBParams.ListSaleData {
			fmt.Printf("[db] data ptr: %p, data: %+v\n", &data, data)
			if bytes.Equal(data.SaleID, saleID) {
				saleData = &data
				saleData.SetProposalTxHash(*proposalTxHash)
				break
			}
		}
	}
	return saleData
}

func (self *BlockChain) GetCrowdsaleData(saleID []byte) (*params.SaleData, error) {
	key := getSaleDataKeyBeacon(saleID)
	if value, ok := self.BestState.Beacon.Params[key]; ok {
		saleData, err := parseSaleDataValueBeacon(value)
		if err != nil {
			return nil, err
		}
		return saleData, nil
	} else {
		return nil, errors.New("Error getting SaleData from beacon best state")
	}
}

func (self *BlockChain) GetAllCrowdsales() ([]*params.SaleData, error) {
	saleDataList := []*params.SaleData{}
	saleIDs, proposalTxHashes, buyingAmounts, sellingAmounts, err := self.config.DataBase.GetAllCrowdsales()
	if err == nil {
		for i, hash := range proposalTxHashes {
			saleData := self.parseProposalCrowdsaleData(&hash, saleIDs[i])
			if saleData != nil {
				saleData.BuyingAmount = buyingAmounts[i]
				saleData.SellingAmount = sellingAmounts[i]
			}
			saleDataList = append(saleDataList, saleData)
		}
	}
	return saleDataList, err
}

func (self *BlockChain) GetCMB(mainAccount []byte) (privacy.PaymentAddress, []privacy.PaymentAddress, uint64, *common.Hash, uint8, uint64, error) {
	reserveAcc, members, capital, hash, state, fine, err := self.config.DataBase.GetCMB(mainAccount)
	if err != nil {
		return privacy.PaymentAddress{}, nil, 0, nil, 0, 0, err
	}

	memberAddresses := []privacy.PaymentAddress{}
	for _, member := range members {
		memberAddress := (&privacy.PaymentAddress{}).SetBytes(member)
		memberAddresses = append(memberAddresses, *memberAddress)
	}

	txHash, _ := (&common.Hash{}).NewHash(hash)
	reserve := (&privacy.PaymentAddress{}).SetBytes(reserveAcc)
	return *reserve, memberAddresses, capital, txHash, state, fine, nil
}

func (self *BlockChain) GetCMBResponse(mainAccount []byte) ([][]byte, error) {
	return self.config.DataBase.GetCMBResponse(mainAccount)
}

func (self *BlockChain) GetDepositSend(contractID []byte) ([]byte, error) {
	return self.config.DataBase.GetDepositSend(contractID)
}

func (self *BlockChain) GetWithdrawRequest(contractID []byte) ([]byte, uint8, error) {
	return self.config.DataBase.GetWithdrawRequest(contractID)
}
