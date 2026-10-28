package com.arbeiter.auth;

import com.sun.net.httpserver.HttpServer;
import org.junit.jupiter.api.AfterEach;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.charset.StandardCharsets;
import java.util.concurrent.atomic.AtomicReference;

import static org.junit.jupiter.api.Assertions.*;

class AuthClientTest {
    private HttpServer server;
    private AtomicReference<String> receivedCookie;

    @BeforeEach void startServer() throws IOException {
        receivedCookie = new AtomicReference<>();
        server = HttpServer.create(new InetSocketAddress("127.0.0.1", 0), 0);
        server.createContext("/v1/sign-in", exchange -> {
            exchange.getRequestBody().readAllBytes();
            exchange.getResponseHeaders().add("Set-Cookie", "session=java-session; Path=/; HttpOnly; SameSite=Lax");
            reply(exchange, 200, "{\"id\":\"u1\"}");
        });
        server.createContext("/v1/me", exchange -> {
            receivedCookie.set(exchange.getRequestHeaders().getFirst("Cookie"));
            if (receivedCookie.get() == null) reply(exchange, 401, "{\"message\":\"authentication required\"}");
            else reply(exchange, 200, "{\"email\":\"user@example.test\"}");
        });
        server.start();
    }

    @AfterEach void stopServer() { server.stop(0); }

    @Test void storesCookieAndUsesItOnNextRequest() {
        AuthClient client = new AuthClient("http://127.0.0.1:" + server.getAddress().getPort());
        client.signIn("user@example.test", "password");
        assertEquals("user@example.test", client.me().path("email").asText());
        assertTrue(receivedCookie.get().contains("session=java-session"));
    }

    @Test void exposesApiStatusForErrors() {
        AuthClient client = new AuthClient("http://127.0.0.1:" + server.getAddress().getPort());
        AuthException error = assertThrows(AuthException.class, client::me);
        assertEquals(401, error.status());
        assertEquals("authentication required", error.getMessage());
    }

    private static void reply(com.sun.net.httpserver.HttpExchange exchange, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().add("Content-Type", "application/json");
        exchange.sendResponseHeaders(status, bytes.length);
        exchange.getResponseBody().write(bytes);
        exchange.close();
    }
}
