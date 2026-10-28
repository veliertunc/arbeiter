import { useCallback, useEffect, useState } from "react";
import type { FormEvent } from "react";
import { trpc } from "../trpc";
import type { ConsoleTab, Organization, User } from "../types";

export function useAuthConsole() {
  const [user, setUser] = useState<User | null>(null);
  const [organizations, setOrganizations] = useState<Organization[]>([]);
  const [sessions, setSessions] = useState<unknown[]>([]);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [notice, setNotice] = useState("");
  const [tab, setTab] = useState<ConsoleTab>("Overview");
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const currentUser = await trpc.auth.me.query();
      const [orgs, activeSessions] = await Promise.all([
        trpc.organizations.list.query(),
        trpc.sessions.query(),
      ]);
      setUser(currentUser as User);
      setOrganizations(Array.isArray(orgs) ? orgs as Organization[] : []);
      setSessions(Array.isArray(activeSessions) ? activeSessions : []);
    } catch {
      setUser(null);
      setOrganizations([]);
      setSessions([]);
    }
  }, []);

  useEffect(() => { void refresh(); }, [refresh]);

  const signIn = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setBusy(true);
    setNotice("");
    try {
      await trpc.auth.signIn.mutate({ email, password });
      await refresh();
      setNotice("Signed in. Your session is ready.");
    } catch (error) {
      setNotice(error instanceof Error ? error.message : "Could not sign in.");
    } finally {
      setBusy(false);
    }
  };

  const signOut = async () => {
    await trpc.auth.signOut.mutate();
    setUser(null);
    setOrganizations([]);
    setSessions([]);
    setNotice("Session ended.");
  };

  return {
    user, organizations, sessions, email, password, notice, tab, busy,
    setEmail, setPassword, setTab, refresh, signIn, signOut,
  };
}
