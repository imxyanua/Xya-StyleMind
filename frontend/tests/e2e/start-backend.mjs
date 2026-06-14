import { mkdirSync } from "node:fs";
import { join } from "node:path";
import { spawn, spawnSync } from "node:child_process";

const backendDir = process.cwd();
const outputDir = join(backendDir, "tmp");
const executablePath = join(outputDir, process.platform === "win32" ? "e2e-server.exe" : "e2e-server");

mkdirSync(outputDir, { recursive: true });

const build = spawnSync("go", ["build", "-o", executablePath, "./cmd/server"], {
  cwd: backendDir,
  stdio: "inherit",
});

if (build.status !== 0) {
  process.exit(build.status ?? 1);
}

const server = spawn(executablePath, {
  cwd: backendDir,
  env: process.env,
  stdio: "inherit",
  shell: false,
});

let shuttingDown = false;

function shutdown(signal) {
  if (shuttingDown) {
    return;
  }
  shuttingDown = true;
  if (!server.killed) {
    if (process.platform === "win32" && server.pid) {
      spawnSync("taskkill", ["/pid", String(server.pid), "/T", "/F"], { stdio: "ignore" });
    } else {
      server.kill(signal);
    }
  }
  process.exit(0);
}

for (const signal of ["SIGINT", "SIGTERM", "SIGHUP"]) {
  process.on(signal, () => shutdown(signal));
}

server.on("exit", (code, signal) => {
  void signal;
  process.exit(code ?? 0);
});
