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
	client, err := ethclient.Dial(m.cfg.Blockchain.RPCURL)
	if err != nil {
		log.Fatalf("Failed to connect to Ethereum RPC: %v", err)
	}

	contractAddress := common.HexToAddress(m.cfg.Blockchain.ContractAddress)
	contractABI, err := abi.JSON(strings.NewReader(m.cfg.Blockchain.ContractABI))
	if err != nil {
		log.Fatalf("Failed to parse contract ABI: %v", err)
	}
	defer client.Close()

	const defaultPollIntervalSeconds = 30
	pollInterval := time.Duration(m.cfg.Blockchain.PollInterval) * time.Second
	if pollInterval == 0 {
		pollInterval = time.Duration(defaultPollIntervalSeconds) * time.Second
	}

	var lastBlock uint64
	m.pollBlockchain(ctx, client, contractAddress, &contractABI, pollInterval, lastBlock)
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

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			currentBlock, err := client.BlockNumber(ctx)
			if err != nil {
				log.Printf("Failed to get current block: %v", err)
				continue
			}

			if currentBlock <= lastBlock {
				continue
			}

			query := ethereum.FilterQuery{
				Addresses: []common.Address{contractAddress},
				FromBlock: new(big.Int).SetUint64(lastBlock + 1),
				ToBlock:   new(big.Int).SetUint64(currentBlock),
			}

			var logs []types.Log
			logs, err = client.FilterLogs(ctx, query)
			if err != nil {
				log.Printf("Failed to filter logs: %v", err)
				continue
			}

			var events []processor.ContractEvent
			for i := range logs {
				event, ok := parseEvent(&logs[i], contractABI)
				if ok && (event.Data["account"] == m.cfg.Blockchain.NodeEOA || event.Data["peerId"] == m.cfg.Blockchain.NodePeerID) {
					events = append(events, event)
				}
			}

			if len(events) > 0 {
				metrics := &processor.BlockchainMetrics{
					ContractEvents: events,
					BlockNumber:    currentBlock,
					GasUsed:        0, // TODO: Calculate if needed
				}

				if err := m.processor.ProcessBlockchain(ctx, metrics); err != nil {
					log.Printf("Failed to process blockchain metrics: %v", err)
				}
			}

			lastBlock = currentBlock
		}
	}
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
