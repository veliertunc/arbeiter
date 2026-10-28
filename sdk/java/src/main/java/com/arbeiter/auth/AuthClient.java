package com.arbeiter.auth;

import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.fasterxml.jackson.databind.node.ObjectNode;

import java.io.IOException;
import java.net.CookieManager;
import java.net.CookiePolicy;
import java.net.URI;
import java.net.URLEncoder;
import java.net.http.HttpClient;
import java.net.http.HttpRequest;
import java.net.http.HttpResponse;
import java.nio.charset.StandardCharsets;
import java.time.Duration;

/** HTTP client for the cookie-session Arbeiter authentication API. */
public final class AuthClient {
    private final String baseUrl;
    private final ObjectMapper json = new ObjectMapper();
    private final HttpClient http;

    public AuthClient(String baseUrl) {
        if (baseUrl == null || baseUrl.isBlank()) throw new IllegalArgumentException("baseUrl is required");
        this.baseUrl = baseUrl.replaceAll("/+$", "");
        CookieManager cookies = new CookieManager(null, CookiePolicy.ACCEPT_ORIGINAL_SERVER);
        this.http = HttpClient.newBuilder().cookieHandler(cookies).connectTimeout(Duration.ofSeconds(10)).build();
    }

    private JsonNode request(String method, String path, JsonNode body) {
        HttpRequest.Builder builder = HttpRequest.newBuilder(URI.create(baseUrl + path))
                .timeout(Duration.ofSeconds(20)).header("Accept", "application/json");
        if (body != null) builder.header("Content-Type", "application/json");
        HttpRequest.BodyPublisher publisher;
        try { publisher = body == null ? HttpRequest.BodyPublishers.noBody() : HttpRequest.BodyPublishers.ofString(json.writeValueAsString(body)); }
        catch (IOException e) { throw new IllegalStateException("Could not encode request JSON", e); }
        HttpResponse<String> response;
        try { response = http.send(builder.method(method, publisher).build(), HttpResponse.BodyHandlers.ofString()); }
        catch (IOException e) { throw new IllegalStateException("Authentication API request failed", e); }
        catch (InterruptedException e) { Thread.currentThread().interrupt(); throw new IllegalStateException("Authentication API request interrupted", e); }
        if (response.statusCode() == 204) return null;
        JsonNode payload;
        try { payload = response.body().isBlank() ? null : json.readTree(response.body()); }
        catch (IOException e) { payload = json.createObjectNode().put("message", response.body()); }
        if (response.statusCode() < 200 || response.statusCode() >= 300) {
            String message = payload == null ? "Authentication API request failed" : payload.path("message").asText("Authentication API request failed");
            throw new AuthException(response.statusCode(), message);
        }
        return payload;
    }

    private ObjectNode object() { return json.createObjectNode(); }
    public JsonNode signUp(String email, String password, String name) { ObjectNode b=object().put("email",email).put("password",password); if(name!=null)b.put("name",name); return request("POST","/v1/sign-up",b); }
    public JsonNode signIn(String email,String password) { return request("POST","/v1/sign-in",object().put("email",email).put("password",password)); }
    public JsonNode signOut() { return request("POST","/v1/sign-out",object()); }
    public JsonNode me() { return request("GET","/v1/me",null); }
    public JsonNode updateMe(String name) { return request("PATCH","/v1/me",object().put("name",name)); }
    public JsonNode deleteAccount(String password) { return request("DELETE","/v1/me",object().put("password",password)); }
    public JsonNode listSessions() { return request("GET","/v1/sessions",null); }
    public JsonNode revokeSession(String id) { return request("DELETE","/v1/sessions/"+segment(id),null); }
    public JsonNode requestEmailVerification() { return request("POST","/v1/email/verification",object()); }
    public JsonNode verifyEmail(String token) { return request("POST","/v1/email/verification/consume",object().put("token",token)); }
    public JsonNode requestPasswordReset(String email) { return request("POST","/v1/password/reset",object().put("email",email)); }
    public JsonNode resetPassword(String token,String password) { return request("POST","/v1/password/reset/consume",object().put("token",token).put("password",password)); }
    public JsonNode listOrganizations() { return request("GET","/v1/organizations",null); }
    public JsonNode createOrganization(String name,String slug) { return request("POST","/v1/organizations",object().put("name",name).put("slug",slug)); }
    public JsonNode deleteOrganization(String id) { return request("DELETE","/v1/organizations/"+segment(id),null); }
    public JsonNode listMembers(String organizationId) { return request("GET","/v1/organizations/"+segment(organizationId)+"/members",null); }
    public JsonNode upsertMember(String organizationId,String userId,String role) { return request("PUT","/v1/organizations/"+segment(organizationId)+"/members/"+segment(userId),object().put("role",role)); }
    public JsonNode deleteMember(String organizationId,String userId) { return request("DELETE","/v1/organizations/"+segment(organizationId)+"/members/"+segment(userId),null); }

    private static String segment(String value) { return URLEncoder.encode(value, StandardCharsets.UTF_8).replace("+", "%20"); }
}
