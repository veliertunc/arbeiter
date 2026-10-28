import * as SecureStore from "expo-secure-store";
import { AuthClient } from "@arbeiter/auth-sdk";

const SESSION_KEY = "arbeiter.session.cookie";
const baseURL = process.env.EXPO_PUBLIC_AUTH_API_URL ?? "http://localhost:8080";

export const authClient = new AuthClient({
  baseURL,
  fetchImpl: async (input, init) => {
    const headers = new Headers(init?.headers);
    const cookie = await SecureStore.getItemAsync(SESSION_KEY);
    if (cookie) headers.set("Cookie", cookie);

    const response = await fetch(input, { ...init, headers });
    const setCookie = response.headers.get("set-cookie");
    if (setCookie) {
      const cookiePair = setCookie.split(";", 1)[0];
      if (cookiePair === "session=" || /max-age=0/i.test(setCookie)) {
        await SecureStore.deleteItemAsync(SESSION_KEY);
      } else if (cookiePair.startsWith("session=")) {
        await SecureStore.setItemAsync(SESSION_KEY, cookiePair);
      }
    }
    return response;
  },
});

export async function clearStoredSession() {
  await SecureStore.deleteItemAsync(SESSION_KEY);
}
