package blockchain

import (
	"context"
	"log"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	"gswarm-sidecar/internal/config"
	"gswarm-sidecar/internal/processor"
)

type Monitor struct {
	cfg       *config.Config
	processor *processor.Processor
}

func New(cfg *config.Config, processor *processor.Processor) *Monitor {
	return &Monitor{
		cfg:       cfg,
		processor: processor,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	log.Printf("[blockchain] Monitor Start: initializing connection to RPC %s", m.cfg.Blockchain.RPCURL)
	client, err := ethclient.Dial(m.cfg.Blockchain.RPCURL)
	if err != nil {
		log.Fatalf("[blockchain] Failed to connect to Ethereum RPC: %v", err)
	}

	log.Printf("[blockchain] Connected to Ethereum RPC at %s", m.cfg.Blockchain.RPCURL)
	contractAddress := common.HexToAddress(m.cfg.Blockchain.ContractAddress)
	log.Printf("[blockchain] Parsing contract ABI from config")
	contractABI, err := abi.JSON(strings.NewReader(m.cfg.Blockchain.ContractABI))
	if err != nil {
		log.Fatalf("[blockchain] Failed to parse contract ABI: %v", err)
	}
	defer client.Close()

	const defaultPollIntervalSeconds = 60
	pollInterval := time.Duration(m.cfg.Blockchain.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Duration(defaultPollIntervalSeconds) * time.Second
	}

	log.Printf("[blockchain] Poll interval set to %v", pollInterval)
	var lastBlock uint64
	log.Printf("[blockchain] Entering pollBlockchain loop")
	m.pollBlockchain(ctx, client, contractAddress, &contractABI, pollInterval, lastBlock)
	log.Printf("[blockchain] Exiting pollBlockchain loop")
}

func (m *Monitor) pollBlockchain(
	ctx context.Context,
	client *ethclient.Client,
	contractAddress common.Address,
	contractABI *abi.ABI,
	pollInterval time.Duration,
	lastBlock uint64,
) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	// Query immediately on startup
	m.pollOnce(ctx, client, contractAddress, contractABI, &lastBlock)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[blockchain] Context done, stopping pollBlockchain")
			return
		case <-ticker.C:
			m.pollOnce(ctx, client, contractAddress, contractABI, &lastBlock)
		}
	}
}

func (m *Monitor) pollOnce(
	ctx context.Context,
	client *ethclient.Client,
	contractAddress common.Address,
	contractABI *abi.ABI,
	lastBlock *uint64,
) {
	log.Printf("[blockchain] Poll tick: getting current block number")
	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		log.Printf("[blockchain] Failed to get current block: %v", err)
		return
	}

	peerId := m.cfg.Blockchain.NodePeerID
	if peerId == "" {
		log.Printf("[blockchain] No peerId configured, skipping blockchain stats poll")
		return
	}

	// getVoterVoteCount(peerId)
	var participation uint64
	voteCountOut, err := contractABI.Pack("getVoterVoteCount", peerId)
	if err != nil {
		log.Printf("[blockchain] Failed to pack getVoterVoteCount: %v", err)
	} else {
		res, callErr := client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddress,
			Data: voteCountOut,
		}, nil)
		if callErr != nil {
			log.Printf("[blockchain] Call to getVoterVoteCount failed: %v", callErr)
		} else {
			out, unpackErr := contractABI.Unpack("getVoterVoteCount", res)
			if unpackErr != nil {
				log.Printf("[blockchain] Failed to unpack getVoterVoteCount: %v", unpackErr)
			} else if len(out) > 0 {
				if v, ok := out[0].(*big.Int); ok {
					participation = v.Uint64()
				}
			}
		}
	}

	// getTotalRewards([peerId])
	var totalRewards int64
	rewardsOut, err := contractABI.Pack("getTotalRewards", []string{peerId})
	if err != nil {
		log.Printf("[blockchain] Failed to pack getTotalRewards: %v", err)
	} else {
		res, callErr := client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddress,
			Data: rewardsOut,
		}, nil)
		if callErr != nil {
			log.Printf("[blockchain] Call to getTotalRewards failed: %v", callErr)
		} else {
			out, unpackErr := contractABI.Unpack("getTotalRewards", res)
			if unpackErr != nil {
				log.Printf("[blockchain] Failed to unpack getTotalRewards: %v", unpackErr)
			} else if len(out) > 0 {
				if arr, ok := out[0].([]*big.Int); ok && len(arr) > 0 {
					totalRewards = arr[0].Int64()
				}
			}
		}
	}

	// getTotalWins(peerId)
	var totalWins uint64
	winsOut, err := contractABI.Pack("getTotalWins", peerId)
	if err != nil {
		log.Printf("[blockchain] Failed to pack getTotalWins: %v", err)
	} else {
		res, callErr := client.CallContract(ctx, ethereum.CallMsg{
			To:   &contractAddress,
			Data: winsOut,
		}, nil)
		if callErr != nil {
			log.Printf("[blockchain] Call to getTotalWins failed: %v", callErr)
		} else {
			out, unpackErr := contractABI.Unpack("getTotalWins", res)
			if unpackErr != nil {
				log.Printf("[blockchain] Failed to unpack getTotalWins: %v", unpackErr)
			} else if len(out) > 0 {
				if v, ok := out[0].(*big.Int); ok {
					totalWins = v.Uint64()
				}
			}
		}
	}

	metrics := &processor.BlockchainMetrics{
		Participation: participation,
		TotalRewards:  totalRewards,
		TotalWins:     totalWins,
		BlockNumber:   currentBlock,
	}

	log.Printf("[blockchain] Blockchain stats: participation=%d, total_rewards=%d, total_wins=%d, block=%d", participation, totalRewards, totalWins, currentBlock)
	if err := m.processor.ProcessBlockchain(ctx, metrics); err != nil {
		log.Printf("[blockchain] Failed to process blockchain metrics: %v", err)
	}

	*lastBlock = currentBlock
}

func parseEvent(vLog *types.Log, contractABI *abi.ABI) (processor.ContractEvent, bool) {
	// TODO: Add case for vote-related events here, e.g., VoteSubmitted, based on the contract ABI. Need details on event name and parameters.
	event := processor.ContractEvent{
		Timestamp: time.Now(),
		BlockHash: vLog.BlockHash.Hex(),
		TxHash:    vLog.TxHash.Hex(),
		Data:      make(map[string]interface{}),
	}
	switch vLog.Topics[0] {
	case contractABI.Events["RewardSubmitted"].ID:
		parsed, err := contractABI.Unpack("RewardSubmitted", vLog.Data)
		if err != nil {
			log.Printf("Failed to unpack RewardSubmitted: %v", err)
			return event, false
		}
		event.EventType = "RewardSubmitted"
		event.Data["account"] = parsed[0].(common.Address).Hex()
		event.Data["roundNumber"] = parsed[1].(*big.Int).String()
		event.Data["stageNumber"] = parsed[2].(*big.Int).String()
		event.Data["reward"] = parsed[3].(*big.Int).String()
		event.Data["peerId"] = parsed[4].(string)
		return event, true
	case contractABI.Events["CumulativeRewardsUpdated"].ID:
		parsed, err := contractABI.Unpack("CumulativeRewardsUpdated", vLog.Data)
		if err != nil {
			log.Printf("Failed to unpack CumulativeRewardsUpdated: %v", err)
			return event, false
		}
		event.EventType = "CumulativeRewardsUpdated"
		event.Data["account"] = parsed[0].(common.Address).Hex()
		event.Data["peerId"] = parsed[1].(string)
		event.Data["totalRewards"] = parsed[2].(*big.Int).String()
		return event, true
	case contractABI.Events["VoteSubmitted"].ID:
		parsed, err := contractABI.Unpack("VoteSubmitted", vLog.Data)
		if err != nil {
			log.Printf("Failed to unpack VoteSubmitted: %v", err)
			return event, false
		}
		event.EventType = "VoteSubmitted"
		event.Data["account"] = parsed[0].(common.Address).Hex()
		event.Data["peerId"] = parsed[1].(string)
		event.Data["roundNumber"] = parsed[2].(*big.Int).String()
		event.Data["stageNumber"] = parsed[3].(*big.Int).String()
		event.Data["voteType"] = parsed[4].(string)
		event.Data["voteValue"] = parsed[5].(bool)
		return event, true
	default:
		return event, false
	}
}
