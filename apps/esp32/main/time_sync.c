#include "time_sync.h"

#include "esp_log.h"
#include "esp_netif_sntp.h"
#include "freertos/FreeRTOS.h"

static const char *TAG = "time_sync";

esp_err_t time_sync_wait_for_network_time(void)
{
    esp_sntp_config_t config = ESP_NETIF_SNTP_DEFAULT_CONFIG("pool.ntp.org");
    esp_err_t err = esp_netif_sntp_init(&config);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "SNTP initialization failed: %s", esp_err_to_name(err));
        return err;
    }

    err = esp_netif_sntp_sync_wait(pdMS_TO_TICKS(20000));
    esp_netif_sntp_deinit();
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "Network time was not synchronized: %s", esp_err_to_name(err));
        return err;
    }
    ESP_LOGI(TAG, "System time synchronized for TLS certificate validation");
    return ESP_OK;
}
