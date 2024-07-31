// Copyright 2024 NASD Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ve

import (
	"encoding/json"
	"fmt"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
)

func ExtendVoteHandler(manager *module.Manager, ordering []string) sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		extension := make(map[string][]byte)

		for _, name := range ordering {
			m := manager.Modules[name]
			if m == nil {
				return nil, fmt.Errorf("module %s does not exist", name)
			}

			handler, ok := m.(HasVoteExtension)
			if !ok {
				return nil, fmt.Errorf("module %s does not implement correct interface", name)
			}

			res, err := handler.ExtendVote(ctx, req)
			if err != nil {
				return nil, err
			}

			extension[name] = res.VoteExtension
		}

		bz, err := json.Marshal(extension)
		return &abci.ResponseExtendVote{VoteExtension: bz}, err
	}
}

func VerifyExtensionHandler(manager *module.Manager, ordering []string) sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		extension := make(map[string][]byte)
		err := json.Unmarshal(req.VoteExtension, &extension)
		if err != nil {
			return nil, err
		}

		for _, name := range ordering {
			m := manager.Modules[name]
			if m == nil {
				return nil, fmt.Errorf("module %s does not exist", name)
			}

			handler, ok := m.(HasVoteExtension)
			if !ok {
				return nil, fmt.Errorf("module %s does not implement correct interface", name)
			}

			req.VoteExtension = extension[name]
			res, err := handler.VerifyVote(ctx, req)
			if err != nil {
				return res, err
			}
		}

		return &abci.ResponseVerifyVoteExtension{Status: abci.ResponseVerifyVoteExtension_ACCEPT}, nil
	}
}

func PrepareProposalHandler(manager *module.Manager, ordering []string) sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		if req.Height <= ctx.ConsensusParams().Abci.VoteExtensionsEnableHeight {
			return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
		}

		rawExtension := GetVoteExtension(req.LocalLastCommit)
		var extension map[string][]byte
		err := json.Unmarshal(rawExtension, &extension)
		if err != nil {
			return nil, err
		}

		for _, name := range ordering {
			// NOTE: We can safely ignore error cases as they were already
			//  caught in ExtendVote and VerifyVote.
			m := manager.Modules[name]
			handler, _ := m.(HasVoteExtension)

			res, err := handler.PrepareProposal(ctx, req, extension[name])
			if err != nil {
				return nil, err
			}

			req.Txs = res.Txs
		}

		return &abci.ResponsePrepareProposal{Txs: req.Txs}, nil
	}
}

func ProcessProposalHandler(manager *module.Manager, ordering []string) sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		for i, name := range ordering {
			// NOTE: We can safely ignore error cases as they were already
			//  caught in ExtendVote and VerifyVote.
			m := manager.Modules[name]
			handler, _ := m.(HasVoteExtension)

			index := len(ordering) - i - 1
			res, err := handler.ProcessProposal(ctx, req, index)
			if err != nil || res.Status != abci.ResponseProcessProposal_ACCEPT {
				return res, err
			}
		}

		return &abci.ResponseProcessProposal{Status: abci.ResponseProcessProposal_ACCEPT}, nil
	}
}

func PreBlocker(manager *module.Manager, ordering []string) sdk.PreBlocker {
	return func(ctx sdk.Context, req *abci.RequestFinalizeBlock) (*sdk.ResponsePreBlock, error) {
		for i, name := range ordering {
			// NOTE: We can safely ignore error cases as they were already
			//  caught in ExtendVote and VerifyVote.
			m := manager.Modules[name]
			handler, _ := m.(HasVoteExtension)

			index := len(ordering) - i - 1
			err := handler.PreBlocker(ctx, req, index)
			if err != nil {
				return nil, err
			}
		}

		return &sdk.ResponsePreBlock{ConsensusParamsChanged: false}, nil
	}
}

//

type entry struct {
	Extension []byte
	Power     int64
}

func GetVoteExtension(info abci.ExtendedCommitInfo) []byte {
	weights := make(map[string]int64)
	for _, vote := range info.Votes {
		weights[string(vote.VoteExtension)] += vote.Validator.Power
	}

	var entries []entry
	for extension, power := range weights {
		entries = append(entries, entry{Extension: []byte(extension), Power: power})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Power > entries[j].Power
	})

	return entries[0].Extension
}
