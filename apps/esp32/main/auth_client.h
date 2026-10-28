#pragma once

#include "esp_err.h"

esp_err_t auth_client_sign_in(const char *email, const char *password);
esp_err_t auth_client_get_current_user(void);
