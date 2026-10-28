"use client";
import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { AuthError } from "@arbeiter/auth-sdk";
import { createAuthClient } from "../lib/authClient";
import type { User } from "../types";

const auth = createAuthClient();

export function useAuthPanel() {
  const [user, setUser] = useState<User | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void auth.me().then(value => setUser(value as User)).catch(() => setUser(null));
  }, []);

  const signIn = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setBusy(true);
    setMessage("");
    try {
      await auth.signIn(email, password);
      setUser(await auth.me() as User);
      setMessage("You are signed in securely.");
    } catch (error) {
      setMessage(error instanceof AuthError ? error.message : "Unable to reach the authentication API. Check its URL and try again.");
    } finally {
      setBusy(false);
    }
  };

  const signOut = async () => {
    await auth.signOut();
    setUser(null);
    setMessage("You have signed out.");
  };

  return { user, email, password, message, busy, setEmail, setPassword, signIn, signOut };
}
