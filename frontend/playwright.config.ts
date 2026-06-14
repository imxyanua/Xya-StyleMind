import { defineConfig, devices } from "@playwright/test";

const frontendDevCommand =
  process.platform === "win32"
    ? "node tests/e2e/start-frontend.mjs"
    : "npm run dev -- -H 127.0.0.1 -p 3000";

const backendDevCommand =
  process.platform === "win32"
    ? "node ../frontend/tests/e2e/start-backend.mjs"
    : "go run ./cmd/server";

const backendEnv = {
  APP_ENV: "development",
  GOCACHE: "tmp/go-cache",
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

const webServer = process.env.PW_SKIP_WEBSERVER === "1"
  ? undefined
  : [
      {
        command: backendDevCommand,
        cwd: "../backend",
        env: backendEnv,
        url: "http://127.0.0.1:8080/readyz",
        timeout: 120_000,
        reuseExistingServer: true,
      },
      {
        command: frontendDevCommand,
        env: {
          NEXT_PUBLIC_API_BASE_URL: "http://127.0.0.1:8080/api/v1",
        },
        url: "http://127.0.0.1:3000",
        timeout: 120_000,
        reuseExistingServer: true,
      },
    ];

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 90_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  workers: 1,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "retain-on-failure",
    ...devices["Desktop Chrome"],
  },
  webServer,
});
