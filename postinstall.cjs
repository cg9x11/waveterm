const fs = require("fs");
const path = require("path");

const skip =
    process.env.WAVETERM_SKIP_APP_DEPS === "1" || process.env.CF_PAGES === "1" || process.env.CF_PAGES === "true";

function applyLocalPatches() {
    const patchScript = path.join(__dirname, "scripts", "patch-electron-builder-win-rcedit.cjs");
    if (fs.existsSync(patchScript)) {
        require(patchScript)();
    }
}

if (skip) {
    console.log("postinstall: skipping electron-builder install-app-deps");
    applyLocalPatches();
    process.exit(0);
}

import("child_process").then(({ execSync }) => {
    execSync("electron-builder install-app-deps", { stdio: "inherit" });
    applyLocalPatches();
});
