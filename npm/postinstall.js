#!/usr/bin/env node
"use strict";

const { execSync } = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");
const https = require("https");

const REPO = "wungjyan/aicommit";
const BINARY = "aicommit";

function getPlatform() {
  switch (os.platform()) {
    case "darwin": return "darwin";
    case "linux": return "linux";
    case "win32": return "windows";
    default: throw new Error(`Unsupported platform: ${os.platform()}`);
  }
}

function getArch() {
  switch (os.arch()) {
    case "x64": return "amd64";
    case "arm64": return "arm64";
    default: throw new Error(`Unsupported architecture: ${os.arch()}`);
  }
}

function getLatestVersion() {
  return new Promise((resolve, reject) => {
    const options = {
      hostname: "api.github.com",
      path: `/repos/${REPO}/releases/latest`,
      headers: { "User-Agent": "@wungjyan/aicommit" },
    };
    https.get(options, (res) => {
      let data = "";
      res.on("data", (chunk) => (data += chunk));
      res.on("end", () => {
        try {
          const json = JSON.parse(data);
          const tag = json.tag_name || "";
          const version = tag.replace(/^v/, "");
          resolve(version);
        } catch (e) {
          reject(new Error(`Failed to parse GitHub API response: ${e.message}`));
        }
      });
    }).on("error", reject);
  });
}

function download(url, dest) {
  return new Promise((resolve, reject) => {
    const request = (reqUrl, redirects = 0) => {
      if (redirects > 5) return reject(new Error("Too many redirects"));
      https.get(reqUrl, { headers: { "User-Agent": "@wungjyan/aicommit" } }, (res) => {
        if ([301, 302, 307, 308].includes(res.statusCode)) {
          return request(res.headers.location, redirects + 1);
        }
        if (res.statusCode !== 200) {
          return reject(new Error(`Download failed: HTTP ${res.statusCode}`));
        }
        const file = fs.createWriteStream(dest, { mode: 0o755 });
        res.pipe(file);
        file.on("finish", () => file.close(resolve));
        file.on("error", reject);
      }).on("error", reject);
    };
    request(url);
  });
}

async function main() {
  const platform = getPlatform();
  const arch = getArch();
  const ext = platform === "windows" ? ".exe" : "";
  const version = await getLatestVersion();
  const filename = `${BINARY}-${platform}-${arch}${ext}`;
  const url = `https://github.com/${REPO}/releases/download/v${version}/${filename}`;

  const binDir = path.join(__dirname, "bin");
  const dest = path.join(binDir, `${BINARY}${ext}`);

  console.log(`Downloading aicommit v${version} for ${platform}/${arch}...`);
  await download(url, dest);

  if (platform !== "windows") {
    fs.chmodSync(dest, 0o755);
  }

  console.log("Done.");
}

main().catch((err) => {
  console.error("postinstall failed:", err.message);
  process.exit(1);
});
