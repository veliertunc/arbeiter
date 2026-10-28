#ifndef ARBEITER_AUTH_CLIENT_H
#define ARBEITER_AUTH_CLIENT_H

/* Runs sign-in then GET /v1/me over lwIP sockets protected by mbedTLS. */
int auth_client_sign_in_and_check(void);

#endif
