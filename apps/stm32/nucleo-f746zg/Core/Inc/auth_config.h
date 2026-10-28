#ifndef ARBEITER_AUTH_CONFIG_H
#define ARBEITER_AUTH_CONFIG_H

/* Configure for a dedicated test account. Never commit real credentials. */
#if defined(__has_include)
#if __has_include("auth_config.local.h")
#include "auth_config.local.h"
#endif
#endif

#ifndef AUTH_API_HOST
#define AUTH_API_HOST "auth.example.com"
#endif
#ifndef AUTH_API_PORT
#define AUTH_API_PORT "443"
#endif
#ifndef AUTH_EMAIL
#define AUTH_EMAIL ""
#endif
#ifndef AUTH_PASSWORD
#define AUTH_PASSWORD ""
#endif

/* Paste the PEM CA certificate that validates AUTH_API_HOST (include newlines). */
#ifndef AUTH_ROOT_CA_PEM
#define AUTH_ROOT_CA_PEM ""
#endif

#endif
