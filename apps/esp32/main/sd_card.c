#include "sd_card.h"

#include <stdbool.h>
#include <stdio.h>
#include <stdint.h>

#include "driver/sdspi_host.h"
#include "driver/spi_master.h"
#include "esp_log.h"
#include "esp_vfs_fat.h"
#include "sdmmc_cmd.h"

#define SD_MOUNT_POINT "/sdcard"

static const char *TAG = "sd_card";
static sdmmc_card_t *s_card;
static bool s_bus_ready;

esp_err_t sd_card_mount(void)
{
    const spi_host_device_t host_id = SPI2_HOST;
    const spi_bus_config_t bus_config = {
        .mosi_io_num = CONFIG_APP_LCD_MOSI_GPIO,
        .miso_io_num = CONFIG_APP_LCD_MISO_GPIO,
        .sclk_io_num = CONFIG_APP_LCD_SCLK_GPIO,
        .quadwp_io_num = -1,
        .quadhd_io_num = -1,
        /* SPI2 is shared by SD, the ILI9341, and XPT2046. */
        .max_transfer_sz = 240 * 20 * sizeof(uint16_t),
    };

    esp_err_t err = spi_bus_initialize(host_id, &bus_config, SDSPI_DEFAULT_DMA);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "SPI bus initialization failed: %s", esp_err_to_name(err));
        return err;
    }
    s_bus_ready = true;

    sdmmc_host_t host = SDSPI_HOST_DEFAULT();
    host.slot = host_id;

    sdspi_device_config_t slot_config = SDSPI_DEVICE_CONFIG_DEFAULT();
    slot_config.host_id = host_id;
    slot_config.gpio_cs = CONFIG_APP_SD_CS_GPIO;

    const esp_vfs_fat_mount_config_t mount_config = {
        .format_if_mount_failed = false,
        .max_files = 4,
        .allocation_unit_size = 16 * 1024,
    };

    err = esp_vfs_fat_sdspi_mount(SD_MOUNT_POINT, &host, &slot_config, &mount_config, &s_card);
    if (err != ESP_OK) {
        ESP_LOGE(TAG, "SD card mount failed: %s", esp_err_to_name(err));
        sd_card_unmount();
        return err;
    }

    ESP_LOGI(TAG, "SD card mounted at %s", SD_MOUNT_POINT);
    sdmmc_card_print_info(stdout, s_card);
    return ESP_OK;
}

void sd_card_unmount(void)
{
    if (s_card) {
        esp_vfs_fat_sdcard_unmount(SD_MOUNT_POINT, s_card);
        s_card = NULL;
    }
    if (s_bus_ready) {
        spi_bus_free(SPI2_HOST);
        s_bus_ready = false;
    }
}
