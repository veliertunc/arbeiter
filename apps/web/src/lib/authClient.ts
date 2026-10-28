import { AuthClient } from "@arbeiter/auth-sdk";

export function createAuthClient() {
  return new AuthClient({
    baseURL: process.env.NEXT_PUBLIC_AUTH_API_URL ?? "http://localhost:8080",
  });
}
