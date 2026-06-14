import { spawn, spawnSync } from "node:child_process";

const isWindows = process.platform === "win32";
const rootDir = new URL("../../..", import.meta.url).pathname.replace(/^\/([A-Z]:)/, "$1");
const frontendDir = new URL("../..", import.meta.url).pathname.replace(/^\/([A-Z]:)/, "$1");
const backendDir = `${rootDir}/backend`;
const args = process.argv.slice(2);

const backendEnv = {
  ...process.env,
  APP_ENV: "development",
  GOCACHE: `${backendDir}/tmp/go-cache`,
  PORT: "8080",
  DB_HOST: process.env.E2E_DB_HOST ?? "127.0.0.1",
  DB_PORT: process.env.E2E_DB_PORT ?? "5432",
  DB_USER: process.env.E2E_DB_USER ?? "postgres",
  DB_PASSWORD: process.env.E2E_DB_PASSWORD ?? "postgres",
  DB_NAME: process.env.E2E_DB_NAME ?? "stylemind",
  JWT_SECRET: "change_me_for_e2e",
  JWT_ISSUER: "stylemind-api",
  JWT_AUDIENCE: "stylemind-web",
  REQUEST_TIMEOUT_SECONDS: "10",
  MAX_REQUEST_BODY_BYTES: "1048576",
  AUTH_RATE_LIMIT_REQUESTS: "1000",
  REDIS_ADDR: process.env.E2E_REDIS_ADDR ?? "",
  REDIS_PASSWORD: "",
  REDIS_DB: "0",
  CORS_ALLOWED_ORIGINS: "http://127.0.0.1:3000,http://localhost:3000",
};

function spawnManaged(command, commandArgs, options) {
  return spawn(command, commandArgs, {
    stdio: "inherit",
    shell: false,
    ...options,
  });
}

function killTree(child) {
  if (!child || child.killed || !child.pid) {
    return;
  }
  if (isWindows) {
    spawnSync("taskkill", ["/pid", String(child.pid), "/T", "/F"], { stdio: "ignore" });
    return;
  }
  child.kill("SIGTERM");
}

async function waitFor(url, timeoutMs) {
  const startedAt = Date.now();
  let lastError;
  while (Date.now() - startedAt < timeoutMs) {
    try {
      const response = await fetch(url);
      if (response.ok) {
        return;
      }
      lastError = new Error(`${url} returned ${response.status}`);
    } catch (error) {
      lastError = error;
    }
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  throw lastError ?? new Error(`${url} was not ready`);
}

const backend = spawnManaged("node", ["../frontend/tests/e2e/start-backend.mjs"], {
  cwd: backendDir,
  env: backendEnv,
});
const frontend = spawnManaged("node", ["tests/e2e/start-frontend.mjs"], {
  cwd: frontendDir,
  env: {
    ...process.env,
    NEXT_PUBLIC_API_BASE_URL: "http://127.0.0.1:8080/api/v1",
  },
});

let exitCode = 1;

try {
  await waitFor("http://127.0.0.1:8080/readyz", 120_000);
  await waitFor("http://127.0.0.1:3000", 120_000);

  const playwrightCommand = isWindows ? "cmd.exe" : "npx";
  const playwrightArgs = isWindows
    ? ["/d", "/s", "/c", ["npx.cmd", "playwright", "test", ...args].join(" ")]
    : ["playwright", "test", ...args];
  const playwright = spawnManaged(playwrightCommand, playwrightArgs, {
    cwd: frontendDir,
    env: {
      ...process.env,
      PW_SKIP_WEBSERVER: "1",
      NEXT_PUBLIC_API_BASE_URL: "http://127.0.0.1:8080/api/v1",
    },
  });

  exitCode = await new Promise((resolve) => {
    playwright.on("exit", (code) => resolve(code ?? 1));
  });
} finally {
  killTree(frontend);
  killTree(backend);
}

process.exit(exitCode);
