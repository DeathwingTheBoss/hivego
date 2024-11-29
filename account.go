package hivego

import (
	"encoding/json"

	"github.com/deathwingtheboss/hivego/types"
)

func (h *HiveRpcNode) GetAccount(accountNames []string) ([]types.AccountData, error) {
	params := [][]string{accountNames}
	var query = hrpcQuery{
		method: "condenser_api.get_accounts",
		params: params,
	}
	endpoint := h.address
	res, err := h.rpcExec(endpoint, query)
	if err != nil {
		return nil, err
	}

	var accountData []types.AccountData
	err = json.Unmarshal(res, &accountData)
	if err != nil {
		return nil, err
	}
	return accountData, nil
}
