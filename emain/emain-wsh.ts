// Copyright 2025, Command Line Inc.
// SPDX-License-Identifier: Apache-2.0

import { WindowService } from "@/app/store/services";
import { RpcResponseHelper, WshClient } from "@/app/store/wshclient";
import { RpcApi } from "@/app/store/wshclientapi";
import { app as electronApp, net, Notification, safeStorage, shell } from "electron";
import { setForceQuit, setUserConfirmedQuit } from "emain/emain-activity";
import { getResolvedUpdateChannel } from "emain/updater";
import { unamePlatform } from "./emain-platform";
import { getWebContentsByBlockId, webGetSelector } from "./emain-web";
import {
    createBrowserWindow,
    createNewWaveWindow,
    createWindowForWorkspace,
    focusedWaveWindow,
    getAllWaveWindows,
    getWaveWindowById,
    getWaveWindowByWorkspaceId,
} from "./emain-window";

export class ElectronWshClientType extends WshClient {
    constructor() {
        super("electron");
    }

    async handle_webselector(rh: RpcResponseHelper, data: CommandWebSelectorData): Promise<string[]> {
        if (!data.tabid || !data.blockid || !data.workspaceid) {
            throw new Error("tabid and blockid are required");
        }
        const ww = getWaveWindowByWorkspaceId(data.workspaceid);
        if (ww == null) {
            throw new Error(`no window found with workspace ${data.workspaceid}`);
        }
        const wc = await getWebContentsByBlockId(ww, data.tabid, data.blockid);
        if (wc == null) {
            throw new Error(`no webcontents found with blockid ${data.blockid}`);
        }
        const rtn = await webGetSelector(wc, data.selector, data.opts);
        return rtn;
    }

    async handle_notify(rh: RpcResponseHelper, notificationOptions: WaveNotificationOptions) {
        new Notification({
            title: notificationOptions.title,
            body: notificationOptions.body,
            silent: notificationOptions.silent,
        }).show();
    }

    async handle_getupdatechannel(rh: RpcResponseHelper): Promise<string> {
        return getResolvedUpdateChannel();
    }

    async handle_focuswindow(rh: RpcResponseHelper, windowId: string) {
        console.log(`focuswindow ${windowId}`);
        const fullConfig = await RpcApi.GetFullConfigCommand(ElectronWshClient);
        let ww = getWaveWindowById(windowId);
        if (ww == null) {
            const window = await WindowService.GetWindow(windowId);
            if (window == null) {
                throw new Error(`window ${windowId} not found`);
            }
            ww = await createBrowserWindow(window, fullConfig, {
                unamePlatform,
                isPrimaryStartupWindow: false,
            });
        }
        ww.focus();
    }

    async handle_opennewwindow(rh: RpcResponseHelper, workspaceId: string): Promise<string> {
        console.log(`opennewwindow workspace=${workspaceId || ""}`);
        if (workspaceId) {
            await createWindowForWorkspace(workspaceId);
            const ww = getWaveWindowByWorkspaceId(workspaceId);
            if (ww == null) {
                throw new Error(`workspace ${workspaceId} did not produce a visible window`);
            }
            ww.show();
            ww.focus();
            return ww.waveWindowId;
        }
        const existingIds = new Set(getAllWaveWindows().map((ww) => ww.waveWindowId));
        await createNewWaveWindow();
        let ww = focusedWaveWindow;
        if (ww == null || existingIds.has(ww.waveWindowId)) {
            ww = getAllWaveWindows().find((entry) => !existingIds.has(entry.waveWindowId)) ?? ww;
        }
        if (ww == null) {
            throw new Error("failed to create a new Wave window");
        }
        return ww.waveWindowId;
    }

    async handle_quitapp(rh: RpcResponseHelper, force: boolean): Promise<void> {
        console.log(`quitapp force=${!!force}`);
        setUserConfirmedQuit(true);
        if (force) {
            setForceQuit(true);
        }
        setTimeout(() => {
            electronApp.quit();
        }, 0);
    }

    async handle_capturewindowscreenshot(rh: RpcResponseHelper, windowId: string): Promise<string> {
        const ww = (windowId ? getWaveWindowById(windowId) : null) ?? focusedWaveWindow;
        if (ww == null) {
            throw new Error(windowId ? `window ${windowId} not found` : "no active Wave window found");
        }

        let image: Electron.NativeImage = null;
        const maybeCapturePage = (ww as unknown as { capturePage?: () => Promise<Electron.NativeImage> }).capturePage;
        if (typeof maybeCapturePage === "function") {
            try {
                image = await maybeCapturePage.call(ww);
            } catch (err) {
                console.log("capturewindowscreenshot falling back to active tab capture", err);
            }
        }
        if (image == null) {
            if (ww.activeTabView?.webContents == null) {
                throw new Error(`window ${ww.waveWindowId} has no active tab view to capture`);
            }
            image = await ww.activeTabView.webContents.capturePage();
        }
        return `data:image/png;base64,${image.toPNG().toString("base64")}`;
    }

    async handle_electronencrypt(
        rh: RpcResponseHelper,
        data: CommandElectronEncryptData
    ): Promise<CommandElectronEncryptRtnData> {
        if (!safeStorage.isEncryptionAvailable()) {
            throw new Error("encryption is not available");
        }
        const encrypted = safeStorage.encryptString(data.plaintext);
        const ciphertext = encrypted.toString("base64");

        let storagebackend = "";
        if (process.platform === "linux") {
            storagebackend = safeStorage.getSelectedStorageBackend();
        }

        return {
            ciphertext,
            storagebackend,
        };
    }

    async handle_electrondecrypt(
        rh: RpcResponseHelper,
        data: CommandElectronDecryptData
    ): Promise<CommandElectronDecryptRtnData> {
        if (!safeStorage.isEncryptionAvailable()) {
            throw new Error("encryption is not available");
        }
        const encrypted = Buffer.from(data.ciphertext, "base64");
        const plaintext = safeStorage.decryptString(encrypted);

        let storagebackend = "";
        if (process.platform === "linux") {
            storagebackend = safeStorage.getSelectedStorageBackend();
        }

        return {
            plaintext,
            storagebackend,
        };
    }

    async handle_networkonline(rh: RpcResponseHelper): Promise<boolean> {
        return net.isOnline();
    }

    async handle_electronsystembell(rh: RpcResponseHelper): Promise<void> {
        shell.beep();
    }

    // async handle_workspaceupdate(rh: RpcResponseHelper) {
    //     console.log("workspaceupdate");
    //     fireAndForget(async () => {
    //         console.log("workspace menu clicked");
    //         const updatedWorkspaceMenu = await getWorkspaceMenu();
    //         const workspaceMenu = Menu.getApplicationMenu().getMenuItemById("workspace-menu");
    //         workspaceMenu.submenu = Menu.buildFromTemplate(updatedWorkspaceMenu);
    //     });
    // }
}

export let ElectronWshClient: ElectronWshClientType;

export function initElectronWshClient() {
    ElectronWshClient = new ElectronWshClientType();
}
