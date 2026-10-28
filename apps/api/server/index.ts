import "dotenv/config";
import express from "express";
import { createExpressMiddleware } from "@trpc/server/adapters/express";
import { appRouter } from "./router.js";
import { createContext } from "./context.js";

const app = express();
app.use("/trpc", createExpressMiddleware({ router: appRouter, createContext }));
app.get("/health", (_req, res) => res.json({ status: "ok" }));
const port = Number(process.env.PORT ?? 4000);
app.listen(port, "0.0.0.0", () => console.info(`tRPC API console listening on :${port}`));
