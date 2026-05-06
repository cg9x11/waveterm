// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import type { BlockNodeModel } from "@/app/block/blocktypes";
import { replaceBlock } from "@/app/store/global";
import { makeORef } from "@/app/store/wos";
import { makeFeBlockRouteId } from "@/app/store/wshrouter";
import type { TabModel } from "@/app/store/tab-model";
import { TabRpcClient } from "@/app/store/wshrpcutil";
import { WaveEnv, WaveEnvSubset, useWaveEnv } from "@/app/waveenv/waveenv";
import { isBlank, isLocalConnName } from "@/util/util";
import clsx from "clsx";
import { atom, useAtomValue } from "jotai";
import * as React from "react";

const MaxScanDepth = 3;
const MaxScanDirs = 300;
const MaxListEntries = 250;
const MaxPromptProbeLines = 48;

type RootCandidate = {
    path: string;
    activeTab: boolean;
    hits: number;
    label: string;
};

type RepoEntry = {
    root: string;
    name: string;
    activeTab: boolean;
    score: number;
    hits: number;
    sources: string[];
};

type DiscoveryResult = {
    roots: RootCandidate[];
    repos: RepoEntry[];
    errors: string[];
    truncated: boolean;
    fallbackRoot: string;
};

type GitLauncherEnv = WaveEnvSubset<{
    platform: WaveEnv["platform"];
    atoms: {
        workspaceId: WaveEnv["atoms"]["workspaceId"];
        workspace: WaveEnv["atoms"]["workspace"];
    };
    rpc: {
        BlocksListCommand: WaveEnv["rpc"]["BlocksListCommand"];
        FileInfoCommand: WaveEnv["rpc"]["FileInfoCommand"];
        FileJoinCommand: WaveEnv["rpc"]["FileJoinCommand"];
        FileListCommand: WaveEnv["rpc"]["FileListCommand"];
        FindLazygitCommand: WaveEnv["rpc"]["FindLazygitCommand"];
        GetRTInfoCommand: WaveEnv["rpc"]["GetRTInfoCommand"];
        TermGetScrollbackLinesCommand: WaveEnv["rpc"]["TermGetScrollbackLinesCommand"];
    };
}>;

function emptyDiscovery(): DiscoveryResult {
    return {
        roots: [],
        repos: [],
        errors: [],
        truncated: false,
        fallbackRoot: "",
    };
}

function stripTrailingSeparators(path: string): string {
    if (isBlank(path)) {
        return "";
    }
    let value = path.trim();
    while (value.length > 1 && /[\\/]/.test(value[value.length - 1])) {
        if (value === "/") {
            break;
        }
        if (/^[a-zA-Z]:[\\/]?$/.test(value)) {
            break;
        }
        value = value.slice(0, -1);
    }
    return value;
}

function normalizePathKey(path: string, platform: NodeJS.Platform): string {
    const normalized = stripTrailingSeparators(path).replace(/[\\/]+/g, "/");
    if (platform === "win32") {
        return normalized.toLowerCase();
    }
    return normalized;
}

function pathsEqual(a: string, b: string, platform: NodeJS.Platform): boolean {
    return normalizePathKey(a, platform) === normalizePathKey(b, platform);
}

function getPathBaseName(path: string): string {
    const cleaned = stripTrailingSeparators(path);
    if (isBlank(cleaned)) {
        return "";
    }
    const segments = cleaned.split(/[\\/]/).filter(Boolean);
    if (segments.length === 0) {
        return cleaned;
    }
    return segments[segments.length - 1];
}

function shouldSkipScanDir(name: string): boolean {
    if (isBlank(name)) {
        return true;
    }
    const lowerName = name.toLowerCase();
    if (lowerName.startsWith(".")) {
        return true;
    }
    return (
        lowerName === "node_modules" ||
        lowerName === "dist" ||
        lowerName === "build" ||
        lowerName === "out" ||
        lowerName === "target" ||
        lowerName === "coverage" ||
        lowerName === ".git" ||
        lowerName === ".next" ||
        lowerName === ".turbo" ||
        lowerName === ".cache"
    );
}

function dedupeStrings(values: string[]): string[] {
    const seen = new Set<string>();
    const result: string[] = [];
    for (const value of values) {
        if (isBlank(value) || seen.has(value)) {
            continue;
        }
        seen.add(value);
        result.push(value);
    }
    return result;
}

function sanitizePromptLine(line: string): string {
    if (isBlank(line)) {
        return "";
    }
    return line
        .replace(/\u001b\[[0-9;?]*[ -/]*[@-~]/g, "")
        .replace(/\u001b\][^\u0007]*(?:\u0007|\u001b\\)/g, "")
        .trim();
}

function extractPromptPathCandidates(lines: string[], platform: NodeJS.Platform): string[] {
    const candidates: string[] = [];

    function addMatches(line: string, patterns: RegExp[]) {
        for (const pattern of patterns) {
            const match = line.match(pattern);
            const value = stripTrailingSeparators(match?.[1]?.trim() ?? "");
            if (!isBlank(value)) {
                candidates.push(value);
            }
        }
    }

    for (let index = lines.length - 1; index >= 0; index--) {
        const line = sanitizePromptLine(lines[index] ?? "");
        if (isBlank(line)) {
            continue;
        }

        if (platform === "win32") {
            addMatches(line, [
                /^PS\s+(.+?)>\s*$/,
                /^([a-zA-Z]:[\\/].*?)>\s*$/,
                /(?:^|\s)([a-zA-Z]:[\\/].+?)>\s*$/,
                /(?:^|\s)(~[\\/].+?)>\s*$/,
            ]);
        }

        addMatches(line, [
            /(?:^|[\s\]])(?:[^@\s]+@[^:\s]+:)?([~\/][^\r\n$#%>]*)[$#%>]\s*$/,
        ]);
    }

    return dedupeStrings(candidates);
}

function compareRoots(a: RootCandidate, b: RootCandidate): number {
    if (a.activeTab !== b.activeTab) {
        return a.activeTab ? -1 : 1;
    }
    if (a.hits !== b.hits) {
        return b.hits - a.hits;
    }
    return a.path.localeCompare(b.path);
}

function compareRepos(a: RepoEntry, b: RepoEntry): number {
    if (a.score !== b.score) {
        return b.score - a.score;
    }
    if (a.activeTab !== b.activeTab) {
        return a.activeTab ? -1 : 1;
    }
    if (a.hits !== b.hits) {
        return b.hits - a.hits;
    }
    return a.root.localeCompare(b.root);
}

function upsertRepo(
    repoMap: Map<string, RepoEntry>,
    repoRoot: string,
    platform: NodeJS.Platform,
    opts: { activeTab: boolean; hits: number; label: string; score: number }
) {
    const normalizedRoot = stripTrailingSeparators(repoRoot);
    if (isBlank(normalizedRoot)) {
        return;
    }
    const repoKey = normalizePathKey(normalizedRoot, platform);
    const currentRepo = repoMap.get(repoKey);
    if (currentRepo == null) {
        repoMap.set(repoKey, {
            root: normalizedRoot,
            name: getPathBaseName(normalizedRoot) || normalizedRoot,
            activeTab: opts.activeTab,
            hits: opts.hits,
            score: opts.score,
            sources: [opts.label],
        });
        return;
    }
    currentRepo.activeTab = currentRepo.activeTab || opts.activeTab;
    currentRepo.hits = Math.max(currentRepo.hits, opts.hits);
    currentRepo.score = Math.max(currentRepo.score, opts.score);
    currentRepo.sources = dedupeStrings([...currentRepo.sources, opts.label]);
}

async function getFileInfo(rpc: GitLauncherEnv["rpc"], path: string): Promise<FileInfo | null> {
    try {
        return await rpc.FileInfoCommand(TabRpcClient, { info: { path } }, { timeout: 4000 });
    } catch {
        return null;
    }
}

async function joinPath(rpc: GitLauncherEnv["rpc"], basePath: string, childName: string): Promise<string | null> {
    try {
        const joined = await rpc.FileJoinCommand(TabRpcClient, [basePath, childName], { timeout: 4000 });
        return joined?.path ?? null;
    } catch {
        return null;
    }
}

async function hasGitMarker(rpc: GitLauncherEnv["rpc"], dirPath: string): Promise<boolean> {
    const gitPath = await joinPath(rpc, dirPath, ".git");
    if (isBlank(gitPath)) {
        return false;
    }
    const gitInfo = await getFileInfo(rpc, gitPath);
    return gitInfo != null && !gitInfo.notfound && isBlank(gitInfo.staterror);
}

async function resolveDirectoryCandidate(rpc: GitLauncherEnv["rpc"], path: string): Promise<string> {
    if (isBlank(path)) {
        return "";
    }
    const info = await getFileInfo(rpc, path);
    if (info == null || info.notfound || !isBlank(info.staterror)) {
        return "";
    }
    return stripTrailingSeparators(info.isdir ? info.path : info.dir);
}

async function getRuntimeCurrentDirectory(rpc: GitLauncherEnv["rpc"], blockId: string): Promise<string> {
    try {
        const rtInfo = await rpc.GetRTInfoCommand(
            TabRpcClient,
            { oref: makeORef("block", blockId) },
            { timeout: 2500 }
        );
        return typeof rtInfo?.["shell:curcwd"] === "string" ? rtInfo["shell:curcwd"] : "";
    } catch {
        return "";
    }
}

async function getPromptCurrentDirectory(
    rpc: GitLauncherEnv["rpc"],
    blockId: string,
    platform: NodeJS.Platform
): Promise<string> {
    try {
        const scrollback = await rpc.TermGetScrollbackLinesCommand(
            TabRpcClient,
            {
                linestart: 0,
                lineend: MaxPromptProbeLines,
                lastcommand: false,
            },
            { route: makeFeBlockRouteId(blockId), timeout: 2500 }
        );
        const promptCandidates = extractPromptPathCandidates(scrollback?.lines ?? [], platform);
        for (const promptCandidate of promptCandidates) {
            const resolvedDir = await resolveDirectoryCandidate(rpc, promptCandidate);
            if (!isBlank(resolvedDir)) {
                return resolvedDir;
            }
        }
    } catch {
        // Ignore per-block prompt probing failures and continue with other fallbacks.
    }
    return "";
}

async function resolveWorkspaceRootPath(
    rpc: GitLauncherEnv["rpc"],
    block: BlocksListEntry,
    platform: NodeJS.Platform
): Promise<string> {
    if (!isLocalConnName(block.meta?.connection)) {
        return "";
    }

    const explicitCwd = typeof block.meta?.["cmd:cwd"] === "string" ? block.meta["cmd:cwd"] : "";
    const explicitDir = await resolveDirectoryCandidate(rpc, explicitCwd);
    if (!isBlank(explicitDir)) {
        return explicitDir;
    }

    if (block.meta?.view === "preview") {
        const previewPath = typeof block.meta?.file === "string" ? block.meta.file : "";
        const previewDir = await resolveDirectoryCandidate(rpc, previewPath);
        if (!isBlank(previewDir)) {
            return previewDir;
        }
    }

    if (block.meta?.view !== "term") {
        return "";
    }

    const runtimeCwd = await getRuntimeCurrentDirectory(rpc, block.blockid);
    const runtimeDir = await resolveDirectoryCandidate(rpc, runtimeCwd);
    if (!isBlank(runtimeDir)) {
        return runtimeDir;
    }

    return getPromptCurrentDirectory(rpc, block.blockid, platform);
}

function getRootLabel(block: BlocksListEntry, activeTabId: string): string {
    const isActiveTab = block.tabid === activeTabId;
    if (block.meta?.view === "term") {
        return isActiveTab ? "active terminal" : "open terminal";
    }
    if (block.meta?.view === "preview") {
        return isActiveTab ? "active file" : "open file";
    }
    return isActiveTab ? "active tab" : "open widget";
}

async function findEnclosingRepoRoot(
    rpc: GitLauncherEnv["rpc"],
    startPath: string,
    platform: NodeJS.Platform
): Promise<string | null> {
    const startInfo = await getFileInfo(rpc, startPath);
    let currentPath = startInfo?.isdir ? startInfo.path : startInfo?.dir ?? stripTrailingSeparators(startPath);
    const seen = new Set<string>();

    while (!isBlank(currentPath)) {
        const currentKey = normalizePathKey(currentPath, platform);
        if (seen.has(currentKey)) {
            break;
        }
        seen.add(currentKey);

        if (await hasGitMarker(rpc, currentPath)) {
            return stripTrailingSeparators(currentPath);
        }

        const currentInfo = await getFileInfo(rpc, currentPath);
        const parentPath = currentInfo?.dir;
        if (isBlank(parentPath) || normalizePathKey(parentPath, platform) === currentKey) {
            break;
        }
        currentPath = parentPath;
    }

    return null;
}

async function collectWorkspaceRoots(
    rpc: GitLauncherEnv["rpc"],
    workspaceId: string,
    activeTabId: string,
    platform: NodeJS.Platform
): Promise<RootCandidate[]> {
    const blocks = await rpc.BlocksListCommand(TabRpcClient, { workspaceid: workspaceId }, { timeout: 8000 });
    const rootMap = new Map<string, RootCandidate>();

    const resolvedRoots = await Promise.all(
        (blocks ?? []).map(async (block) => {
            const rootPath = await resolveWorkspaceRootPath(rpc, block, platform);
            if (isBlank(rootPath)) {
                return null;
            }
            return {
                path: rootPath,
                activeTab: block.tabid === activeTabId,
                label: getRootLabel(block, activeTabId),
            };
        })
    );

    for (const resolvedRoot of resolvedRoots) {
        if (resolvedRoot == null) {
            continue;
        }
        const normalizedCwd = stripTrailingSeparators(resolvedRoot.path);
        const rootKey = normalizePathKey(normalizedCwd, platform);
        const existing = rootMap.get(rootKey);
        if (existing == null) {
            rootMap.set(rootKey, {
                path: normalizedCwd,
                activeTab: resolvedRoot.activeTab,
                hits: 1,
                label: resolvedRoot.label,
            });
            continue;
        }
        existing.activeTab = existing.activeTab || resolvedRoot.activeTab;
        existing.hits += 1;
        if (resolvedRoot.label.includes("terminal") && !existing.label.includes("terminal")) {
            existing.label = resolvedRoot.activeTab ? "active terminal" : "open terminal";
        }
        if (existing.activeTab) {
            existing.label = existing.label.includes("terminal") ? "active terminal" : "active file";
        }
    }

    return Array.from(rootMap.values()).sort(compareRoots);
}

async function scanRepoRootsBelow(
    rpc: GitLauncherEnv["rpc"],
    root: RootCandidate,
    platform: NodeJS.Platform
): Promise<{ repos: RepoEntry[]; errors: string[]; truncated: boolean }> {
    const repoMap = new Map<string, RepoEntry>();
    const errors: string[] = [];
    const visited = new Set<string>();
    const queue: Array<{ path: string; depth: number }> = [{ path: root.path, depth: 0 }];
    let truncated = false;

    while (queue.length > 0) {
        if (visited.size >= MaxScanDirs) {
            truncated = true;
            break;
        }

        const current = queue.shift();
        const currentPath = stripTrailingSeparators(current.path);
        const currentKey = normalizePathKey(currentPath, platform);
        if (visited.has(currentKey)) {
            continue;
        }
        visited.add(currentKey);

        if (await hasGitMarker(rpc, currentPath)) {
            upsertRepo(repoMap, currentPath, platform, {
                activeTab: root.activeTab,
                hits: root.hits,
                label: "workspace scan",
                score: root.activeTab ? 80 : 50,
            });
            continue;
        }

        if (current.depth >= MaxScanDepth) {
            continue;
        }

        try {
            const entries = await rpc.FileListCommand(
                TabRpcClient,
                { path: currentPath, opts: { limit: MaxListEntries } },
                { timeout: 8000 }
            );
            for (const entry of entries ?? []) {
                if (!entry?.isdir) {
                    continue;
                }
                const childName = entry.name ?? getPathBaseName(entry.path);
                if (shouldSkipScanDir(childName)) {
                    continue;
                }
                queue.push({ path: stripTrailingSeparators(entry.path), depth: current.depth + 1 });
            }
        } catch (error) {
            errors.push(`Could not read ${currentPath}: ${String(error)}`);
        }
    }

    return {
        repos: Array.from(repoMap.values()).sort(compareRepos),
        errors,
        truncated,
    };
}

async function discoverWorkspaceRepos(
    rpc: GitLauncherEnv["rpc"],
    workspaceId: string,
    activeTabId: string,
    platform: NodeJS.Platform
): Promise<DiscoveryResult> {
    if (isBlank(workspaceId)) {
        return emptyDiscovery();
    }

    const roots = await collectWorkspaceRoots(rpc, workspaceId, activeTabId, platform);
    const repoMap = new Map<string, RepoEntry>();
    const errors: string[] = [];
    const scanRoots: RootCandidate[] = [];
    let fallbackRoot = "";
    let truncated = false;

    for (const root of roots) {
        const repoRoot = await findEnclosingRepoRoot(rpc, root.path, platform);
        if (!isBlank(repoRoot)) {
            upsertRepo(repoMap, repoRoot, platform, {
                activeTab: root.activeTab,
                hits: root.hits,
                label: root.label,
                score: root.activeTab ? 100 : 70,
            });
            continue;
        }
        scanRoots.push(root);
        if (isBlank(fallbackRoot)) {
            fallbackRoot = root.path;
        }
    }

    if (isBlank(fallbackRoot) && roots.length > 0) {
        fallbackRoot = roots[0].path;
    }

    for (const root of scanRoots) {
        const scanResult = await scanRepoRootsBelow(rpc, root, platform);
        truncated = truncated || scanResult.truncated;
        errors.push(...scanResult.errors);
        for (const repo of scanResult.repos) {
            upsertRepo(repoMap, repo.root, platform, {
                activeTab: repo.activeTab,
                hits: repo.hits,
                label: "workspace scan",
                score: repo.score,
            });
        }
    }

    return {
        roots,
        repos: Array.from(repoMap.values()).sort(compareRepos),
        errors: dedupeStrings(errors),
        truncated,
        fallbackRoot,
    };
}

async function discoverReposFromRoot(
    rpc: GitLauncherEnv["rpc"],
    rootPath: string,
    platform: NodeJS.Platform
): Promise<DiscoveryResult> {
    const trimmedRoot = stripTrailingSeparators(rootPath);
    if (isBlank(trimmedRoot)) {
        return emptyDiscovery();
    }

    const root: RootCandidate = {
        path: trimmedRoot,
        activeTab: false,
        hits: 1,
        label: "manual root",
    };
    const repoMap = new Map<string, RepoEntry>();
    const errors: string[] = [];
    let truncated = false;

    const enclosingRepo = await findEnclosingRepoRoot(rpc, trimmedRoot, platform);
    if (!isBlank(enclosingRepo)) {
        upsertRepo(repoMap, enclosingRepo, platform, {
            activeTab: false,
            hits: 1,
            label: "manual root",
            score: 65,
        });
    } else {
        const scanResult = await scanRepoRootsBelow(rpc, root, platform);
        truncated = scanResult.truncated;
        errors.push(...scanResult.errors);
        for (const repo of scanResult.repos) {
            upsertRepo(repoMap, repo.root, platform, {
                activeTab: repo.activeTab,
                hits: repo.hits,
                label: "manual scan",
                score: Math.max(repo.score, 60),
            });
        }
    }

    return {
        roots: [root],
        repos: Array.from(repoMap.values()).sort(compareRepos),
        errors: dedupeStrings(errors),
        truncated,
        fallbackRoot: trimmedRoot,
    };
}

function mergeRepoLists(platform: NodeJS.Platform, ...repoLists: RepoEntry[][]): RepoEntry[] {
    const repoMap = new Map<string, RepoEntry>();
    for (const repoList of repoLists) {
        for (const repo of repoList ?? []) {
            upsertRepo(repoMap, repo.root, platform, {
                activeTab: repo.activeTab,
                hits: repo.hits,
                label: repo.sources[0] ?? "repo",
                score: repo.score,
            });
            const repoKey = normalizePathKey(repo.root, platform);
            const merged = repoMap.get(repoKey);
            merged.sources = dedupeStrings([...(merged?.sources ?? []), ...(repo.sources ?? [])]);
        }
    }
    return Array.from(repoMap.values()).sort(compareRepos);
}

function buildLazygitEnv(info: CommandFindLazygitRtnData, platform: NodeJS.Platform): Record<string, string> {
    const binaryDir = stripTrailingSeparators(info?.dir ?? "");
    if (isBlank(binaryDir)) {
        return {};
    }

    const pathSep = platform === "win32" ? ";" : ":";
    const pathKey = info?.pathkey || (platform === "win32" ? "Path" : "PATH");
    const existingPath = info?.pathvalue ?? "";
    const nextPath = isBlank(existingPath) ? binaryDir : `${binaryDir}${pathSep}${existingPath}`;
    const envMap: Record<string, string> = {
        [pathKey]: nextPath,
    };

    if (platform === "win32") {
        envMap["Path"] = nextPath;
        envMap["PATH"] = nextPath;
    }

    return envMap;
}

function statusBadge(label: string, active = false) {
    return (
        <span
            className={clsx(
                "inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] uppercase tracking-[0.14em]",
                active
                    ? "border-white/20 bg-white/10 text-white"
                    : "border-white/10 bg-black/20 text-secondary"
            )}
        >
            {label}
        </span>
    );
}

export class GitLauncherViewModel implements ViewModel {
    blockId: string;
    nodeModel: BlockNodeModel;
    tabModel: TabModel;
    viewType = "gitlauncher";
    viewIcon = atom("solid@code-branch");
    viewName = atom("Git");
    noPadding = atom(true);
    viewComponent = GitLauncherView;

    constructor({ blockId, nodeModel, tabModel }: ViewModelInitType) {
        this.blockId = blockId;
        this.nodeModel = nodeModel;
        this.tabModel = tabModel;
    }
}

export const GitLauncherView: React.FC<ViewComponentProps<GitLauncherViewModel>> = React.memo(function GitLauncherView({
    model,
}) {
    const env = useWaveEnv<GitLauncherEnv>();
    const workspaceId = useAtomValue(env.atoms.workspaceId);
    const workspace = useAtomValue(env.atoms.workspace);
    const activeTabId = workspace?.activetabid ?? "";

    const [autoDiscovery, setAutoDiscovery] = React.useState<DiscoveryResult>(emptyDiscovery());
    const [manualDiscovery, setManualDiscovery] = React.useState<DiscoveryResult>(emptyDiscovery());
    const [lazygitInfo, setLazygitInfo] = React.useState<CommandFindLazygitRtnData | null>(null);
    const [manualRoot, setManualRoot] = React.useState("");
    const [loading, setLoading] = React.useState(true);
    const [refreshEpoch, setRefreshEpoch] = React.useState(0);
    const [manualScanBusy, setManualScanBusy] = React.useState(false);
    const [launchingRepo, setLaunchingRepo] = React.useState("");
    const [statusMessage, setStatusMessage] = React.useState("");
    const lastFallbackRootRef = React.useRef("");

    React.useEffect(() => {
        let disposed = false;

        const runDiscovery = async () => {
            setLoading(true);
            setStatusMessage("");
            try {
                const [nextLazygitInfo, nextAutoDiscovery] = await Promise.all([
                    env.rpc.FindLazygitCommand(TabRpcClient, refreshEpoch > 0, { timeout: 4000 }),
                    discoverWorkspaceRepos(env.rpc, workspaceId, activeTabId, env.platform),
                ]);
                if (disposed) {
                    return;
                }
                setLazygitInfo(nextLazygitInfo);
                setAutoDiscovery(nextAutoDiscovery);
                setManualDiscovery(emptyDiscovery());
            } catch (error) {
                if (!disposed) {
                    setStatusMessage(String(error));
                    setAutoDiscovery(emptyDiscovery());
                }
            } finally {
                if (!disposed) {
                    setLoading(false);
                }
            }
        };

        runDiscovery();
        return () => {
            disposed = true;
        };
    }, [activeTabId, env.platform, env.rpc, refreshEpoch, workspaceId]);

    React.useEffect(() => {
        const nextFallbackRoot = autoDiscovery.fallbackRoot;
        if (isBlank(nextFallbackRoot)) {
            lastFallbackRootRef.current = nextFallbackRoot;
            return;
        }
        setManualRoot((currentValue) => {
            if (isBlank(currentValue) || currentValue === lastFallbackRootRef.current) {
                return nextFallbackRoot;
            }
            return currentValue;
        });
        lastFallbackRootRef.current = nextFallbackRoot;
    }, [autoDiscovery.fallbackRoot]);

    const mergedRepos = React.useMemo(
        () => mergeRepoLists(env.platform, autoDiscovery.repos, manualDiscovery.repos),
        [autoDiscovery.repos, env.platform, manualDiscovery.repos]
    );

    const mergedErrors = React.useMemo(
        () => dedupeStrings([...autoDiscovery.errors, ...manualDiscovery.errors, statusMessage].filter(Boolean)),
        [autoDiscovery.errors, manualDiscovery.errors, statusMessage]
    );

    async function handleManualScan() {
        if (isBlank(manualRoot)) {
            setStatusMessage("Enter a folder to scan for repositories.");
            return;
        }
        setManualScanBusy(true);
        setStatusMessage("");
        try {
            const result = await discoverReposFromRoot(env.rpc, manualRoot, env.platform);
            setManualDiscovery(result);
        } catch (error) {
            setStatusMessage(String(error));
        } finally {
            setManualScanBusy(false);
        }
    }

    async function handleLaunchRepo(repo: RepoEntry) {
        if (!lazygitInfo?.found) {
            return;
        }
        setLaunchingRepo(repo.root);
        setStatusMessage("");
        try {
            await replaceBlock(
                model.blockId,
                {
                    meta: {
                        view: "term",
                        controller: "cmd",
                        connection: "local",
                        cmd: "lazygit",
                        "cmd:shell": false,
                        "cmd:cwd": repo.root,
                        "cmd:env": buildLazygitEnv(lazygitInfo, env.platform),
                        "frame:title": `Git - ${repo.name}`,
                        "frame:icon": "solid@code-branch",
                        "icon:color": "#ffffff",
                    },
                },
                true
            );
        } catch (error) {
            setStatusMessage(String(error));
            setLaunchingRepo("");
        }
    }

    const noWorkspaceRoots = !loading && autoDiscovery.roots.length === 0;
    const noReposFound = !loading && mergedRepos.length === 0;
    const autoTruncated = autoDiscovery.truncated || manualDiscovery.truncated;

    return (
        <div className="flex h-full w-full flex-col overflow-hidden bg-[radial-gradient(circle_at_top,_rgba(255,255,255,0.07),_transparent_55%),linear-gradient(180deg,_rgba(255,255,255,0.03),_rgba(0,0,0,0.18))]">
            <div className="flex flex-wrap items-center gap-2 border-b border-white/10 bg-black/20 px-3 py-2">
                <div className="text-sm font-semibold text-primary">Git Launcher</div>
                {lazygitInfo?.found ? statusBadge("lazygit ready", true) : statusBadge("lazygit missing")}
                {autoDiscovery.roots.length > 0 ? statusBadge(`${autoDiscovery.roots.length} roots`) : null}
                {mergedRepos.length > 0 ? statusBadge(`${mergedRepos.length} repos`) : null}
                <button
                    type="button"
                    className="ml-auto rounded-md border border-white/10 bg-white/5 px-2.5 py-1 text-xs text-secondary transition-colors hover:bg-white/10 hover:text-white"
                    onClick={() => setRefreshEpoch((value) => value + 1)}
                >
                    Refresh
                </button>
            </div>

            <div className="flex-1 overflow-auto">
                <div className="mx-auto flex w-full max-w-4xl flex-col gap-3 p-3">
                    {!lazygitInfo?.found && !loading ? (
                        <div className="rounded-xl border border-amber-400/20 bg-amber-400/10 px-3 py-3 text-sm text-amber-100">
                            <div className="font-medium">Lazygit is not installed on this machine.</div>
                            <div className="mt-1 text-xs text-amber-100/80">
                                Install `lazygit`, then press Refresh. This widget re-checks automatically instead of
                                relying on a hardcoded path.
                            </div>
                        </div>
                    ) : null}

                    <div className="rounded-xl border border-white/10 bg-black/20 p-3">
                        <div className="flex flex-wrap items-center gap-2">
                            <div className="text-xs font-semibold uppercase tracking-[0.18em] text-secondary">
                                Scan Root
                            </div>
                            <div className="text-xs text-secondary">
                                Scan additional repositories inside folders currently used by open terminals or file
                                previews in this workspace.
                            </div>
                        </div>
                        <div className="mt-3 flex flex-col gap-2 sm:flex-row">
                            <input
                                type="text"
                                value={manualRoot}
                                onChange={(event) => setManualRoot(event.target.value)}
                                placeholder="Enter a folder to scan..."
                                className="min-w-0 flex-1 rounded-lg border border-white/10 bg-black/30 px-3 py-2 text-sm text-primary outline-none transition-colors placeholder:text-secondary focus:border-white/20"
                                onKeyDown={(event) => {
                                    if (event.key === "Enter") {
                                        event.preventDefault();
                                        handleManualScan();
                                    }
                                }}
                            />
                            <button
                                type="button"
                                onClick={handleManualScan}
                                disabled={manualScanBusy}
                                className="rounded-lg border border-white/10 bg-white/5 px-3 py-2 text-sm text-secondary transition-colors hover:bg-white/10 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
                            >
                                {manualScanBusy ? "Scanning..." : "Scan"}
                            </button>
                        </div>
                        {autoDiscovery.roots.length > 0 ? (
                            <div className="mt-3 flex flex-wrap gap-2">
                                {autoDiscovery.roots.map((root) => (
                                    <div
                                        key={root.path}
                                        className={clsx(
                                            "rounded-full border px-2.5 py-1 text-[11px] font-mono",
                                            root.activeTab
                                                ? "border-white/20 bg-white/10 text-white"
                                                : "border-white/10 bg-black/20 text-secondary"
                                        )}
                                        title={root.path}
                                    >
                                        {root.path}
                                    </div>
                                ))}
                            </div>
                        ) : null}
                    </div>

                    {loading ? (
                        <div className="rounded-xl border border-white/10 bg-black/20 px-3 py-10 text-center text-sm text-secondary">
                            Checking for `lazygit` and discovering repositories in the workspace...
                        </div>
                    ) : null}

                    {mergedErrors.length > 0 ? (
                        <div className="rounded-xl border border-red-400/20 bg-red-500/10 px-3 py-3 text-xs text-red-100">
                            {mergedErrors.map((error) => (
                                <div key={error}>{error}</div>
                            ))}
                        </div>
                    ) : null}

                    {autoTruncated ? (
                        <div className="rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-xs text-secondary">
                            Automatic scanning is capped at {MaxScanDirs} folders and {MaxScanDepth} levels deep so the
                            widget stays lightweight and resizable.
                        </div>
                    ) : null}

                    {noWorkspaceRoots ? (
                        <div className="rounded-xl border border-white/10 bg-black/20 px-3 py-4 text-sm text-secondary">
                            Could not resolve any local working directories from open terminals or file previews in the
                            current workspace. You can still enter a folder above to scan manually.
                        </div>
                    ) : null}

                    {noReposFound ? (
                        <div className="rounded-xl border border-white/10 bg-black/20 px-3 py-8 text-center text-sm text-secondary">
                            No repositories were found from the current workspace.
                        </div>
                    ) : null}

                    {mergedRepos.length > 0 ? (
                        <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
                            {mergedRepos.map((repo) => {
                                const isLaunching = launchingRepo !== "" && pathsEqual(launchingRepo, repo.root, env.platform);
                                return (
                                    <button
                                        key={repo.root}
                                        type="button"
                                        disabled={!lazygitInfo?.found || isLaunching}
                                        onClick={() => handleLaunchRepo(repo)}
                                        className={clsx(
                                            "group rounded-2xl border border-white/10 bg-white/[0.035] px-3 py-3 text-left transition-colors",
                                            "hover:border-white/20 hover:bg-white/[0.06]",
                                            "disabled:cursor-not-allowed disabled:opacity-60"
                                        )}
                                    >
                                        <div className="flex items-start gap-3">
                                            <div className="pt-0.5 text-lg text-white/90">
                                                <i className="fa fa-solid fa-code-branch" />
                                            </div>
                                            <div className="min-w-0 flex-1">
                                                <div className="flex flex-wrap items-center gap-2">
                                                    <div className="truncate text-sm font-semibold text-primary">
                                                        {repo.name}
                                                    </div>
                                                    {repo.activeTab ? statusBadge("active", true) : null}
                                                    {repo.sources.slice(0, 2).map((source) => (
                                                        <React.Fragment key={`${repo.root}-${source}`}>
                                                            {statusBadge(source)}
                                                        </React.Fragment>
                                                    ))}
                                                </div>
                                                <div className="mt-1 truncate font-mono text-[11px] text-secondary">
                                                    {repo.root}
                                                </div>
                                            </div>
                                            <div className="shrink-0 pt-0.5 text-xs text-secondary transition-colors group-hover:text-white">
                                                {isLaunching ? "Opening..." : "Open"}
                                            </div>
                                        </div>
                                    </button>
                                );
                            })}
                        </div>
                    ) : null}
                </div>
            </div>
        </div>
    );
});
