#include "display/tft_display.h"
#include "sd_card.h"
#include "ui/user_screen.h"
#include "users/user_repository.h"

#include "esp_err.h"
#include "esp_lvgl_port.h"
#include "esp_log.h"

static const char *TAG = "arbeiter_user_admin";

void app_main(void)
{
    esp_err_t err = sd_card_mount();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "SD card mount failed; check the card and SPI pin configuration");
        return;
    }
    err = user_repository_open("/sdcard/app.db");
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Could not open /sdcard/app.db: %s", esp_err_to_name(err));
        sd_card_unmount();
        return;
    }

    err = tft_display_init();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Display/touch initialization failed: %s", esp_err_to_name(err));
        user_repository_close();
        sd_card_unmount();
        return;
    }

    if (!lvgl_port_lock(0)) {
        ESP_LOGE(TAG, "Could not lock LVGL for initial screen setup");
        user_repository_close();
        sd_card_unmount();
        return;
    }
    err = user_screen_init();
    lvgl_port_unlock();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "User screen initialization failed: %s", esp_err_to_name(err));
        user_repository_close();
        sd_card_unmount();
        return;
    }
    ESP_LOGI(TAG, "User manager ready; SQLite database is /sdcard/app.db");
}
