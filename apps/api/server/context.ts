import type { CreateExpressContextOptions } from "@trpc/server/adapters/express";
import { AuthClient } from "@arbeiter/auth-sdk";

export function createContext({ req, res }: CreateExpressContextOptions) {
  const baseURL = process.env.AUTH_API_URL ?? "http://localhost:8080";
  const auth = new AuthClient({
    baseURL,
    fetchImpl: async (input, init) => {
      const headers = new Headers(init?.headers);
      if (req.headers.cookie) headers.set("Cookie", req.headers.cookie);
      const response = await fetch(input, { ...init, headers });
      const cookieHeaders = response.headers.getSetCookie?.() ?? [response.headers.get("set-cookie")].filter((value): value is string => Boolean(value));
      if (cookieHeaders.length) res.setHeader("set-cookie", cookieHeaders);
      return response;
    },
  });
  return { auth };
}
export type Context = ReturnType<typeof createContext>;
