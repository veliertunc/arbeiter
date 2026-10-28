import { useEffect, useState } from "react";
import { AuthError } from "@arbeiter/auth-sdk";
import { authClient, clearStoredSession } from "../auth/client";
import type { User } from "../types";

export function useAuth() {
  const [user, setUser] = useState<User | null>(null);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [name, setName] = useState("");
  const [isSignUp, setIsSignUp] = useState(false);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("");

  const refresh = async () => {
    try {
      setUser(await authClient.me() as User);
    } catch {
      setUser(null);
    }
  };

  useEffect(() => { void refresh().finally(() => setBusy(false)); }, []);

  const submit = async () => {
    setBusy(true);
    setMessage("");
    try {
      if (isSignUp) await authClient.signUp(email, password, name || undefined);
      else await authClient.signIn(email, password);
      await refresh();
      setMessage(isSignUp ? "Your account is ready." : "You are signed in.");
    } catch (error) {
      setMessage(error instanceof AuthError ? error.message : "Could not connect to the authentication service.");
    } finally {
      setBusy(false);
    }
  };

  const signOut = async () => {
    await authClient.signOut();
    await clearStoredSession();
    setUser(null);
    setMessage("You have signed out.");
  };

  return {
    user, email, password, name, isSignUp, busy, message,
    setEmail, setPassword, setName, setIsSignUp, submit, signOut,
  };
}
