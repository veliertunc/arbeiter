#pragma once

#include "esp_err.h"

esp_err_t sd_card_mount(void);
esp_err_t sd_card_append_event(const char *event, int status_code);
void sd_card_unmount(void);
