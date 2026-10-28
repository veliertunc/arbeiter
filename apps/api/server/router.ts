import { initTRPC, TRPCError } from "@trpc/server";
import { z } from "zod";
import type { Context } from "./context.js";

const t = initTRPC.context<Context>().create();
const call = async <T>(action: () => Promise<T>): Promise<T> => {
  try { return await action(); }
  catch (error) {
    const candidate = error as { status?: number; message?: string };
    throw new TRPCError({ code: candidate.status === 401 ? "UNAUTHORIZED" : "BAD_REQUEST", message: candidate.message ?? "Authentication API request failed", cause: error });
  }
};

export const appRouter = t.router({
  auth: t.router({
    me: t.procedure.query(({ ctx }) => call(() => ctx.auth.me())),
    signIn: t.procedure.input(z.object({ email: z.email(), password: z.string().min(1) })).mutation(({ ctx, input }) => call(() => ctx.auth.signIn(input.email, input.password))),
    signUp: t.procedure.input(z.object({ email: z.email(), password: z.string().min(12), name: z.string().optional() })).mutation(({ ctx, input }) => call(() => ctx.auth.signUp(input.email, input.password, input.name))),
    signOut: t.procedure.mutation(({ ctx }) => call(() => ctx.auth.signOut())),
    updateName: t.procedure.input(z.object({ name: z.string().min(1).max(120) })).mutation(({ ctx, input }) => call(() => ctx.auth.updateMe(input.name))),
  }),
  sessions: t.procedure.query(({ ctx }) => call(() => ctx.auth.listSessions())),
  organizations: t.router({
    list: t.procedure.query(({ ctx }) => call(() => ctx.auth.listOrganizations())),
    create: t.procedure.input(z.object({ name: z.string().min(1).max(120), slug: z.string().min(2).max(60).regex(/^[a-z0-9-]+$/) })).mutation(({ ctx, input }) => call(() => ctx.auth.createOrganization(input.name, input.slug))),
  }),
});

export type AppRouter = typeof appRouter;
