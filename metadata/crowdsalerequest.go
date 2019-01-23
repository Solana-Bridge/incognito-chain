package metadata

import (
	"bytes"
	"encoding/hex"
	"strconv"

	"github.com/ninjadotorg/constant/common"
	"github.com/ninjadotorg/constant/database"
	"github.com/ninjadotorg/constant/privacy"
	"github.com/ninjadotorg/constant/wallet"
	"github.com/pkg/errors"
)

// CrowdsaleRequest represents a buying request created by user to send to DCB
type CrowdsaleRequest struct {
	PaymentAddress privacy.PaymentAddress
	SaleID         []byte

	PriceLimit uint64 // max price set by user

	// PriceLimit and Amount is in selling asset: i.e., tx is valid only when price(SellingAsset)/price(BuyingAsset) <= PriceLimit
	LimitSellingAssetPrice bool

	ValidUntil uint64
	MetadataBase
}

func NewCrowdsaleRequest(csReqData map[string]interface{}) (*CrowdsaleRequest, error) {
	errSaver := &ErrorSaver{}
	saleIDStr, okID := csReqData["SaleID"].(string)
	saleID, errSale := hex.DecodeString(saleIDStr)
	priceLimit, okPrice := csReqData["PriceLimit"].(float64)
	validUntil, okValid := csReqData["ValidUntil"].(float64)
	paymentAddressStr, okAddr := csReqData["PaymentAddress"].(string)
	limitSellingAsset, okLimit := csReqData["LimitSellingAssetPrice"].(bool)
	keyWallet, errPayment := wallet.Base58CheckDeserialize(paymentAddressStr)

	if !okID || !okPrice || !okValid || !okAddr || !okLimit {
		return nil, errors.Errorf("Error parsing crowdsale request data")
	}
	if errSaver.Save(errSale, errPayment) != nil {
		return nil, errSaver.Get()
	}

	result := &CrowdsaleRequest{
		PaymentAddress:         keyWallet.KeySet.PaymentAddress,
		SaleID:                 saleID,
		PriceLimit:             uint64(priceLimit),
		ValidUntil:             uint64(validUntil),
		LimitSellingAssetPrice: limitSellingAsset,
	}
	result.Type = CrowdsaleRequestMeta
	return result, nil
}

func (csReq *CrowdsaleRequest) ValidateTxWithBlockChain(txr Transaction, bcr BlockchainRetriever, chainID byte, db database.DatabaseInterface) (bool, error) {
	// Check if sale exists and ongoing
	saleData, err := bcr.GetCrowdsaleData(csReq.SaleID)
	if err != nil {
		return false, err
	}
	// TODO(@0xbunyip): get height of beacon chain on new consensus
	height, err := bcr.GetTxChainHeight(txr)
	if err != nil || saleData.EndBlock >= height {
		return false, errors.Errorf("Crowdsale ended")
	}

	// Check if request is still valid
	if height >= csReq.ValidUntil {
		return false, errors.Errorf("Crowdsale request is not valid anymore")
	}

	// Check if asset is sent to correct address
	// TODO(@0xbunyip): validate type and amount of asset sent and if price limit is not violated
	if saleData.BuyingAsset.IsEqual(&common.ConstantID) {
		keyWalletBurnAccount, _ := wallet.Base58CheckDeserialize(common.BurningAddress)
		unique, pubkey, _ := txr.GetUniqueReceiver()
		if !unique || !bytes.Equal(pubkey, keyWalletBurnAccount.KeySet.PaymentAddress.Pk[:]) {
			return false, errors.Errorf("Crowdsale request must send CST to DCBAddress")
		}
	} else {
		keyWalletDCBAccount, _ := wallet.Base58CheckDeserialize(common.DCBAddress)
		unique, pubkey, _ := txr.GetTokenUniqueReceiver()
		if !unique || !bytes.Equal(pubkey, keyWalletDCBAccount.KeySet.PaymentAddress.Pk[:]) {
			return false, errors.Errorf("Crowdsale request must send tokens to BurningAddress")
		}
	}

	return true, nil
}

func (csReq *CrowdsaleRequest) ValidateSanityData(bcr BlockchainRetriever, txr Transaction) (bool, bool, error) {
	if len(csReq.PaymentAddress.Pk) == 0 {
		return false, false, errors.New("Wrong request info's payment address")
	}
	return false, true, nil
}

func (csReq *CrowdsaleRequest) ValidateMetadataByItself() bool {
	// The validation just need to check at tx level, so returning true here
	// TODO(@0xbunyip): accept only some pairs of assets
	return true
}

func (csReq *CrowdsaleRequest) Hash() *common.Hash {
	record := csReq.PaymentAddress.String()
	record += string(csReq.SaleID)
	record += string(csReq.PriceLimit)
	record += string(csReq.ValidUntil)
	record += strconv.FormatBool(csReq.LimitSellingAssetPrice)

	// final hash
	record += csReq.MetadataBase.Hash().String()
	hash := common.DoubleHashH([]byte(record))
	return &hash
}
