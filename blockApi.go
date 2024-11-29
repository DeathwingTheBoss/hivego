package hivego

import (
	"encoding/json"
	"log"
	"time"

	"github.com/deathwingtheboss/hivego/types"
)

const (
	failureWaitTime = 2500 * time.Millisecond
	retryWaitTime   = 1000 * time.Millisecond
)

func (h *HiveRpcNode) GetBlockRange(startBlock int, count int) (<-chan types.Block, error) {
	if h.MaxConn < 10 {
		h.MaxConn = 10
	}
	if h.MaxBatch < 4 {
		h.MaxBatch = 4
	}

	blockChan := make(chan types.Block)
	go func() {
		defer close(blockChan)
		for i := startBlock; i < startBlock+count; {
			blocks, err := h.fetchBlockInRange(i, count)
			if err != nil {
				log.Printf("Error fetching block range starting from %d: %v\n. Retrying in 3 seconds...", i, err)
				time.Sleep(failureWaitTime)
				continue
			}

			for _, block := range blocks {
				blockChan <- block
				i++
			}

			time.Sleep(retryWaitTime)
		}
	}()
	return blockChan, nil
}

func (h *HiveRpcNode) GetBlock(blockNum int) (types.Block, error) {
	blocks, err := h.fetchBlock([]types.GetBlockQueryParams{{BlockNum: blockNum}})
	if err != nil || len(blocks) == 0 {
		return types.Block{}, err
	}
	return blocks[0], nil
}

func (h *HiveRpcNode) StreamBlocks() (<-chan types.Block, error) {
	blockChan := make(chan types.Block)

	go func() {
		dynProps := hrpcQuery{method: "condenser_api.get_dynamic_global_properties", params: []string{}}
		res, err := h.rpcExec(h.address, dynProps)
		if err != nil {
			log.Fatalf("Failed to fetch dynamic global properties: %v", err)
			close(blockChan)
			return
		}

		var props globalProps
		err = json.Unmarshal(res, &props)
		if err != nil {
			log.Fatalf("Failed to unmarshal dynamic global properties: %v", err)
			close(blockChan)
			return
		}

		currentBlock := props.HeadBlockNumber

		for {
			blockData, err := h.GetBlock(currentBlock)
			if err != nil {
				log.Printf("Error fetching block %d: %v\n. Retrying in 3 seconds...", currentBlock, err)
				time.Sleep(failureWaitTime)
				continue
			}

			blockChan <- blockData
			currentBlock++
			time.Sleep(retryWaitTime)
		}
	}()

	return blockChan, nil
}

func (h *HiveRpcNode) fetchBlockInRange(startBlock, count int) ([]types.Block, error) {
	params := types.GetBlockRangeQueryParams{StartingBlockNum: startBlock, Count: count}
	query := hrpcQuery{method: "block_api.get_block_range", params: params}
	queries := []hrpcQuery{query}

	endpoint := h.address
	res, err := h.rpcExecBatchFast(endpoint, queries)
	if err != nil {
		return nil, err
	}

	var blockRangeResponses []struct {
		ID      int    `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  struct {
			Blocks []types.Block `json:"blocks"`
		} `json:"result"`
	}

	err = json.Unmarshal(res[0], &blockRangeResponses)
	if err != nil {
		return nil, err
	}

	var blocks []types.Block
	for _, blockRangeResponse := range blockRangeResponses {
		blocks = append(blocks, blockRangeResponse.Result.Blocks...)
	}
	return blocks, nil
}

func (h *HiveRpcNode) fetchBlock(params []types.GetBlockQueryParams) ([]types.Block, error) {
	var queries []hrpcQuery
	for _, param := range params {
		query := hrpcQuery{method: "block_api.get_block", params: param}
		queries = append(queries, query)
	}

	endpoint := h.address
	res, err := h.rpcExecBatchFast(endpoint, queries)
	if err != nil {
		return nil, err
	}

	var blockResponses []struct {
		ID      int    `json:"id"`
		JsonRPC string `json:"jsonrpc"`
		Result  struct {
			Block types.Block `json:"block"`
		} `json:"result"`
	}

	err = json.Unmarshal(res[0], &blockResponses)
	if err != nil {
		return nil, err
	}

	var blocks []types.Block
	for _, blockResponse := range blockResponses {
		blocks = append(blocks, blockResponse.Result.Block)
	}
	return blocks, nil
}
