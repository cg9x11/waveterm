const fs = require("fs");
const path = require("path");

function patchElectronBuilderWinRcedit() {
    const targetFile = path.join(__dirname, "..", "node_modules", "app-builder-lib", "out", "winPackager.js");
    if (!fs.existsSync(targetFile)) {
        console.log("postinstall: electron-builder winPackager.js not found, skipping RCEdit patch");
        return;
    }

    const source = fs.readFileSync(targetFile, "utf8");
    const marker = "This avoids the legacy app-builder winCodeSign archive path";
    if (source.includes(marker)) {
        console.log("postinstall: electron-builder RCEdit patch already applied");
        return;
    }

    const search = `        // rcedit crashed of executed using wine, resourcehacker works
        if (process.platform === "win32" || process.platform === "darwin") {
            await (0, builder_util_1.executeAppBuilder)(["rcedit", "--args", JSON.stringify(args)], undefined /* child-process */, {}, 3 /* retry three times */);
        }
        else if (this.info.framework.name === "electron") {
            const vendor = await (0, windows_1.getRceditBundle)((_c = this.config.toolsets) === null || _c === void 0 ? void 0 : _c.winCodeSign);
            await (0, wine_1.execWine)(vendor.x86, vendor.x64, args);
        }`;

    const replacement = `        // On Windows, prefer the standalone rcedit bundle selected by the configured toolset.
        // This avoids the legacy app-builder winCodeSign archive path, which can fail to extract
        // on systems without symlink privileges.
        if (process.platform === "win32") {
            const vendor = await (0, windows_1.getRceditBundle)((_c = this.config.toolsets) === null || _c === void 0 ? void 0 : _c.winCodeSign);
            await (0, builder_util_1.exec)(process.arch === "ia32" ? vendor.x86 : vendor.x64, args);
        }
        else if (process.platform === "darwin") {
            await (0, builder_util_1.executeAppBuilder)(["rcedit", "--args", JSON.stringify(args)], undefined /* child-process */, {}, 3 /* retry three times */);
        }
        else if (this.info.framework.name === "electron") {
            const vendor = await (0, windows_1.getRceditBundle)((_c = this.config.toolsets) === null || _c === void 0 ? void 0 : _c.winCodeSign);
            await (0, wine_1.execWine)(vendor.x86, vendor.x64, args);
        }`;

    if (!source.includes(search)) {
        throw new Error("postinstall: electron-builder RCEdit patch target changed; review patch script");
    }

    fs.writeFileSync(targetFile, source.replace(search, replacement));
    console.log("postinstall: applied electron-builder RCEdit patch for Windows packaging");
}

module.exports = patchElectronBuilderWinRcedit;

if (require.main === module) {
    patchElectronBuilderWinRcedit();
}
