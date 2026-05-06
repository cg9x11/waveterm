// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/wavetermdev/waveterm/pkg/waveobj"
	"github.com/wavetermdev/waveterm/pkg/wps"
	"github.com/wavetermdev/waveterm/pkg/wshrpc"
	"github.com/wavetermdev/waveterm/pkg/wshrpc/wshclient"
	"github.com/wavetermdev/waveterm/pkg/wshutil"
)

var automationCmd = &cobra.Command{
	Use:               "automation",
	Aliases:           []string{"auto"},
	Short:             "Machine-readable automation helpers for Wave",
	PersistentPreRunE: preRunSetupRpcClient,
	SilenceUsage:      true,
}

var automationReadyCmd = &cobra.Command{
	Use:   "ready",
	Short: "Emit a ready payload for automation clients",
	RunE:  automationReadyRun,
}

var automationStateCmd = &cobra.Command{
	Use:   "state",
	Short: "Emit a structured state snapshot for a block or the focused block",
	RunE:  automationStateRun,
}

var automationEventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Stream Wave events as JSON lines",
	RunE:  automationEventsRun,
}

var automationWaitCmd = &cobra.Command{
	Use:   "wait",
	Short: "Wait for a Wave event or block/job condition",
	RunE:  automationWaitRun,
}

var automationScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a block screenshot and emit a machine-readable artifact record",
	RunE:  automationScreenshotRun,
}

var automationActionCmd = &cobra.Command{
	Use:   "action",
	Short: "Perform machine-oriented actions against a block",
}

var automationAppCmd = &cobra.Command{
	Use:   "app",
	Short: "Perform machine-oriented actions against the Wave app",
}

var automationAppDismissWelcomeCmd = &cobra.Command{
	Use:   "dismiss-welcome",
	Short: "Dismiss onboarding and persist that it should not reappear",
	RunE:  automationAppDismissWelcomeRun,
}

var automationAppQuitCmd = &cobra.Command{
	Use:   "quit",
	Short: "Quit the running Wave app",
	RunE:  automationAppQuitRun,
}

var automationAppStateCmd = &cobra.Command{
	Use:   "state",
	Short: "Emit a structured app-wide state snapshot",
	RunE:  automationAppStateRun,
}

var automationAppScreenshotCmd = &cobra.Command{
	Use:   "screenshot",
	Short: "Capture a whole-app UI snapshot from the current window",
	RunE:  automationAppScreenshotRun,
}

var automationWorkspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Inspect and control workspaces",
}

var automationWorkspaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List workspaces with their owning windows",
	RunE:  automationWorkspaceListRun,
}

var automationWorkspaceCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWorkspaceCreateRun,
}

var automationWorkspaceStateCmd = &cobra.Command{
	Use:   "state [workspace-id]",
	Short: "Emit a structured state snapshot for a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWorkspaceStateRun,
}

var automationWorkspaceSwitchCmd = &cobra.Command{
	Use:   "switch [workspace-id]",
	Short: "Switch a window to a workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  automationWorkspaceSwitchRun,
}

var automationTabCmd = &cobra.Command{
	Use:   "tab",
	Short: "Inspect and control tabs",
}

var automationTabStateCmd = &cobra.Command{
	Use:   "state [tab-id]",
	Short: "Emit a structured state snapshot for a tab",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationTabStateRun,
}

var automationTabCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a tab in a workspace",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationTabCreateRun,
}

var automationTabSwitchCmd = &cobra.Command{
	Use:   "switch [tab-id]",
	Short: "Activate a tab inside its workspace",
	Args:  cobra.ExactArgs(1),
	RunE:  automationTabSwitchRun,
}

var automationTabCloseCmd = &cobra.Command{
	Use:   "close [tab-id]",
	Short: "Close a tab",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationTabCloseRun,
}

var automationWindowCmd = &cobra.Command{
	Use:   "window",
	Short: "Inspect and control windows",
}

var automationWindowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List open windows",
	RunE:  automationWindowListRun,
}

var automationWindowStateCmd = &cobra.Command{
	Use:   "state [window-id]",
	Short: "Emit a structured state snapshot for a window",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWindowStateRun,
}

var automationWindowCreateCmd = &cobra.Command{
	Use:   "create [workspace-id]",
	Short: "Create and show a new window",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWindowCreateRun,
}

var automationWindowFocusCmd = &cobra.Command{
	Use:   "focus [window-id]",
	Short: "Focus a window",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWindowFocusRun,
}

var automationWindowCloseCmd = &cobra.Command{
	Use:   "close [window-id]",
	Short: "Close a window",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWindowCloseRun,
}

var automationWindowScreenshotCmd = &cobra.Command{
	Use:   "screenshot [window-id]",
	Short: "Capture a whole-window or active-window UI screenshot",
	Args:  cobra.MaximumNArgs(1),
	RunE:  automationWindowScreenshotRun,
}

const currentOnboardingVersion = "v0.14.5"

var automationActionFocusCmd = &cobra.Command{
	Use:   "focus",
	Short: "Focus the target block",
	RunE:  automationActionFocusRun,
}

var automationActionTypeCmd = &cobra.Command{
	Use:   "type [text]",
	Short: "Send text input to the target block controller",
	Args:  cobra.ExactArgs(1),
	RunE:  automationActionTypeRun,
}

var automationActionSignalCmd = &cobra.Command{
	Use:   "signal [name]",
	Short: "Send a signal such as INT or TERM to the target block controller",
	Args:  cobra.ExactArgs(1),
	RunE:  automationActionSignalRun,
}

var automationActionResizeCmd = &cobra.Command{
	Use:   "resize [rows] [cols]",
	Short: "Send a terminal resize event to the target block controller",
	Args:  cobra.ExactArgs(2),
	RunE:  automationActionResizeRun,
}

var (
	automationStateFocused           bool
	automationEventsName             string
	automationEventsScope            string
	automationEventsAll              bool
	automationEventsFollow           bool
	automationEventsMax              int
	automationWaitTimeout            int
	automationWaitEvent              string
	automationWaitScope              string
	automationWaitAll                bool
	automationWaitJobDone            bool
	automationWaitJobState           string
	automationWaitPollMs             int
	automationShotOutput             string
	automationAppQuitForce           bool
	automationWorkspaceId            string
	automationTabId                  string
	automationWindowId               string
	automationTabActivate            bool
	automationWorkspaceIcon          string
	automationWorkspaceColor         string
	automationWorkspaceApplyDefaults bool
)

type automationEnvelope struct {
	Type string `json:"type"`
	Ts   int64  `json:"ts"`
	Data any    `json:"data,omitempty"`
}

type automationReadyPayload struct {
	Version     string            `json:"version"`
	ClientId    string            `json:"clientId"`
	BuildTime   string            `json:"buildTime"`
	ConfigDir   string            `json:"configDir"`
	DataDir     string            `json:"dataDir"`
	RouteId     string            `json:"routeId"`
	BlockId     string            `json:"blockId,omitempty"`
	TabId       string            `json:"tabId,omitempty"`
	WorkspaceId string            `json:"workspaceId,omitempty"`
	ConnName    string            `json:"connName,omitempty"`
	RpcContext  wshrpc.RpcContext `json:"rpcContext"`
}

type automationStateSnapshot struct {
	Millis         int64                    `json:"millis"`
	Target         string                   `json:"target"`
	Focused        *wshrpc.FocusedBlockData `json:"focused,omitempty"`
	Block          *automationBlockSnapshot `json:"block,omitempty"`
	FocusedMatches bool                     `json:"focusedMatches"`
}

type automationBlockSnapshot struct {
	BlockId     string                     `json:"blockId"`
	TabId       string                     `json:"tabId"`
	WorkspaceId string                     `json:"workspaceId"`
	Meta        waveobj.MetaMapType        `json:"meta"`
	RuntimeOpts *waveobj.RuntimeOpts       `json:"runtimeOpts,omitempty"`
	SubBlockIds []string                   `json:"subBlockIds,omitempty"`
	Files       []*wshrpc.WaveFileInfo     `json:"files,omitempty"`
	RTInfo      *waveobj.ObjRTInfo         `json:"rtInfo,omitempty"`
	JobStatus   *wshrpc.BlockJobStatusData `json:"jobStatus,omitempty"`
	ConnStatus  *wshrpc.ConnStatus         `json:"connStatus,omitempty"`
}

type automationArtifactPayload struct {
	ArtifactType string `json:"artifactType"`
	BlockId      string `json:"blockId"`
	WindowId     string `json:"windowId,omitempty"`
	Path         string `json:"path,omitempty"`
	MimeType     string `json:"mimeType"`
	Bytes        int    `json:"bytes"`
	DataURL      string `json:"dataUrl,omitempty"`
}

type automationAppState struct {
	Millis             int64                      `json:"millis"`
	CurrentWindowId    string                     `json:"currentWindowId,omitempty"`
	CurrentWorkspaceId string                     `json:"currentWorkspaceId,omitempty"`
	CurrentTabId       string                     `json:"currentTabId,omitempty"`
	Windows            []*automationWindowState   `json:"windows"`
	Workspaces         []*automationWorkspaceInfo `json:"workspaces"`
}

type automationWorkspaceInfo struct {
	WindowId  string             `json:"windowId,omitempty"`
	Workspace *waveobj.Workspace `json:"workspace"`
	ActiveTab *waveobj.Tab       `json:"activeTab,omitempty"`
	IsCurrent bool               `json:"isCurrent,omitempty"`
}

type automationWorkspaceState struct {
	Millis    int64              `json:"millis"`
	Window    *waveobj.Window    `json:"window,omitempty"`
	Workspace *waveobj.Workspace `json:"workspace"`
	ActiveTab *waveobj.Tab       `json:"activeTab,omitempty"`
	Tabs      []*waveobj.Tab     `json:"tabs"`
	IsCurrent bool               `json:"isCurrent,omitempty"`
}

type automationTabState struct {
	Millis    int64                    `json:"millis"`
	Window    *waveobj.Window          `json:"window,omitempty"`
	Workspace *waveobj.Workspace       `json:"workspace,omitempty"`
	Tab       *waveobj.Tab             `json:"tab"`
	Blocks    []wshrpc.BlocksListEntry `json:"blocks,omitempty"`
	Focused   *wshrpc.FocusedBlockData `json:"focused,omitempty"`
	IsCurrent bool                     `json:"isCurrent,omitempty"`
}

type automationWindowState struct {
	Millis      int64              `json:"millis"`
	Window      *waveobj.Window    `json:"window"`
	Workspace   *waveobj.Workspace `json:"workspace,omitempty"`
	ActiveTab   *waveobj.Tab       `json:"activeTab,omitempty"`
	IsCurrent   bool               `json:"isCurrent,omitempty"`
	IsFocusable bool               `json:"isFocusable"`
}

func init() {
	automationStateCmd.Flags().BoolVar(&automationStateFocused, "focused", false, "use the currently focused block")

	automationEventsCmd.Flags().StringVar(&automationEventsName, "event", "", "event name to subscribe to")
	automationEventsCmd.Flags().StringVar(&automationEventsScope, "scope", "", "restrict to a specific scope")
	automationEventsCmd.Flags().BoolVar(&automationEventsAll, "all-scopes", false, "subscribe across all scopes")
	automationEventsCmd.Flags().BoolVar(&automationEventsFollow, "follow", true, "continue streaming live events after history")
	automationEventsCmd.Flags().IntVar(&automationEventsMax, "history", 0, "emit up to N matching history events before following")

	automationWaitCmd.Flags().IntVar(&automationWaitTimeout, "timeout", 30000, "timeout in milliseconds")
	automationWaitCmd.Flags().StringVar(&automationWaitEvent, "event", "", "wait for a specific event")
	automationWaitCmd.Flags().StringVar(&automationWaitScope, "scope", "", "restrict event waits to a specific scope")
	automationWaitCmd.Flags().BoolVar(&automationWaitAll, "all-scopes", false, "wait across all scopes")
	automationWaitCmd.Flags().BoolVar(&automationWaitJobDone, "job-done", false, "wait until the target block's attached job is done")
	automationWaitCmd.Flags().StringVar(&automationWaitJobState, "job-state", "", "wait until the target block's attached job reaches a specific status")
	automationWaitCmd.Flags().IntVar(&automationWaitPollMs, "poll-ms", 250, "poll interval for job-based waits")

	automationScreenshotCmd.Flags().StringVarP(&automationShotOutput, "output", "o", "", "write screenshot bytes to a path resolved from the initial cwd")
	automationAppQuitCmd.Flags().BoolVar(&automationAppQuitForce, "force", false, "force quit without waiting for graceful shutdown prompts")
	automationAppScreenshotCmd.Flags().StringVarP(&automationShotOutput, "output", "o", "", "write screenshot bytes to a path resolved from the initial cwd")
	automationWorkspaceCreateCmd.Flags().StringVar(&automationWorkspaceIcon, "icon", "", "workspace icon")
	automationWorkspaceCreateCmd.Flags().StringVar(&automationWorkspaceColor, "color", "", "workspace color")
	automationWorkspaceCreateCmd.Flags().BoolVar(&automationWorkspaceApplyDefaults, "apply-defaults", true, "apply default starter content")
	automationWorkspaceStateCmd.Flags().StringVar(&automationWorkspaceId, "workspace", "", "workspace id override")
	automationWorkspaceStateCmd.Flags().StringVar(&automationWorkspaceId, "workspace-id", "", "alias for --workspace")
	automationWorkspaceSwitchCmd.Flags().StringVar(&automationWindowId, "window", "", "window id to switch; defaults to the current window")
	automationWorkspaceSwitchCmd.Flags().StringVar(&automationWindowId, "window-id", "", "alias for --window")
	automationTabStateCmd.Flags().StringVar(&automationTabId, "tab", "", "tab id override")
	automationTabStateCmd.Flags().StringVar(&automationTabId, "tab-id", "", "alias for --tab")
	automationTabCreateCmd.Flags().StringVar(&automationWorkspaceId, "workspace", "", "workspace id to create the tab in")
	automationTabCreateCmd.Flags().StringVar(&automationWorkspaceId, "workspace-id", "", "alias for --workspace")
	automationTabCreateCmd.Flags().BoolVar(&automationTabActivate, "activate", true, "make the new tab active")
	automationTabSwitchCmd.Flags().StringVar(&automationWorkspaceId, "workspace", "", "workspace id override when switching")
	automationTabSwitchCmd.Flags().StringVar(&automationWorkspaceId, "workspace-id", "", "alias for --workspace")
	automationTabCloseCmd.Flags().StringVar(&automationTabId, "tab", "", "tab id override")
	automationTabCloseCmd.Flags().StringVar(&automationTabId, "tab-id", "", "alias for --tab")
	automationTabCloseCmd.Flags().StringVar(&automationWorkspaceId, "workspace", "", "workspace id override when closing")
	automationTabCloseCmd.Flags().StringVar(&automationWorkspaceId, "workspace-id", "", "alias for --workspace")
	automationWindowStateCmd.Flags().StringVar(&automationWindowId, "window", "", "window id override")
	automationWindowStateCmd.Flags().StringVar(&automationWindowId, "window-id", "", "alias for --window")
	automationWindowFocusCmd.Flags().StringVar(&automationWindowId, "window", "", "window id override")
	automationWindowFocusCmd.Flags().StringVar(&automationWindowId, "window-id", "", "alias for --window")
	automationWindowCloseCmd.Flags().StringVar(&automationWindowId, "window", "", "window id override")
	automationWindowCloseCmd.Flags().StringVar(&automationWindowId, "window-id", "", "alias for --window")
	automationWindowScreenshotCmd.Flags().StringVar(&automationWindowId, "window", "", "window id override")
	automationWindowScreenshotCmd.Flags().StringVar(&automationWindowId, "window-id", "", "alias for --window")
	automationWindowScreenshotCmd.Flags().StringVarP(&automationShotOutput, "output", "o", "", "write screenshot bytes to a path resolved from the initial cwd")

	automationCmd.AddCommand(automationReadyCmd)
	automationCmd.AddCommand(automationStateCmd)
	automationCmd.AddCommand(automationEventsCmd)
	automationCmd.AddCommand(automationWaitCmd)
	automationCmd.AddCommand(automationScreenshotCmd)
	automationActionCmd.AddCommand(automationActionFocusCmd)
	automationActionCmd.AddCommand(automationActionTypeCmd)
	automationActionCmd.AddCommand(automationActionSignalCmd)
	automationActionCmd.AddCommand(automationActionResizeCmd)
	automationCmd.AddCommand(automationActionCmd)
	automationAppCmd.AddCommand(automationAppStateCmd)
	automationAppCmd.AddCommand(automationAppScreenshotCmd)
	automationAppCmd.AddCommand(automationAppDismissWelcomeCmd)
	automationAppCmd.AddCommand(automationAppQuitCmd)
	automationCmd.AddCommand(automationAppCmd)
	automationWorkspaceCmd.AddCommand(automationWorkspaceCreateCmd)
	automationWorkspaceCmd.AddCommand(automationWorkspaceListCmd)
	automationWorkspaceCmd.AddCommand(automationWorkspaceStateCmd)
	automationWorkspaceCmd.AddCommand(automationWorkspaceSwitchCmd)
	automationCmd.AddCommand(automationWorkspaceCmd)
	automationTabCmd.AddCommand(automationTabStateCmd)
	automationTabCmd.AddCommand(automationTabCreateCmd)
	automationTabCmd.AddCommand(automationTabSwitchCmd)
	automationTabCmd.AddCommand(automationTabCloseCmd)
	automationCmd.AddCommand(automationTabCmd)
	automationWindowCmd.AddCommand(automationWindowListCmd)
	automationWindowCmd.AddCommand(automationWindowCreateCmd)
	automationWindowCmd.AddCommand(automationWindowStateCmd)
	automationWindowCmd.AddCommand(automationWindowFocusCmd)
	automationWindowCmd.AddCommand(automationWindowCloseCmd)
	automationWindowCmd.AddCommand(automationWindowScreenshotCmd)
	automationCmd.AddCommand(automationWindowCmd)
	rootCmd.AddCommand(automationCmd)
}

func emitAutomationEnvelope(eventType string, data any) error {
	return writeJSONLineStdout(automationEnvelope{
		Type: eventType,
		Ts:   time.Now().UnixMilli(),
		Data: data,
	})
}

func emitAutomationActionLifecycle(action string, blockId string, extra map[string]any, run func() error) error {
	startData := map[string]any{
		"action":  action,
		"blockId": blockId,
	}
	for key, value := range extra {
		startData[key] = value
	}
	if err := emitAutomationEnvelope("action_started", startData); err != nil {
		return err
	}
	if err := run(); err != nil {
		failedData := map[string]any{
			"action":  action,
			"blockId": blockId,
			"error":   err.Error(),
		}
		for key, value := range extra {
			failedData[key] = value
		}
		_ = emitAutomationEnvelope("action_failed", failedData)
		return err
	}
	completedData := map[string]any{
		"action":  action,
		"blockId": blockId,
	}
	for key, value := range extra {
		completedData[key] = value
	}
	return emitAutomationEnvelope("action_completed", completedData)
}

func automationReadyRun(cmd *cobra.Command, args []string) error {
	info, err := wshclient.WaveInfoCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return err
	}
	if info == nil {
		return fmt.Errorf("waveinfo returned no data")
	}
	payload := automationReadyPayload{
		Version:     info.Version,
		ClientId:    info.ClientId,
		BuildTime:   info.BuildTime,
		ConfigDir:   info.ConfigDir,
		DataDir:     info.DataDir,
		RouteId:     RpcClientRouteId,
		BlockId:     os.Getenv("WAVETERM_BLOCKID"),
		TabId:       os.Getenv("WAVETERM_TABID"),
		WorkspaceId: os.Getenv("WAVETERM_WORKSPACEID"),
		ConnName:    os.Getenv("WAVETERM_CONN"),
		RpcContext:  RpcContext,
	}
	return emitAutomationEnvelope("ready", payload)
}

func automationStateRun(cmd *cobra.Command, args []string) error {
	focused, err := wshclient.GetFocusedBlockDataCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		if automationStateFocused {
			return err
		}
		focused = nil
	}

	targetBlockId := ""
	if !automationStateFocused {
		oref, err := resolveBlockArg()
		if err != nil {
			return err
		}
		targetBlockId = oref.OID
	} else if focused != nil {
		targetBlockId = focused.BlockId
	}
	if targetBlockId == "" {
		return fmt.Errorf("no target block resolved")
	}

	blockSnapshot, err := getAutomationBlockSnapshot(targetBlockId)
	if err != nil {
		return err
	}
	snapshot := automationStateSnapshot{
		Millis:         time.Now().UnixMilli(),
		Target:         targetBlockId,
		Focused:        focused,
		Block:          blockSnapshot,
		FocusedMatches: focused != nil && focused.BlockId == targetBlockId,
	}
	return writeJSONStdout(snapshot)
}

func automationEventsRun(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(automationEventsName) == "" {
		return fmt.Errorf("--event is required")
	}
	if automationEventsAll && automationEventsScope != "" {
		return fmt.Errorf("--scope and --all-scopes are mutually exclusive")
	}

	if automationEventsMax > 0 {
		history, err := wshclient.EventReadHistoryCommand(RpcClient, wshrpc.CommandEventReadHistoryData{
			Event:    automationEventsName,
			Scope:    automationEventsScope,
			MaxItems: automationEventsMax,
		}, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return err
		}
		for _, event := range history {
			if err := emitAutomationEnvelope("event", event); err != nil {
				return err
			}
		}
	}
	if !automationEventsFollow {
		return nil
	}

	eventsCh := make(chan *wps.WaveEvent, 16)
	listenerId := RpcClient.EventListener.On(automationEventsName, func(event *wps.WaveEvent) {
		select {
		case eventsCh <- event:
		default:
		}
	})
	defer RpcClient.EventListener.Unregister(automationEventsName, listenerId)
	defer wshclient.EventUnsubCommand(RpcClient, automationEventsName, &wshrpc.RpcOpts{NoResponse: true})

	subReq := wps.SubscriptionRequest{Event: automationEventsName}
	if automationEventsAll {
		subReq.AllScopes = true
	} else if automationEventsScope != "" {
		subReq.Scopes = []string{automationEventsScope}
	} else {
		subReq.AllScopes = true
	}
	if err := wshclient.EventSubCommand(RpcClient, subReq, &wshrpc.RpcOpts{NoResponse: true}); err != nil {
		return err
	}

	for event := range eventsCh {
		if err := emitAutomationEnvelope("event", event); err != nil {
			return err
		}
	}
	return nil
}

func automationWaitRun(cmd *cobra.Command, args []string) error {
	waitModes := 0
	if automationWaitEvent != "" {
		waitModes++
	}
	if automationWaitJobDone {
		waitModes++
	}
	if automationWaitJobState != "" {
		waitModes++
	}
	if waitModes != 1 {
		return fmt.Errorf("select exactly one wait mode: --event, --job-done, or --job-state")
	}

	if automationWaitEvent != "" {
		return automationWaitForEvent()
	}
	return automationWaitForJobState()
}

func automationWaitForEvent() error {
	if automationWaitAll && automationWaitScope != "" {
		return fmt.Errorf("--scope and --all-scopes are mutually exclusive")
	}
	eventsCh := make(chan *wps.WaveEvent, 1)
	listenerId := RpcClient.EventListener.On(automationWaitEvent, func(event *wps.WaveEvent) {
		select {
		case eventsCh <- event:
		default:
		}
	})
	defer RpcClient.EventListener.Unregister(automationWaitEvent, listenerId)
	defer wshclient.EventUnsubCommand(RpcClient, automationWaitEvent, &wshrpc.RpcOpts{NoResponse: true})

	subReq := wps.SubscriptionRequest{Event: automationWaitEvent}
	if automationWaitAll || automationWaitScope == "" {
		subReq.AllScopes = true
	} else {
		subReq.Scopes = []string{automationWaitScope}
	}
	if err := wshclient.EventSubCommand(RpcClient, subReq, &wshrpc.RpcOpts{NoResponse: true}); err != nil {
		return err
	}
	waitData := map[string]any{
		"mode":      "event",
		"event":     automationWaitEvent,
		"scope":     automationWaitScope,
		"allScopes": automationWaitAll || automationWaitScope == "",
		"timeoutMs": automationWaitTimeout,
	}
	if err := emitAutomationEnvelope("wait_started", waitData); err != nil {
		return err
	}
	if err := emitAutomationEnvelope("wait_progress", map[string]any{
		"mode":      "event",
		"event":     automationWaitEvent,
		"scope":     automationWaitScope,
		"allScopes": automationWaitAll || automationWaitScope == "",
		"phase":     "subscribed",
	}); err != nil {
		return err
	}

	timeoutCh := time.After(time.Duration(automationWaitTimeout) * time.Millisecond)
	select {
	case event := <-eventsCh:
		if err := emitAutomationEnvelope("wait_satisfied", event); err != nil {
			return err
		}
		return nil
	case <-timeoutCh:
		_ = emitAutomationEnvelope("wait_timeout", map[string]any{
			"mode":      "event",
			"event":     automationWaitEvent,
			"scope":     automationWaitScope,
			"timeoutMs": automationWaitTimeout,
		})
		return fmt.Errorf("timed out waiting for event %q", automationWaitEvent)
	}
}

func automationWaitForJobState() error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	targetState := automationWaitJobState
	if automationWaitJobDone {
		targetState = "done"
	}

	if err := emitAutomationEnvelope("wait_started", map[string]any{
		"mode":      "job",
		"blockId":   oref.OID,
		"target":    targetState,
		"timeoutMs": automationWaitTimeout,
		"pollMs":    automationWaitPollMs,
	}); err != nil {
		return err
	}

	deadline := time.Now().Add(time.Duration(automationWaitTimeout) * time.Millisecond)
	ticker := time.NewTicker(time.Duration(automationWaitPollMs) * time.Millisecond)
	defer ticker.Stop()
	lastProgressKey := "\x00"

	for {
		status, err := wshclient.BlockJobStatusCommand(RpcClient, oref.OID, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return err
		}
		progressKey := ""
		if status != nil {
			progressKey = fmt.Sprintf("%s:%d:%s:%s", status.Status, status.VersionTs, status.JobId, status.StartupError)
		}
		if progressKey != lastProgressKey {
			progressData := map[string]any{
				"mode":    "job",
				"blockId": oref.OID,
				"target":  targetState,
				"status":  status,
			}
			if err := emitAutomationEnvelope("wait_progress", progressData); err != nil {
				return err
			}
			lastProgressKey = progressKey
		}
		if status != nil && strings.EqualFold(status.Status, targetState) {
			return emitAutomationEnvelope("wait_satisfied", status)
		}
		if time.Now().After(deadline) {
			_ = emitAutomationEnvelope("wait_timeout", map[string]any{
				"mode":      "job",
				"blockId":   oref.OID,
				"target":    targetState,
				"timeoutMs": automationWaitTimeout,
				"last":      status,
			})
			return fmt.Errorf("timed out waiting for block %q to reach job state %q", oref.OID, targetState)
		}
		<-ticker.C
	}
}

func automationScreenshotRun(cmd *cobra.Command, args []string) error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	blockSnapshot, err := getAutomationBlockSnapshot(oref.OID)
	if err != nil {
		return err
	}
	if blockSnapshot == nil || blockSnapshot.TabId == "" {
		return fmt.Errorf("could not resolve tab for block %q", oref.OID)
	}
	dataURL, err := wshclient.CaptureBlockScreenshotCommand(RpcClient, wshrpc.CommandCaptureBlockScreenshotData{
		BlockId: oref.OID,
	}, &wshrpc.RpcOpts{
		Route:   wshutil.MakeTabRouteId(blockSnapshot.TabId),
		Timeout: 5000,
	})
	if err != nil {
		return err
	}
	mimeType, rawBytes, err := decodeScreenshotDataURL(dataURL)
	if err != nil {
		return err
	}

	artifact := automationArtifactPayload{
		ArtifactType: "screenshot",
		BlockId:      oref.OID,
		MimeType:     mimeType,
		Bytes:        len(rawBytes),
	}
	if automationShotOutput != "" {
		outputPath, err := resolvePathFromInitialCwd(automationShotOutput)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("creating screenshot directory: %w", err)
		}
		if err := os.WriteFile(outputPath, rawBytes, 0600); err != nil {
			return fmt.Errorf("writing screenshot: %w", err)
		}
		artifact.Path = outputPath
	} else {
		artifact.DataURL = dataURL
	}
	return emitAutomationEnvelope("artifact", artifact)
}

func automationActionFocusRun(cmd *cobra.Command, args []string) error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	tabId := getTabIdFromEnv()
	if tabId == "" {
		return fmt.Errorf("no WAVETERM_TABID env var set")
	}
	return emitAutomationActionLifecycle("focus", oref.OID, nil, func() error {
		if err := wshclient.SetBlockFocusCommand(RpcClient, oref.OID, &wshrpc.RpcOpts{
			Route:   fmt.Sprintf("tab:%s", tabId),
			Timeout: 2000,
		}); err != nil {
			return fmt.Errorf("focusing block: %w", err)
		}
		return nil
	})
}

func automationActionTypeRun(cmd *cobra.Command, args []string) error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	payload64 := base64.StdEncoding.EncodeToString([]byte(args[0]))
	return emitAutomationActionLifecycle("type", oref.OID, map[string]any{
		"bytes": len(args[0]),
	}, func() error {
		if err := wshclient.ControllerInputCommand(RpcClient, wshrpc.CommandBlockInputData{
			BlockId:     oref.OID,
			InputData64: payload64,
		}, &wshrpc.RpcOpts{Timeout: 2000}); err != nil {
			return fmt.Errorf("sending input: %w", err)
		}
		return nil
	})
}

func automationActionSignalRun(cmd *cobra.Command, args []string) error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	sigName := strings.TrimSpace(args[0])
	if sigName == "" {
		return fmt.Errorf("signal name cannot be empty")
	}
	return emitAutomationActionLifecycle("signal", oref.OID, map[string]any{
		"signal": sigName,
	}, func() error {
		if err := wshclient.ControllerInputCommand(RpcClient, wshrpc.CommandBlockInputData{
			BlockId: oref.OID,
			SigName: sigName,
		}, &wshrpc.RpcOpts{Timeout: 2000}); err != nil {
			return fmt.Errorf("sending signal: %w", err)
		}
		return nil
	})
}

func automationActionResizeRun(cmd *cobra.Command, args []string) error {
	oref, err := resolveBlockArg()
	if err != nil {
		return err
	}
	rows, err := strconv.Atoi(args[0])
	if err != nil || rows <= 0 {
		return fmt.Errorf("invalid rows value %q", args[0])
	}
	cols, err := strconv.Atoi(args[1])
	if err != nil || cols <= 0 {
		return fmt.Errorf("invalid cols value %q", args[1])
	}
	return emitAutomationActionLifecycle("resize", oref.OID, map[string]any{
		"rows": rows,
		"cols": cols,
	}, func() error {
		if err := wshclient.ControllerInputCommand(RpcClient, wshrpc.CommandBlockInputData{
			BlockId:  oref.OID,
			TermSize: &waveobj.TermSize{Rows: rows, Cols: cols},
		}, &wshrpc.RpcOpts{Timeout: 2000}); err != nil {
			return fmt.Errorf("sending resize: %w", err)
		}
		return nil
	})
}

func automationAppDismissWelcomeRun(cmd *cobra.Command, args []string) error {
	info, err := wshclient.WaveInfoCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return err
	}
	if info == nil || info.ClientId == "" {
		return fmt.Errorf("waveinfo returned no client id")
	}
	tabId := getTabIdFromEnv()
	extra := map[string]any{
		"clientId":          info.ClientId,
		"tabId":             tabId,
		"onboardingVersion": currentOnboardingVersion,
	}
	return emitAutomationActionLifecycle("dismiss_welcome", "", extra, func() error {
		if err := wshclient.AgreeTosCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000}); err != nil {
			return fmt.Errorf("agreeing to tos: %w", err)
		}
		if err := wshclient.SetMetaCommand(RpcClient, wshrpc.CommandSetMetaData{
			ORef: waveobj.MakeORef(waveobj.OType_Client, info.ClientId),
			Meta: waveobj.MetaMapType{
				waveobj.MetaKey_OnboardingLastVersion: currentOnboardingVersion,
			},
		}, &wshrpc.RpcOpts{Timeout: 2000}); err != nil {
			return fmt.Errorf("updating onboarding version: %w", err)
		}
		if tabId != "" {
			if err := wshclient.DismissOnboardingCommand(RpcClient, &wshrpc.RpcOpts{
				Route:   wshutil.MakeTabRouteId(tabId),
				Timeout: 2000,
			}); err != nil {
				return fmt.Errorf("dismissing onboarding ui: %w", err)
			}
		}
		return nil
	})
}

func automationAppStateRun(cmd *cobra.Command, args []string) error {
	state, err := buildAutomationAppState()
	if err != nil {
		return err
	}
	return writeJSONStdout(state)
}

func automationAppScreenshotRun(cmd *cobra.Command, args []string) error {
	return automationWindowScreenshotRun(cmd, nil)
}

func automationAppQuitRun(cmd *cobra.Command, args []string) error {
	return emitAutomationActionLifecycle("quit_app", "", map[string]any{
		"force": automationAppQuitForce,
	}, func() error {
		return wshclient.QuitAppCommand(RpcClient, automationAppQuitForce, &wshrpc.RpcOpts{
			Route:   wshutil.ElectronRoute,
			Timeout: 2000,
		})
	})
}

func automationWorkspaceListRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	currentWorkspaceId, _ := resolveAutomationWorkspaceId("", infos)
	list := make([]*automationWorkspaceInfo, 0, len(infos))
	for _, info := range infos {
		var activeTab *waveobj.Tab
		if info.WorkspaceData != nil && info.WorkspaceData.ActiveTabId != "" {
			activeTab, _ = wshclient.GetTabCommand(RpcClient, info.WorkspaceData.ActiveTabId, &wshrpc.RpcOpts{Timeout: 2000})
		}
		list = append(list, &automationWorkspaceInfo{
			WindowId:  info.WindowId,
			Workspace: info.WorkspaceData,
			ActiveTab: activeTab,
			IsCurrent: info.WorkspaceData != nil && info.WorkspaceData.OID == currentWorkspaceId,
		})
	}
	return writeJSONStdout(list)
}

func automationWorkspaceCreateRun(cmd *cobra.Command, args []string) error {
	name := firstArg(args)
	var workspaceId string
	err := emitAutomationActionLifecycle("create_workspace", "", map[string]any{
		"name":          name,
		"icon":          automationWorkspaceIcon,
		"color":         automationWorkspaceColor,
		"applyDefaults": automationWorkspaceApplyDefaults,
	}, func() error {
		var err error
		workspaceId, err = wshclient.CreateWorkspaceCommand(RpcClient, wshrpc.CommandCreateWorkspaceData{
			Name:          name,
			Icon:          automationWorkspaceIcon,
			Color:         automationWorkspaceColor,
			ApplyDefaults: automationWorkspaceApplyDefaults,
		}, &wshrpc.RpcOpts{Timeout: 3000})
		return err
	})
	if err != nil {
		return err
	}
	return writeJSONStdout(map[string]any{
		"workspaceId":   workspaceId,
		"name":          name,
		"icon":          automationWorkspaceIcon,
		"color":         automationWorkspaceColor,
		"applyDefaults": automationWorkspaceApplyDefaults,
	})
}

func automationWorkspaceStateRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	workspaceId, err := resolveAutomationWorkspaceId(firstAutomationValue(automationWorkspaceId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	state, err := buildAutomationWorkspaceState(workspaceId, infos)
	if err != nil {
		return err
	}
	return writeJSONStdout(state)
}

func automationWorkspaceSwitchRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	windowId, err := resolveAutomationWindowId(automationWindowId, infos)
	if err != nil {
		return err
	}
	workspaceId := args[0]
	return emitAutomationActionLifecycle("switch_workspace", "", map[string]any{
		"windowId":    windowId,
		"workspaceId": workspaceId,
	}, func() error {
		_, err := wshclient.SwitchWorkspaceCommand(RpcClient, wshrpc.CommandSwitchWorkspaceData{
			WindowId:    windowId,
			WorkspaceId: workspaceId,
		}, &wshrpc.RpcOpts{Timeout: 3000})
		return err
	})
}

func automationTabStateRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	tabId, err := resolveAutomationTabId(firstAutomationValue(automationTabId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	state, err := buildAutomationTabState(tabId, infos)
	if err != nil {
		return err
	}
	return writeJSONStdout(state)
}

func automationTabCreateRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	workspaceId, err := resolveAutomationWorkspaceId(automationWorkspaceId, infos)
	if err != nil {
		return err
	}
	tabName := firstArg(args)
	var createdTabId string
	err = emitAutomationActionLifecycle("create_tab", "", map[string]any{
		"workspaceId": workspaceId,
		"tabName":     tabName,
		"activate":    automationTabActivate,
	}, func() error {
		createdTabId, err = wshclient.CreateTabCommand(RpcClient, wshrpc.CommandCreateTabData{
			WorkspaceId: workspaceId,
			TabName:     tabName,
			ActivateTab: automationTabActivate,
		}, &wshrpc.RpcOpts{Timeout: 3000})
		return err
	})
	if err != nil {
		return err
	}
	return writeJSONStdout(map[string]any{
		"tabId":       createdTabId,
		"workspaceId": workspaceId,
		"activate":    automationTabActivate,
	})
}

func automationTabSwitchRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	tabId, err := resolveAutomationTabId(args[0], infos)
	if err != nil {
		return err
	}
	workspaceId, err := resolveAutomationWorkspaceIdForTab(tabId, automationWorkspaceId, infos)
	if err != nil {
		return err
	}
	return emitAutomationActionLifecycle("switch_tab", "", map[string]any{
		"workspaceId": workspaceId,
		"tabId":       tabId,
	}, func() error {
		return wshclient.SetActiveTabCommand(RpcClient, wshrpc.CommandSetActiveTabData{
			WorkspaceId: workspaceId,
			TabId:       tabId,
		}, &wshrpc.RpcOpts{Timeout: 3000})
	})
}

func automationTabCloseRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	tabId, err := resolveAutomationTabId(firstAutomationValue(automationTabId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	workspaceId, err := resolveAutomationWorkspaceIdForTab(tabId, automationWorkspaceId, infos)
	if err != nil {
		return err
	}
	var result *wshrpc.CommandCloseTabRtnData
	err = emitAutomationActionLifecycle("close_tab", "", map[string]any{
		"workspaceId": workspaceId,
		"tabId":       tabId,
	}, func() error {
		result, err = wshclient.CloseTabCommand(RpcClient, wshrpc.CommandCloseTabData{
			WorkspaceId: workspaceId,
			TabId:       tabId,
		}, &wshrpc.RpcOpts{Timeout: 3000})
		return err
	})
	if err != nil {
		return err
	}
	return writeJSONStdout(result)
}

func automationWindowListRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	currentWindowId, _ := resolveAutomationWindowId("", infos)
	windowIds := listAutomationWindowIds(infos)
	states := make([]*automationWindowState, 0, len(windowIds))
	for _, windowId := range windowIds {
		state, err := buildAutomationWindowState(windowId, infos)
		if err != nil {
			return err
		}
		state.IsCurrent = state.Window != nil && state.Window.OID == currentWindowId
		states = append(states, state)
	}
	return writeJSONStdout(states)
}

func automationWindowStateRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	windowId, err := resolveAutomationWindowId(firstAutomationValue(automationWindowId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	state, err := buildAutomationWindowState(windowId, infos)
	if err != nil {
		return err
	}
	return writeJSONStdout(state)
}

func automationWindowCreateRun(cmd *cobra.Command, args []string) error {
	workspaceId := firstArg(args)
	var windowId string
	err := emitAutomationActionLifecycle("create_window", "", map[string]any{
		"workspaceId": workspaceId,
	}, func() error {
		var err error
		windowId, err = wshclient.OpenNewWindowCommand(RpcClient, workspaceId, &wshrpc.RpcOpts{
			Route:   wshutil.ElectronRoute,
			Timeout: 5000,
		})
		return err
	})
	if err != nil {
		return err
	}
	state, err := buildAutomationWindowState(windowId, nil)
	if err != nil {
		return writeJSONStdout(map[string]any{
			"windowId":    windowId,
			"workspaceId": workspaceId,
		})
	}
	return writeJSONStdout(state)
}

func automationWindowFocusRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	windowId, err := resolveAutomationWindowId(firstAutomationValue(automationWindowId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	return emitAutomationActionLifecycle("focus_window", "", map[string]any{
		"windowId": windowId,
	}, func() error {
		return wshclient.FocusWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{
			Route:   wshutil.ElectronRoute,
			Timeout: 3000,
		})
	})
}

func automationWindowCloseRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	windowId, err := resolveAutomationWindowId(firstAutomationValue(automationWindowId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	return emitAutomationActionLifecycle("close_window", "", map[string]any{
		"windowId": windowId,
	}, func() error {
		return wshclient.CloseWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{Timeout: 3000})
	})
}

func automationWindowScreenshotRun(cmd *cobra.Command, args []string) error {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return err
	}
	windowId, err := resolveAutomationWindowId(firstAutomationValue(automationWindowId, firstArg(args)), infos)
	if err != nil {
		return err
	}
	dataURL, err := wshclient.CaptureWindowScreenshotCommand(RpcClient, windowId, &wshrpc.RpcOpts{
		Route:   wshutil.ElectronRoute,
		Timeout: 5000,
	})
	if err != nil {
		return err
	}
	mimeType, rawBytes, err := decodeScreenshotDataURL(dataURL)
	if err != nil {
		return err
	}
	artifact := automationArtifactPayload{
		ArtifactType: "window_screenshot",
		WindowId:     windowId,
		MimeType:     mimeType,
		Bytes:        len(rawBytes),
	}
	if automationShotOutput != "" {
		outputPath, err := resolvePathFromInitialCwd(automationShotOutput)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("creating screenshot directory: %w", err)
		}
		if err := os.WriteFile(outputPath, rawBytes, 0600); err != nil {
			return fmt.Errorf("writing screenshot: %w", err)
		}
		artifact.Path = outputPath
	} else {
		artifact.DataURL = dataURL
	}
	return emitAutomationEnvelope("artifact", artifact)
}

func getAutomationBlockSnapshot(blockId string) (*automationBlockSnapshot, error) {
	info, err := wshclient.BlockInfoCommand(RpcClient, blockId, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, fmt.Errorf("block info not found: %s", blockId)
	}
	rtInfo, err := wshclient.GetRTInfoCommand(RpcClient, wshrpc.CommandGetRTInfoData{
		ORef: waveobj.MakeORef(waveobj.OType_Block, blockId),
	}, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}
	jobStatus, err := wshclient.BlockJobStatusCommand(RpcClient, blockId, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}

	block := info.Block
	if block == nil {
		return nil, fmt.Errorf("block not found: %s", blockId)
	}
	return &automationBlockSnapshot{
		BlockId:     blockId,
		TabId:       info.TabId,
		WorkspaceId: info.WorkspaceId,
		Meta:        block.Meta,
		RuntimeOpts: block.RuntimeOpts,
		SubBlockIds: block.SubBlockIds,
		Files:       info.Files,
		RTInfo:      rtInfo,
		JobStatus:   jobStatus,
		ConnStatus:  findConnStatus(block.Meta.GetString(waveobj.MetaKey_Connection, "")),
	}, nil
}

func findConnStatus(connName string) *wshrpc.ConnStatus {
	if strings.TrimSpace(connName) == "" {
		return nil
	}
	statuses, err := wshclient.ConnStatusCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil
	}
	for i := range statuses {
		if statuses[i].Connection == connName {
			status := statuses[i]
			return &status
		}
	}
	return nil
}

func decodeScreenshotDataURL(dataURL string) (string, []byte, error) {
	if !strings.HasPrefix(dataURL, "data:") {
		return "", nil, fmt.Errorf("invalid screenshot payload: expected data URL")
	}
	commaIdx := strings.IndexByte(dataURL, ',')
	if commaIdx < 0 {
		return "", nil, fmt.Errorf("invalid screenshot payload: malformed data URL")
	}
	header := dataURL[5:commaIdx]
	dataPart := dataURL[commaIdx+1:]
	if !strings.HasSuffix(header, ";base64") {
		return "", nil, fmt.Errorf("invalid screenshot payload: expected base64 data URL")
	}
	mimeType := strings.TrimSuffix(header, ";base64")
	decoded, err := base64.StdEncoding.DecodeString(dataPart)
	if err != nil {
		return "", nil, fmt.Errorf("decoding screenshot data: %w", err)
	}
	return mimeType, decoded, nil
}

func buildAutomationAppState() (*automationAppState, error) {
	infos, err := getAutomationWorkspaceInfos()
	if err != nil {
		return nil, err
	}
	currentWindowId, _ := resolveAutomationWindowId("", infos)
	currentWorkspaceId, _ := resolveAutomationWorkspaceId("", infos)
	currentTabId, _ := resolveAutomationTabId("", infos)
	windowIds := listAutomationWindowIds(infos)
	windowStates := make([]*automationWindowState, 0, len(windowIds))
	for _, windowId := range windowIds {
		state, err := buildAutomationWindowState(windowId, infos)
		if err != nil {
			return nil, err
		}
		state.IsCurrent = state.Window != nil && state.Window.OID == currentWindowId
		windowStates = append(windowStates, state)
	}
	workspaces := make([]*automationWorkspaceInfo, 0, len(infos))
	for _, info := range infos {
		var activeTab *waveobj.Tab
		if info.WorkspaceData != nil && info.WorkspaceData.ActiveTabId != "" {
			activeTab, _ = wshclient.GetTabCommand(RpcClient, info.WorkspaceData.ActiveTabId, &wshrpc.RpcOpts{Timeout: 2000})
		}
		workspaces = append(workspaces, &automationWorkspaceInfo{
			WindowId:  info.WindowId,
			Workspace: info.WorkspaceData,
			ActiveTab: activeTab,
			IsCurrent: info.WorkspaceData != nil && info.WorkspaceData.OID == currentWorkspaceId,
		})
	}
	return &automationAppState{
		Millis:             time.Now().UnixMilli(),
		CurrentWindowId:    currentWindowId,
		CurrentWorkspaceId: currentWorkspaceId,
		CurrentTabId:       currentTabId,
		Windows:            windowStates,
		Workspaces:         workspaces,
	}, nil
}

func buildAutomationWorkspaceState(workspaceId string, infos []wshrpc.WorkspaceInfoData) (*automationWorkspaceState, error) {
	info := findAutomationWorkspaceById(infos, workspaceId)
	if info == nil {
		fallbackWorkspace, err := wshclient.GetWorkspaceCommand(RpcClient, workspaceId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, fmt.Errorf("workspace not found: %s", workspaceId)
		}
		info = &wshrpc.WorkspaceInfoData{WorkspaceData: fallbackWorkspace}
	}
	if info.WorkspaceData == nil {
		return nil, fmt.Errorf("workspace not found: %s", workspaceId)
	}
	var window *waveobj.Window
	var err error
	if info.WindowId != "" {
		window, err = wshclient.GetWindowCommand(RpcClient, info.WindowId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
	}
	tabs := make([]*waveobj.Tab, 0, len(info.WorkspaceData.TabIds))
	for _, tabId := range info.WorkspaceData.TabIds {
		tab, err := wshclient.GetTabCommand(RpcClient, tabId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
		if tab != nil {
			tabs = append(tabs, tab)
		}
	}
	var activeTab *waveobj.Tab
	if info.WorkspaceData.ActiveTabId != "" {
		activeTab, err = wshclient.GetTabCommand(RpcClient, info.WorkspaceData.ActiveTabId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
	}
	currentWorkspaceId, _ := resolveAutomationWorkspaceId("", infos)
	return &automationWorkspaceState{
		Millis:    time.Now().UnixMilli(),
		Window:    window,
		Workspace: info.WorkspaceData,
		ActiveTab: activeTab,
		Tabs:      tabs,
		IsCurrent: workspaceId == currentWorkspaceId,
	}, nil
}

func buildAutomationTabState(tabId string, infos []wshrpc.WorkspaceInfoData) (*automationTabState, error) {
	tab, err := wshclient.GetTabCommand(RpcClient, tabId, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}
	if tab == nil {
		return nil, fmt.Errorf("tab not found: %s", tabId)
	}
	info := findAutomationWorkspaceByTabId(infos, tabId)
	if info == nil || info.WorkspaceData == nil {
		return nil, fmt.Errorf("workspace not found for tab: %s", tabId)
	}
	var window *waveobj.Window
	if info.WindowId != "" {
		window, err = wshclient.GetWindowCommand(RpcClient, info.WindowId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
	}
	blocks, err := wshclient.BlocksListCommand(RpcClient, wshrpc.BlocksListRequest{
		WorkspaceId: info.WorkspaceData.OID,
	}, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}
	filteredBlocks := make([]wshrpc.BlocksListEntry, 0, len(blocks))
	for _, block := range blocks {
		if block.TabId == tabId {
			filteredBlocks = append(filteredBlocks, block)
		}
	}
	focused, _ := wshclient.GetFocusedBlockDataCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	currentTabId, _ := resolveAutomationTabId("", infos)
	return &automationTabState{
		Millis:    time.Now().UnixMilli(),
		Window:    window,
		Workspace: info.WorkspaceData,
		Tab:       tab,
		Blocks:    filteredBlocks,
		Focused:   focused,
		IsCurrent: tabId == currentTabId,
	}, nil
}

func buildAutomationWindowState(windowId string, infos []wshrpc.WorkspaceInfoData) (*automationWindowState, error) {
	window, err := wshclient.GetWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return nil, err
	}
	if window == nil {
		return nil, fmt.Errorf("window not found: %s", windowId)
	}
	info := findAutomationWorkspaceById(infos, window.WorkspaceId)
	var workspace *waveobj.Workspace
	if info != nil {
		workspace = info.WorkspaceData
	}
	if workspace == nil && window.WorkspaceId != "" {
		workspace, err = wshclient.GetWorkspaceCommand(RpcClient, window.WorkspaceId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
	}
	var activeTab *waveobj.Tab
	if workspace != nil && workspace.ActiveTabId != "" {
		activeTab, err = wshclient.GetTabCommand(RpcClient, workspace.ActiveTabId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil {
			return nil, err
		}
	}
	return &automationWindowState{
		Millis:      time.Now().UnixMilli(),
		Window:      window,
		Workspace:   workspace,
		ActiveTab:   activeTab,
		IsFocusable: true,
	}, nil
}

func getAutomationWorkspaceInfos() ([]wshrpc.WorkspaceInfoData, error) {
	client, err := wshclient.GetClientCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return wshclient.WorkspaceListCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	}
	rtn := make([]wshrpc.WorkspaceInfoData, 0)
	seenWorkspaceIds := make(map[string]bool)
	for _, windowId := range client.WindowIds {
		window, err := wshclient.GetWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil || window == nil {
			continue
		}
		workspace, err := wshclient.GetWorkspaceCommand(RpcClient, window.WorkspaceId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil || workspace == nil {
			continue
		}
		seenWorkspaceIds[workspace.OID] = true
		rtn = append(rtn, wshrpc.WorkspaceInfoData{
			WindowId:      window.OID,
			WorkspaceData: workspace,
		})
	}
	workspaceInfos, err := wshclient.WorkspaceListCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil {
		return rtn, nil
	}
	for _, info := range workspaceInfos {
		if info.WorkspaceData == nil || seenWorkspaceIds[info.WorkspaceData.OID] {
			continue
		}
		seenWorkspaceIds[info.WorkspaceData.OID] = true
		rtn = append(rtn, info)
	}
	return rtn, nil
}

func findAutomationWorkspaceById(infos []wshrpc.WorkspaceInfoData, workspaceId string) *wshrpc.WorkspaceInfoData {
	for i := range infos {
		if infos[i].WorkspaceData != nil && infos[i].WorkspaceData.OID == workspaceId {
			return &infos[i]
		}
	}
	return nil
}

func findAutomationWorkspaceByWindowId(infos []wshrpc.WorkspaceInfoData, windowId string) *wshrpc.WorkspaceInfoData {
	for i := range infos {
		if infos[i].WindowId == windowId {
			return &infos[i]
		}
	}
	return nil
}

func findAutomationWorkspaceByTabId(infos []wshrpc.WorkspaceInfoData, tabId string) *wshrpc.WorkspaceInfoData {
	for i := range infos {
		workspace := infos[i].WorkspaceData
		if workspace == nil {
			continue
		}
		for _, workspaceTabId := range workspace.TabIds {
			if workspaceTabId == tabId {
				return &infos[i]
			}
		}
	}
	return nil
}

func listAutomationWindowIds(infos []wshrpc.WorkspaceInfoData) []string {
	windowIds := make([]string, 0, len(infos))
	seen := make(map[string]bool)
	for _, info := range infos {
		if info.WindowId == "" || seen[info.WindowId] {
			continue
		}
		seen[info.WindowId] = true
		windowIds = append(windowIds, info.WindowId)
	}
	return windowIds
}

func resolveAutomationWindowId(explicitWindowId string, infos []wshrpc.WorkspaceInfoData) (string, error) {
	if explicitWindowId != "" {
		return explicitWindowId, nil
	}
	if workspaceId := strings.TrimSpace(os.Getenv("WAVETERM_WORKSPACEID")); workspaceId != "" {
		if info := findAutomationWorkspaceById(infos, workspaceId); info != nil && info.WindowId != "" {
			return info.WindowId, nil
		}
	}
	if tabId := strings.TrimSpace(os.Getenv("WAVETERM_TABID")); tabId != "" {
		if info := findAutomationWorkspaceByTabId(infos, tabId); info != nil && info.WindowId != "" {
			return info.WindowId, nil
		}
	}
	client, err := wshclient.GetClientCommand(RpcClient, &wshrpc.RpcOpts{Timeout: 2000})
	if err == nil {
		for _, windowId := range client.WindowIds {
			if strings.TrimSpace(windowId) != "" {
				return windowId, nil
			}
		}
	}
	windowIds := listAutomationWindowIds(infos)
	if len(windowIds) == 0 {
		return "", fmt.Errorf("no windows available")
	}
	if len(windowIds) == 1 {
		return windowIds[0], nil
	}
	bestWindowId := windowIds[0]
	bestFocusTs := int64(-1)
	for _, windowId := range windowIds {
		window, err := wshclient.GetWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{Timeout: 2000})
		if err != nil || window == nil {
			continue
		}
		if window.LastFocusTs > bestFocusTs {
			bestWindowId = windowId
			bestFocusTs = window.LastFocusTs
		}
	}
	return bestWindowId, nil
}

func resolveAutomationWorkspaceId(explicitWorkspaceId string, infos []wshrpc.WorkspaceInfoData) (string, error) {
	if explicitWorkspaceId != "" {
		return explicitWorkspaceId, nil
	}
	if workspaceId := strings.TrimSpace(os.Getenv("WAVETERM_WORKSPACEID")); workspaceId != "" {
		return workspaceId, nil
	}
	if tabId := strings.TrimSpace(os.Getenv("WAVETERM_TABID")); tabId != "" {
		if info := findAutomationWorkspaceByTabId(infos, tabId); info != nil && info.WorkspaceData != nil {
			return info.WorkspaceData.OID, nil
		}
	}
	windowId, err := resolveAutomationWindowId("", infos)
	if err == nil {
		if info := findAutomationWorkspaceByWindowId(infos, windowId); info != nil && info.WorkspaceData != nil {
			return info.WorkspaceData.OID, nil
		}
		window, getWindowErr := wshclient.GetWindowCommand(RpcClient, windowId, &wshrpc.RpcOpts{Timeout: 2000})
		if getWindowErr == nil && window != nil && window.WorkspaceId != "" {
			return window.WorkspaceId, nil
		}
	}
	if len(infos) == 1 && infos[0].WorkspaceData != nil {
		return infos[0].WorkspaceData.OID, nil
	}
	return "", fmt.Errorf("no workspace available")
}

func resolveAutomationWorkspaceIdForTab(tabId string, explicitWorkspaceId string, infos []wshrpc.WorkspaceInfoData) (string, error) {
	if explicitWorkspaceId != "" {
		return explicitWorkspaceId, nil
	}
	info := findAutomationWorkspaceByTabId(infos, tabId)
	if info == nil || info.WorkspaceData == nil {
		return "", fmt.Errorf("workspace not found for tab: %s", tabId)
	}
	return info.WorkspaceData.OID, nil
}

func resolveAutomationTabId(explicitTabId string, infos []wshrpc.WorkspaceInfoData) (string, error) {
	if explicitTabId != "" {
		return explicitTabId, nil
	}
	if tabId := strings.TrimSpace(os.Getenv("WAVETERM_TABID")); tabId != "" {
		return tabId, nil
	}
	workspaceId, err := resolveAutomationWorkspaceId("", infos)
	if err != nil {
		return "", err
	}
	info := findAutomationWorkspaceById(infos, workspaceId)
	if info != nil && info.WorkspaceData != nil && info.WorkspaceData.ActiveTabId != "" {
		return info.WorkspaceData.ActiveTabId, nil
	}
	workspace, err := wshclient.GetWorkspaceCommand(RpcClient, workspaceId, &wshrpc.RpcOpts{Timeout: 2000})
	if err != nil || workspace == nil || workspace.ActiveTabId == "" {
		return "", fmt.Errorf("no active tab available")
	}
	return workspace.ActiveTabId, nil
}

func firstAutomationValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
