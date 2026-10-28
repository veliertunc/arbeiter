package com.arbeiter.auth;

public final class AuthException extends RuntimeException {
    private final int status;

    public AuthException(int status, String message) {
        super(message == null || message.isBlank() ? "Authentication API request failed" : message);
        this.status = status;
    }

    public int status() { return status; }
}
