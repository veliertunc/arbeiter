#include "auth_client.h"

#include "auth_config.h"
#include "stm32f7xx.h"
#include "lwip/inet.h"
#include "lwip/netdb.h"
#include "lwip/sockets.h"
#include "mbedtls/ctr_drbg.h"
#include "mbedtls/entropy.h"
#include "mbedtls/error.h"
#include "mbedtls/net_sockets.h"
#include "mbedtls/ssl.h"
#include "mbedtls/x509_crt.h"

#include <stdio.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <sys/time.h>

#define HTTP_REQUEST_CAPACITY 2048U
#define HTTP_RESPONSE_CAPACITY 4096U
#define SESSION_COOKIE_CAPACITY 256U

typedef struct {
    mbedtls_ssl_context ssl;
    mbedtls_ssl_config config;
    mbedtls_entropy_context entropy;
    mbedtls_ctr_drbg_context random;
    mbedtls_x509_crt ca;
    int socket_fd;
    bool handshake_complete;
} tls_connection_t;

static char s_session_cookie[SESSION_COOKIE_CAPACITY];
static char s_request[HTTP_REQUEST_CAPACITY];
static char s_response[HTTP_RESPONSE_CAPACITY];

/* mbedTLS strong entropy callback backed by the STM32F746 hardware RNG. */
static int stm32_rng_poll(void *context, unsigned char *output, size_t length, size_t *produced)
{
    (void)context;
    *produced = 0U;
    RCC->AHB2ENR |= RCC_AHB2ENR_RNGEN;
    (void)RCC->AHB2ENR;
    RNG->CR |= RNG_CR_RNGEN;

    while (*produced < length) {
        uint32_t timeout = 1000000U;
        while ((RNG->SR & RNG_SR_DRDY) == 0U) {
            if ((RNG->SR & (RNG_SR_CECS | RNG_SR_SECS)) != 0U || timeout-- == 0U) return -1;
        }
        uint32_t word = RNG->DR;
        size_t amount = length - *produced;
        if (amount > sizeof(word)) amount = sizeof(word);
        memcpy(output + *produced, &word, amount);
        *produced += amount;
    }
    return 0;
}

static int socket_send(void *context, const unsigned char *data, size_t length)
{
    int fd = *(int *)context;
    int sent = lwip_send(fd, data, length, 0);
    return sent > 0 ? sent : MBEDTLS_ERR_NET_SEND_FAILED;
}

static int socket_receive(void *context, unsigned char *data, size_t length)
{
    int fd = *(int *)context;
    int received = lwip_recv(fd, data, length, 0);
    return received >= 0 ? received : MBEDTLS_ERR_NET_RECV_FAILED;
}

static void tls_close(tls_connection_t *connection)
{
    if (connection->handshake_complete) {
        (void)mbedtls_ssl_close_notify(&connection->ssl);
    }
    if (connection->socket_fd >= 0) {
        lwip_close(connection->socket_fd);
        connection->socket_fd = -1;
    }
    mbedtls_x509_crt_free(&connection->ca);
    mbedtls_ctr_drbg_free(&connection->random);
    mbedtls_entropy_free(&connection->entropy);
    mbedtls_ssl_config_free(&connection->config);
    mbedtls_ssl_free(&connection->ssl);
}

static int tls_open(tls_connection_t *connection)
{
    memset(connection, 0, sizeof(*connection));
    connection->socket_fd = -1;
    mbedtls_ssl_init(&connection->ssl);
    mbedtls_ssl_config_init(&connection->config);
    mbedtls_entropy_init(&connection->entropy);
    mbedtls_ctr_drbg_init(&connection->random);
    mbedtls_x509_crt_init(&connection->ca);

    if (AUTH_EMAIL[0] == '\0' || AUTH_PASSWORD[0] == '\0' || AUTH_ROOT_CA_PEM[0] == '\0') {
        return -1;
    }

    int result = mbedtls_entropy_add_source(&connection->entropy, stm32_rng_poll, NULL,
                                            32U, MBEDTLS_ENTROPY_SOURCE_STRONG);
    if (result != 0) return result;
    result = mbedtls_ctr_drbg_seed(&connection->random, mbedtls_entropy_func,
                                   &connection->entropy,
                                   (const unsigned char *)"arbeiter-auth", 13U);
    if (result != 0) return result;
    result = mbedtls_x509_crt_parse(&connection->ca,
                                    (const unsigned char *)AUTH_ROOT_CA_PEM,
                                    sizeof(AUTH_ROOT_CA_PEM));
    if (result != 0) return result;
    result = mbedtls_ssl_config_defaults(&connection->config,
                                         MBEDTLS_SSL_IS_CLIENT,
                                         MBEDTLS_SSL_TRANSPORT_STREAM,
                                         MBEDTLS_SSL_PRESET_DEFAULT);
    if (result != 0) return result;

    mbedtls_ssl_conf_authmode(&connection->config, MBEDTLS_SSL_VERIFY_REQUIRED);
    mbedtls_ssl_conf_min_version(&connection->config,
                                 MBEDTLS_SSL_MAJOR_VERSION_3,
                                 MBEDTLS_SSL_MINOR_VERSION_3); /* TLS 1.2 minimum */
    mbedtls_ssl_conf_ca_chain(&connection->config, &connection->ca, NULL);
    mbedtls_ssl_conf_rng(&connection->config, mbedtls_ctr_drbg_random, &connection->random);
    result = mbedtls_ssl_setup(&connection->ssl, &connection->config);
    if (result != 0) return result;
    result = mbedtls_ssl_set_hostname(&connection->ssl, AUTH_API_HOST);
    if (result != 0) return result;

    struct addrinfo hints;
    memset(&hints, 0, sizeof(hints));
    hints.ai_family = AF_UNSPEC;
    hints.ai_socktype = SOCK_STREAM;
    struct addrinfo *addresses = NULL;
    if (lwip_getaddrinfo(AUTH_API_HOST, AUTH_API_PORT, &hints, &addresses) != 0 || !addresses) {
        return -1;
    }
    for (struct addrinfo *address = addresses; address; address = address->ai_next) {
        connection->socket_fd = lwip_socket(address->ai_family, address->ai_socktype,
                                            address->ai_protocol);
        if (connection->socket_fd >= 0 &&
            lwip_connect(connection->socket_fd, address->ai_addr, address->ai_addrlen) == 0) {
            break;
        }
        if (connection->socket_fd >= 0) lwip_close(connection->socket_fd);
        connection->socket_fd = -1;
    }
    lwip_freeaddrinfo(addresses);
    if (connection->socket_fd < 0) return -1;

    struct timeval timeout = { .tv_sec = 10, .tv_usec = 0 };
    (void)lwip_setsockopt(connection->socket_fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, sizeof(timeout));
    (void)lwip_setsockopt(connection->socket_fd, SOL_SOCKET, SO_SNDTIMEO, &timeout, sizeof(timeout));
    mbedtls_ssl_set_bio(&connection->ssl, &connection->socket_fd,
                        socket_send, socket_receive, NULL);
    do {
        result = mbedtls_ssl_handshake(&connection->ssl);
    } while (result == MBEDTLS_ERR_SSL_WANT_READ || result == MBEDTLS_ERR_SSL_WANT_WRITE);
    if (result != 0 || mbedtls_ssl_get_verify_result(&connection->ssl) != 0) {
        return result != 0 ? result : MBEDTLS_ERR_X509_CERT_VERIFY_FAILED;
    }
    connection->handshake_complete = true;
    return 0;
}

static int json_escape(const char *input, char *output, size_t capacity)
{
    size_t used = 0U;
    for (const unsigned char *p = (const unsigned char *)input; *p; ++p) {
        const char *escape = NULL;
        char unicode_escape[7];
        if (*p == '"') escape = "\\\"";
        else if (*p == '\\') escape = "\\\\";
        else if (*p == '\n') escape = "\\n";
        else if (*p == '\r') escape = "\\r";
        else if (*p == '\t') escape = "\\t";
        else if (*p < 0x20U) {
            (void)snprintf(unicode_escape, sizeof(unicode_escape), "\\u%04x", *p);
            escape = unicode_escape;
        }
        if (escape) {
            size_t n = strlen(escape);
            if (used + n >= capacity) return -1;
            memcpy(output + used, escape, n);
            used += n;
        } else {
            if (used + 1U >= capacity) return -1;
            output[used++] = (char)*p;
        }
    }
    output[used] = '\0';
    return 0;
}

static int https_request(const char *method, const char *path, const char *body)
{
    tls_connection_t *connection = calloc(1U, sizeof(*connection));
    if (!connection) return -1;
    int result = tls_open(connection);
    if (result != 0) {
        tls_close(connection);
        free(connection);
        return result;
    }

    int count = snprintf(s_request, sizeof(s_request),
                         "%s %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n",
                         method, path, AUTH_API_HOST);
    if (count < 0 || (size_t)count >= sizeof(s_request)) result = -1;
    size_t request_length = result == 0 ? (size_t)count : 0U;
    if (result == 0 && s_session_cookie[0] != '\0') {
        count = snprintf(s_request + request_length, sizeof(s_request) - request_length,
                         "Cookie: %s\r\n", s_session_cookie);
        if (count < 0 || (size_t)count >= sizeof(s_request) - request_length) result = -1;
        else request_length += (size_t)count;
    }
    if (result == 0 && body) {
        count = snprintf(s_request + request_length, sizeof(s_request) - request_length,
                         "Content-Type: application/json\r\nContent-Length: %u\r\n\r\n%s",
                         (unsigned int)strlen(body), body);
        if (count < 0 || (size_t)count >= sizeof(s_request) - request_length) result = -1;
        else request_length += (size_t)count;
    } else if (result == 0) {
        count = snprintf(s_request + request_length, sizeof(s_request) - request_length, "\r\n");
        if (count < 0 || (size_t)count >= sizeof(s_request) - request_length) result = -1;
        else request_length += (size_t)count;
    }

    size_t sent = 0U;
    while (result == 0 && sent < request_length) {
        int written = mbedtls_ssl_write(&connection->ssl,
                                        (const unsigned char *)s_request + sent,
                                        request_length - sent);
        if (written <= 0) result = written;
        else sent += (size_t)written;
    }

    size_t received = 0U;
    while (result == 0 && received + 1U < sizeof(s_response)) {
        int n = mbedtls_ssl_read(&connection->ssl,
                                 (unsigned char *)s_response + received,
                                 sizeof(s_response) - received - 1U);
        if (n == 0 || n == MBEDTLS_ERR_SSL_PEER_CLOSE_NOTIFY) break;
        if (n < 0) {
            if (n == MBEDTLS_ERR_SSL_WANT_READ || n == MBEDTLS_ERR_SSL_WANT_WRITE) continue;
            result = n;
            break;
        }
        received += (size_t)n;
        s_response[received] = '\0';
    }
    if (received == 0U) result = result == 0 ? -1 : result;

    int status = 0;
    if (result == 0 && sscanf(s_response, "HTTP/%*u.%*u %d", &status) != 1) result = -1;
    if (result == 0 && status >= 200 && status < 300 && strcmp(method, "POST") == 0) {
        const char *header = s_response;
        while ((header = strstr(header, "\n")) != NULL) {
            ++header;
            if (strncasecmp(header, "Set-Cookie: session=", 20U) == 0) {
                const char *value = header + 12U;
                const char *end = strchr(value, ';');
                if (!end) end = strchr(value, '\r');
                size_t length = end ? (size_t)(end - value) : 0U;
                if (length > 8U && length < sizeof(s_session_cookie)) {
                    memcpy(s_session_cookie, value, length);
                    s_session_cookie[length] = '\0';
                }
                break;
            }
            if (header[0] == '\r' || header[0] == '\n') break;
        }
        if (s_session_cookie[0] == '\0') result = -1;
    } else if (result == 0 && status >= 200 && status < 300 && strcmp(method, "GET") == 0) {
        /* Profile response stays in RAM and is deliberately never logged. */
    } else if (result == 0 && status == 401 && strcmp(method, "GET") == 0) {
        memset(s_session_cookie, 0, sizeof(s_session_cookie));
        result = -1;
    } else if (result == 0) {
        result = -1;
    }

    tls_close(connection);
    free(connection);
    memset(s_request, 0, sizeof(s_request));
    memset(s_response, 0, sizeof(s_response));
    return result;
}

int auth_client_sign_in_and_check(void)
{
    if (s_session_cookie[0] == '\0') {
        char email[320];
        char password[512];
        char body[HTTP_REQUEST_CAPACITY / 2U];
        if (json_escape(AUTH_EMAIL, email, sizeof(email)) != 0 ||
            json_escape(AUTH_PASSWORD, password, sizeof(password)) != 0) return -1;
        int length = snprintf(body, sizeof(body), "{\"email\":\"%s\",\"password\":\"%s\"}", email, password);
        memset(email, 0, sizeof(email));
        memset(password, 0, sizeof(password));
        if (length < 0 || (size_t)length >= sizeof(body)) {
            memset(body, 0, sizeof(body));
            return -1;
        }
        int result = https_request("POST", "/v1/sign-in", body);
        memset(body, 0, sizeof(body));
        if (result != 0) return result;
    }
    return https_request("GET", "/v1/me", NULL);
}
