package committeestate

import (
	"github.com/incognitochain/incognito-chain/common"
	"github.com/incognitochain/incognito-chain/incognitokey"
	"github.com/incognitochain/incognito-chain/instruction"
	"github.com/incognitochain/incognito-chain/metadata"
	"github.com/incognitochain/incognito-chain/privacy"
)

//BeaconCommitteeEngine :
type BeaconCommitteeEngine interface {
	Version() uint
	Clone() BeaconCommitteeEngine
	GetBeaconHeight() uint64
	GetBeaconHash() common.Hash
	GetBeaconCommittee() []incognitokey.CommitteePublicKey
	GetBeaconSubstitute() []incognitokey.CommitteePublicKey
	GetCandidateShardWaitingForCurrentRandom() []incognitokey.CommitteePublicKey
	GetCandidateBeaconWaitingForCurrentRandom() []incognitokey.CommitteePublicKey
	GetCandidateShardWaitingForNextRandom() []incognitokey.CommitteePublicKey
	GetCandidateBeaconWaitingForNextRandom() []incognitokey.CommitteePublicKey
	GetOneShardCommittee(shardID byte) []incognitokey.CommitteePublicKey
	GetShardCommittee() map[byte][]incognitokey.CommitteePublicKey
	GetUncommittedCommittee() map[byte][]incognitokey.CommitteePublicKey
	GetOneShardSubstitute(shardID byte) []incognitokey.CommitteePublicKey
	GetShardSubstitute() map[byte][]incognitokey.CommitteePublicKey
	GetAutoStaking() map[string]bool
	GetStakingTx() map[string]common.Hash
	GetRewardReceiver() map[string]privacy.PaymentAddress
	GetAllCandidateSubstituteCommittee() []string
	Commit(*BeaconCommitteeStateHash) error
	AbortUncommittedBeaconState()
	UpdateCommitteeState(env *BeaconCommitteeStateEnvironment) (
		*BeaconCommitteeStateHash,
		*CommitteeChange,
		[][]string,
		error)
	InitCommitteeState(env *BeaconCommitteeStateEnvironment)
	GenerateAllSwapShardInstructions(env *BeaconCommitteeStateEnvironment) ([]*instruction.SwapShardInstruction, error)
	SplitReward(*BeaconCommitteeStateEnvironment) (map[common.Hash]uint64, map[common.Hash]uint64, map[common.Hash]uint64, map[common.Hash]uint64, error)
	ActiveShards() int
	AssignInstructions(env *BeaconCommitteeStateEnvironment) []*instruction.AssignInstruction
	SyncingValidators() map[byte][]incognitokey.CommitteePublicKey
	NumberOfAssignedCandidates() int
	AddFinishedSyncValidators([]string) error
	GenerateFinishSyncInstructions() ([]*instruction.FinishSyncInstruction, error)
}

//ShardCommitteeEngine :
type ShardCommitteeEngine interface {
	Version() uint
	Clone() ShardCommitteeEngine
	Commit(*ShardCommitteeStateHash) error
	AbortUncommittedShardState()
	UpdateCommitteeState(env ShardCommitteeStateEnvironment) (*ShardCommitteeStateHash,
		*CommitteeChange, error)
	InitCommitteeState(env ShardCommitteeStateEnvironment)
	GetShardCommittee() []incognitokey.CommitteePublicKey
	GetShardSubstitute() []incognitokey.CommitteePublicKey
	CommitteeFromBlock() common.Hash
	ProcessInstructionFromBeacon(env ShardCommitteeStateEnvironment) (*CommitteeChange, error)
	GenerateSwapInstruction(env ShardCommitteeStateEnvironment) (*instruction.SwapInstruction, []string, []string, error)
	BuildTotalTxsFeeFromTxs(txs []metadata.Transaction) map[common.Hash]uint64
}
