#include "auth_client.h"

#include <strings.h>
#include <stdio.h>
#include <string.h>

#include "cJSON.h"
#include "esp_crt_bundle.h"
#include "esp_http_client.h"
#include "esp_log.h"

#define RESPONSE_CAPACITY 1024
#define SESSION_COOKIE_CAPACITY 192

typedef struct {
    char body[RESPONSE_CAPACITY];
    size_t body_length;
    char session_cookie[SESSION_COOKIE_CAPACITY];
} response_context_t;

static const char *TAG = "auth_client";
static char s_base_url[256];
static char s_session_cookie[SESSION_COOKIE_CAPACITY];

static void initialize_base_url(void)
{
    if (s_base_url[0] != '\0') {
        return;
    }
    snprintf(s_base_url, sizeof(s_base_url), "%s", CONFIG_APP_AUTH_BASE_URL);
    size_t length = strlen(s_base_url);
    while (length > 0 && s_base_url[length - 1] == '/') {
        s_base_url[--length] = '\0';
    }
}

static esp_err_t collect_response(esp_http_client_event_t *event)
{
    response_context_t *context = event->user_data;
    if (!context) {
        return ESP_OK;
    }

    if (event->event_id == HTTP_EVENT_ON_HEADER && event->header_key && event->header_value &&
        strcasecmp(event->header_key, "Set-Cookie") == 0) {
        const char *cookie = event->header_value;
        if (strncmp(cookie, "session=", 8) == 0) {
            const char *end = strchr(cookie, ';');
            size_t length = end ? (size_t)(end - cookie) : strlen(cookie);
            if (length < sizeof(context->session_cookie)) {
                memcpy(context->session_cookie, cookie, length);
                context->session_cookie[length] = '\0';
            }
        }
    } else if (event->event_id == HTTP_EVENT_ON_DATA && event->data && event->data_len > 0) {
        size_t available = sizeof(context->body) - context->body_length - 1;
        size_t copy_length = (size_t)event->data_len;
        if (copy_length > available) {
            copy_length = available;
        }
        memcpy(context->body + context->body_length, event->data, copy_length);
        context->body_length += copy_length;
        context->body[context->body_length] = '\0';
    }
    return ESP_OK;
}

static esp_err_t request(const char *method, const char *path, const char *body,
                         response_context_t *response, int *status_code)
{
    initialize_base_url();
#if !CONFIG_APP_ALLOW_INSECURE_HTTP
    if (strncmp(s_base_url, "https://", 8) != 0) {
        ESP_LOGE(TAG, "HTTPS is required; enable insecure HTTP only for isolated development");
        return ESP_ERR_INVALID_ARG;
    }
#else
    if (strncmp(s_base_url, "https://", 8) != 0 && strncmp(s_base_url, "http://", 7) != 0) {
        return ESP_ERR_INVALID_ARG;
    }
    if (strncmp(s_base_url, "http://", 7) == 0) {
        ESP_LOGW(TAG, "INSECURE DEVELOPMENT MODE: credentials and session are sent without TLS");
    }
#endif
    char url[384];
    int url_length = snprintf(url, sizeof(url), "%s%s", s_base_url, path);
    if (url_length < 0 || (size_t)url_length >= sizeof(url)) {
        return ESP_ERR_INVALID_SIZE;
    }

    esp_http_client_config_t config = {
        .url = url,
        .event_handler = collect_response,
        .user_data = response,
        .timeout_ms = 15000,
        .crt_bundle_attach = esp_crt_bundle_attach,
        .keep_alive_enable = true,
    };
    esp_http_client_handle_t client = esp_http_client_init(&config);
    if (!client) {
        return ESP_ERR_NO_MEM;
    }

    esp_err_t err = ESP_OK;
    esp_http_client_set_method(client, strcmp(method, "POST") == 0 ? HTTP_METHOD_POST : HTTP_METHOD_GET);
    if (body) {
        err = esp_http_client_set_header(client, "Content-Type", "application/json");
        if (err == ESP_OK) {
            err = esp_http_client_set_post_field(client, body, (int)strlen(body));
        }
    }
    if (err == ESP_OK && s_session_cookie[0] != '\0') {
        err = esp_http_client_set_header(client, "Cookie", s_session_cookie);
    }
    if (err == ESP_OK) {
        err = esp_http_client_perform(client);
    }
    if (err == ESP_OK && status_code) {
        *status_code = esp_http_client_get_status_code(client);
    }
    esp_http_client_cleanup(client);
    return err;
}

esp_err_t auth_client_sign_in(const char *email, const char *password)
{
    if (!email || !password || email[0] == '\0' || password[0] == '\0') {
        return ESP_ERR_INVALID_ARG;
    }

    cJSON *payload = cJSON_CreateObject();
    if (!payload ||
        !cJSON_AddStringToObject(payload, "email", email) ||
        !cJSON_AddStringToObject(payload, "password", password)) {
        cJSON_Delete(payload);
        return ESP_ERR_NO_MEM;
    }
    char *body = cJSON_PrintUnformatted(payload);
    cJSON_Delete(payload);
    if (!body) {
        return ESP_ERR_NO_MEM;
    }

    response_context_t response = {0};
    int status_code = 0;
    esp_err_t err = request("POST", "/v1/sign-in", body, &response, &status_code);
    cJSON_free(body);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Sign-in request failed: %s", esp_err_to_name(err));
        return err;
    }
    if (status_code < 200 || status_code >= 300) {
        ESP_LOGW(TAG, "Sign-in rejected with HTTP %d", status_code);
        return ESP_ERR_INVALID_RESPONSE;
    }
    if (response.session_cookie[0] == '\0') {
        ESP_LOGE(TAG, "Sign-in response did not contain a session cookie");
        return ESP_ERR_INVALID_RESPONSE;
    }

    snprintf(s_session_cookie, sizeof(s_session_cookie), "%s", response.session_cookie);
    ESP_LOGI(TAG, "Signed in; session is held in RAM only");
    return ESP_OK;
}

esp_err_t auth_client_get_current_user(void)
{
    if (s_session_cookie[0] == '\0') {
        return ESP_ERR_INVALID_STATE;
    }
    response_context_t response = {0};
    int status_code = 0;
    esp_err_t err = request("GET", "/v1/me", NULL, &response, &status_code);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Current-user request failed: %s", esp_err_to_name(err));
        return err;
    }
    if (status_code != 200) {
        ESP_LOGW(TAG, "Current-user request returned HTTP %d", status_code);
        return ESP_ERR_INVALID_RESPONSE;
    }
    ESP_LOGI(TAG, "Current session is valid (profile response not persisted)");
    return ESP_OK;
}
