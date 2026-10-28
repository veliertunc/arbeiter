#include "tft_display.h"

#include "driver/gpio.h"
#include "driver/spi_master.h"
#include "esp_check.h"
#include "esp_log.h"
#include "esp_lcd_ili9341.h"
#include "esp_lcd_panel_io.h"
#include "esp_lcd_panel_ops.h"
#include "esp_lcd_touch.h"
#include "esp_lcd_touch_xpt2046.h"
#include "esp_lvgl_port.h"
#include "lvgl.h"

#define LCD_H_RES 240
#define LCD_V_RES 320
#define LCD_DRAW_BUFFER_LINES 20

static const char *TAG = "tft_display";
static esp_lcd_panel_handle_t s_panel;
static esp_lcd_panel_io_handle_t s_panel_io;
static esp_lcd_touch_handle_t s_touch;
static lv_display_t *s_lv_display;

static esp_err_t initialize_backlight(void)
{
#if CONFIG_APP_LCD_BACKLIGHT_GPIO >= 0
    const gpio_config_t config = {
        .pin_bit_mask = 1ULL << CONFIG_APP_LCD_BACKLIGHT_GPIO,
        .mode = GPIO_MODE_OUTPUT,
        .pull_up_en = GPIO_PULLUP_DISABLE,
        .pull_down_en = GPIO_PULLDOWN_DISABLE,
        .intr_type = GPIO_INTR_DISABLE,
    };
    ESP_RETURN_ON_ERROR(gpio_config(&config), TAG, "Backlight GPIO setup failed");
#if CONFIG_APP_LCD_BACKLIGHT_ACTIVE_LOW
    ESP_RETURN_ON_ERROR(gpio_set_level(CONFIG_APP_LCD_BACKLIGHT_GPIO, 0), TAG, "Backlight enable failed");
#else
    ESP_RETURN_ON_ERROR(gpio_set_level(CONFIG_APP_LCD_BACKLIGHT_GPIO, 1), TAG, "Backlight enable failed");
#endif
#endif
    return ESP_OK;
}

static esp_err_t initialize_panel(void)
{
    esp_lcd_panel_io_spi_config_t io_config =
        ILI9341_PANEL_IO_SPI_CONFIG(CONFIG_APP_LCD_CS_GPIO, CONFIG_APP_LCD_DC_GPIO, NULL, NULL);
    io_config.pclk_hz = 40 * 1000 * 1000;
    ESP_RETURN_ON_ERROR(esp_lcd_new_panel_io_spi((esp_lcd_spi_bus_handle_t)SPI2_HOST,
                                                  &io_config, &s_panel_io),
                        TAG, "ILI9341 panel I/O creation failed");

    const esp_lcd_panel_dev_config_t panel_config = {
        .reset_gpio_num = CONFIG_APP_LCD_RST_GPIO,
        .rgb_endian = LCD_RGB_ENDIAN_RGB,
        .bits_per_pixel = 16,
    };
    ESP_RETURN_ON_ERROR(esp_lcd_new_panel_ili9341(s_panel_io, &panel_config, &s_panel),
                        TAG, "ILI9341 driver creation failed");
    ESP_RETURN_ON_ERROR(esp_lcd_panel_reset(s_panel), TAG, "ILI9341 reset failed");
    ESP_RETURN_ON_ERROR(esp_lcd_panel_init(s_panel), TAG, "ILI9341 initialization failed");
    ESP_RETURN_ON_ERROR(esp_lcd_panel_disp_on_off(s_panel, true), TAG, "ILI9341 display enable failed");
    return ESP_OK;
}

static esp_err_t initialize_touch(void)
{
    const esp_lcd_panel_io_spi_config_t io_config =
        ESP_LCD_TOUCH_IO_SPI_XPT2046_CONFIG(CONFIG_APP_TOUCH_CS_GPIO);
    esp_lcd_panel_io_handle_t io = NULL;
    ESP_RETURN_ON_ERROR(esp_lcd_new_panel_io_spi((esp_lcd_spi_bus_handle_t)SPI2_HOST,
                                                  &io_config, &io),
                        TAG, "XPT2046 panel I/O creation failed");

    const esp_lcd_touch_config_t touch_config = {
        .x_max = LCD_H_RES,
        .y_max = LCD_V_RES,
        .rst_gpio_num = -1,
        .int_gpio_num = CONFIG_APP_TOUCH_IRQ_GPIO,
        .flags = {
            .swap_xy = CONFIG_APP_TOUCH_SWAP_XY,
            .mirror_x = CONFIG_APP_TOUCH_MIRROR_X,
            .mirror_y = CONFIG_APP_TOUCH_MIRROR_Y,
        },
    };
    return esp_lcd_touch_new_spi_xpt2046(io, &touch_config, &s_touch);
}

esp_err_t tft_display_init(void)
{
    ESP_RETURN_ON_ERROR(initialize_backlight(), TAG, "Backlight initialization failed");
    ESP_RETURN_ON_ERROR(initialize_panel(), TAG, "Panel initialization failed");
    ESP_RETURN_ON_ERROR(initialize_touch(), TAG, "Touch initialization failed");

    const lvgl_port_cfg_t lvgl_config = ESP_LVGL_PORT_INIT_CONFIG();
    ESP_RETURN_ON_ERROR(lvgl_port_init(&lvgl_config), TAG, "LVGL port initialization failed");

    const lvgl_port_display_cfg_t display_config = {
        .io_handle = s_panel_io,
        .panel_handle = s_panel,
        .buffer_size = LCD_H_RES * LCD_DRAW_BUFFER_LINES,
        .double_buffer = true,
        .hres = LCD_H_RES,
        .vres = LCD_V_RES,
        .monochrome = false,
        .color_format = LV_COLOR_FORMAT_RGB565,
        .rotation = {
            .swap_xy = false,
            .mirror_x = false,
            .mirror_y = false,
        },
        .flags = {
            .buff_dma = true,
            .swap_bytes = true,
        },
    };
    s_lv_display = lvgl_port_add_disp(&display_config);
    if (!s_lv_display) return ESP_ERR_NO_MEM;

    const lvgl_port_touch_cfg_t touch_config = {
        .disp = s_lv_display,
        .handle = s_touch,
    };
    if (!lvgl_port_add_touch(&touch_config)) return ESP_ERR_NO_MEM;

    ESP_LOGI(TAG, "ILI9341 and XPT2046 ready at %dx%d", LCD_H_RES, LCD_V_RES);
    return ESP_OK;
}
