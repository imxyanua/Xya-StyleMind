import { spawn, spawnSync } from "node:child_process";

const command = process.platform === "win32" ? "cmd.exe" : "npm";
const args =
  process.platform === "win32"
    ? ["/d", "/s", "/c", "npm.cmd run dev -- -H 127.0.0.1 -p 3000"]
    : ["run", "dev", "--", "-H", "127.0.0.1", "-p", "3000"];

const server = spawn(command, args, {
  cwd: process.cwd(),
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
