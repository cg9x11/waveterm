// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package wshclient

import (
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
	"github.com/wavetermdev/waveterm/pkg/wshutil"
)

// command "dismissonboarding", handled by the tab RPC route
func DismissOnboardingCommand(w *wshutil.WshRpc, opts *wshrpc.RpcOpts) error {
	_, err := sendRpcRequestCallHelper[any](w, "dismissonboarding", nil, opts)
	return err
}

// command "quitapp", handled by the electron RPC route
func QuitAppCommand(w *wshutil.WshRpc, force bool, opts *wshrpc.RpcOpts) error {
	_, err := sendRpcRequestCallHelper[any](w, "quitapp", force, opts)
	return err
}
