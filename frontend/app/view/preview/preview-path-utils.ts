// Copyright 2026, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { PlatformWindows } from "@/util/platformutil";
import { isBlank, isLocalConnName } from "@/util/util";

export type DirectoryBookmark = {
    label: string;
    path: string;
    icon?: string;
};

export const WINDOWS_DRIVES_VIRTUAL_PATH = "__wave_windows_drives__";
export const SYNTHETIC_WINDOWS_DRIVE_META_KEY = "wave:syntheticdrive";

export function stripTrailingSeparators(path: string | null | undefined): string {
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

export function normalizePathKey(path: string | null | undefined, platform: NodeJS.Platform): string {
    const normalized = stripTrailingSeparators(path).replace(/[\\/]+/g, "/");
    if (platform === PlatformWindows) {
        return normalized.toLowerCase();
    }
    return normalized;
}

export function isSamePath(a: string | null | undefined, b: string | null | undefined, platform: NodeJS.Platform): boolean {
    return normalizePathKey(a, platform) === normalizePathKey(b, platform);
}

export function isWindowsDrivePath(path: string | null | undefined): boolean {
    return /^[a-zA-Z]:(?:[\\/]|$)/.test(path?.trim() ?? "");
}

export function isWindowsDriveRoot(path: string | null | undefined): boolean {
    return /^[a-zA-Z]:[\\/]?$/.test(stripTrailingSeparators(path));
}

export function isWindowsDrivesVirtualPath(path: string | null | undefined): boolean {
    return path === WINDOWS_DRIVES_VIRTUAL_PATH;
}

export function isWindowsFilesystemContext(
    platform: NodeJS.Platform,
    connection: string,
    path: string | null | undefined
): boolean {
    if (isWindowsDrivesVirtualPath(path) || isWindowsDrivePath(path) || isWindowsDriveRoot(path)) {
        return true;
    }
    return platform === PlatformWindows && isLocalConnName(connection);
}

export function getDirectoryBookmarks(
    platform: NodeJS.Platform,
    connection: string,
    path: string | null | undefined
): DirectoryBookmark[] {
    const bookmarks: DirectoryBookmark[] = [
        { label: "Home", path: "~", icon: "house" },
        { label: "Desktop", path: "~/Desktop", icon: "desktop" },
        { label: "Downloads", path: "~/Downloads", icon: "download" },
        { label: "Documents", path: "~/Documents", icon: "file-lines" },
    ];
    if (isWindowsFilesystemContext(platform, connection, path)) {
        return [...bookmarks, { label: "This PC", path: WINDOWS_DRIVES_VIRTUAL_PATH, icon: "hard-drive" }];
    }
    return [...bookmarks, { label: "Root", path: "/", icon: "folder-open" }];
}

export function formatDirectoryLocationLabel(path: string | null | undefined): string {
    if (isWindowsDrivesVirtualPath(path)) {
        return "This PC";
    }
    return path ?? "";
}

export function isNavigationRootPath(
    platform: NodeJS.Platform,
    connection: string,
    path: string | null | undefined
): boolean {
    if (isBlank(path)) {
        return false;
    }
    if (path === "/") {
        return true;
    }
    if (isWindowsFilesystemContext(platform, connection, path) && isWindowsDrivesVirtualPath(path)) {
        return true;
    }
    return false;
}

export function getWindowsDriveLabel(path: string | null | undefined): string {
    const match = stripTrailingSeparators(path).match(/^([a-zA-Z]:)/);
    if (match == null) {
        return "";
    }
    return match[1].toUpperCase();
}
