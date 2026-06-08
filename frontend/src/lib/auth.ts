import { apiRequest } from "@/lib/api";

const TOKEN_KEY = "stylemind_token";
const USER_KEY = "stylemind_user";

export type AuthUser = {
  id: string;
  email: string;
  full_name: string;
  role: string;
  status?: string;
};

type AuthPayload = {
  token: string;
  user: AuthUser;
};

type RegisterInput = {
  email: string;
  full_name: string;
  password: string;
};

type LoginInput = {
  email: string;
  password: string;
};

export function getToken(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  return window.localStorage.getItem(TOKEN_KEY);
}

export function isLoggedIn() {
  return Boolean(getToken());
}

export function setToken(token: string) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function setStoredUser(user: AuthUser) {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getStoredUser(): AuthUser | null {
  if (typeof window === "undefined") {
    return null;
  }
  const raw = window.localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as AuthUser;
  } catch {
    window.localStorage.removeItem(USER_KEY);
    return null;
  }
}

export function logout() {
  if (typeof window === "undefined") {
    return;
  }
  window.localStorage.removeItem(TOKEN_KEY);
  window.localStorage.removeItem(USER_KEY);
}

export async function register(input: RegisterInput) {
  const res = await apiRequest<AuthPayload>(
    "/auth/register",
    {
      method: "POST",
      body: JSON.stringify(input),
    },
    { auth: false }
  );

  if (res.data?.token) {
    setToken(res.data.token);
  }
  if (res.data?.user) {
    setStoredUser(res.data.user);
  }
  return res.data;
}

export async function login(input: LoginInput) {
  const res = await apiRequest<AuthPayload>(
    "/auth/login",
    {
      method: "POST",
      body: JSON.stringify(input),
    },
    { auth: false }
  );

  if (res.data?.token) {
    setToken(res.data.token);
  }
  if (res.data?.user) {
    setStoredUser(res.data.user);
  }
  return res.data;
}

export async function getMe() {
  return apiRequest<{ user_id: string; role: string }>("/auth/me", {
    method: "GET",
  });
}
