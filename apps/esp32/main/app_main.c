#include "auth_client.h"
#include "sd_card.h"
#include "time_sync.h"
#include "wifi_station.h"

#include "esp_err.h"
#include "esp_log.h"
#include "nvs_flash.h"

static const char *TAG = "arbeiter_esp32";

static esp_err_t initialize_nvs(void)
{
    esp_err_t err = nvs_flash_init();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "NVS initialization failed: %s", esp_err_to_name(err));
    }
    return err;
}

void app_main(void)
{
    ESP_ERROR_CHECK(initialize_nvs());

    esp_err_t err = sd_card_mount();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Storage is unavailable; stopping the demo");
        return;
    }
    sd_card_append_event("sd_mounted", 0);

    err = wifi_station_connect();
    sd_card_append_event("wifi_connect", err == ESP_OK ? 0 : (int)err);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Wi-Fi setup failed; configure the network and reboot");
        return;
    }

    err = time_sync_wait_for_network_time();
    sd_card_append_event("time_sync", err == ESP_OK ? 0 : (int)err);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "TLS requests require valid network time");
        return;
    }

    if (CONFIG_APP_AUTH_EMAIL[0] == '\0' || CONFIG_APP_AUTH_PASSWORD[0] == '\0') {
        ESP_LOGW(TAG, "Set demo credentials with idf.py menuconfig to run sign-in");
        sd_card_append_event("credentials_missing", 0);
        return;
    }

    err = auth_client_sign_in(CONFIG_APP_AUTH_EMAIL, CONFIG_APP_AUTH_PASSWORD);
    sd_card_append_event("sign_in", err == ESP_OK ? 200 : (int)err);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Sign-in failed; see the HTTP status in the serial log");
        return;
    }

    err = auth_client_get_current_user();
    sd_card_append_event("current_user", err == ESP_OK ? 200 : (int)err);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "The API did not accept the new session");
        return;
    }

    ESP_LOGI(TAG, "ESP32 auth demo completed; non-sensitive events are on the SD card");
}
