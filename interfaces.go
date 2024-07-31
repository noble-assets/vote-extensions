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
	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// HasVoteExtension is an opinionated interface that module's wanting to utilize
// vote extensions must implement.
type HasVoteExtension interface {
	ExtendVote(sdk.Context, *abci.RequestExtendVote) (*abci.ResponseExtendVote, error)
	VerifyVote(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error)
	PrepareProposal(sdk.Context, *abci.RequestPrepareProposal, []byte) (*abci.ResponsePrepareProposal, error)
	ProcessProposal(sdk.Context, *abci.RequestProcessProposal, int) (*abci.ResponseProcessProposal, error)
	PreBlocker(sdk.Context, *abci.RequestFinalizeBlock, int) error
}
