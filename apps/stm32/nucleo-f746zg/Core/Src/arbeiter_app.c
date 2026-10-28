#include "arbeiter_app.h"

#include "auth_client.h"
#include "board_gpio.h"
#include "FreeRTOS.h"
#include "lwip/apps/sntp.h"
#include "lwip/ip4_addr.h"
#include "lwip/netif.h"
#include "lwip/tcpip.h"
#include "queue.h"
#include "task.h"
#include <time.h>

#define APP_EVENT_QUEUE_LENGTH 8U
#define APP_STATUS_TASK_STACK  256U
#define APP_BUTTON_TASK_STACK  256U
#define APP_AUTH_TASK_STACK    768U
#define APP_TASK_PRIORITY      (tskIDLE_PRIORITY + 1U)

typedef struct {
    uint8_t event;
} app_event_t;

enum { APP_EVENT_BUTTON = 1U };

static QueueHandle_t s_events;
static bool s_sntp_started;

static void sntp_start_callback(void *argument)
{
    (void)argument;
    sntp_setoperatingmode(SNTP_OPMODE_POLL);
    sntp_setservername(0U, "pool.ntp.org");
    sntp_init();
}

static void auth_task(void *argument)
{
    (void)argument;
    for (;;) {
        if (!netif_default || !netif_is_up(netif_default) ||
            !netif_is_link_up(netif_default) ||
            ip4_addr_isany_val(*netif_ip4_addr(netif_default))) {
            board_status_led_set(2U, true);
            vTaskDelay(pdMS_TO_TICKS(5000U));
            continue;
        }
        if (!s_sntp_started && time(NULL) < 1735689600) {
            if (tcpip_callback(sntp_start_callback, NULL) == ERR_OK) {
                s_sntp_started = true;
            }
        }
        if (time(NULL) < 1735689600) {
            board_status_led_set(2U, true);
            vTaskDelay(pdMS_TO_TICKS(5000U));
            continue;
        }
        int result = auth_client_sign_in_and_check();
        board_status_led_set(0U, result == 0);
        board_status_led_set(2U, result != 0);
        /* Retry slowly; never print or persist credentials, tokens, or profile data. */
        vTaskDelay(pdMS_TO_TICKS(result == 0 ? 60000U : 15000U));
    }
}

static void status_task(void *argument)
{
    (void)argument;
    app_event_t event;
    unsigned int active_led = 0U;
    board_status_led_set(0U, true);

    for (;;) {
        if (xQueueReceive(s_events, &event, pdMS_TO_TICKS(500U)) == pdPASS &&
            event.event == APP_EVENT_BUTTON) {
            board_status_led_set(active_led, false);
            active_led = (active_led + 1U) % 3U;
            board_status_led_set(active_led, true);
        }
    }
}

static void button_task(void *argument)
{
    (void)argument;
    bool was_pressed = false;
    const TickType_t poll = pdMS_TO_TICKS(20U);

    for (;;) {
        bool pressed = board_user_button_pressed();
        if (pressed && !was_pressed) {
            app_event_t event = { .event = APP_EVENT_BUTTON };
            (void)xQueueSend(s_events, &event, 0U);
        }
        was_pressed = pressed;
        vTaskDelay(poll);
    }
}

void arbeiter_app_start(void)
{
    board_gpio_init();
    s_events = xQueueCreate(APP_EVENT_QUEUE_LENGTH, sizeof(app_event_t));
    if (s_events == NULL) {
        /* Safe visible fault state if the RTOS heap cannot allocate the queue. */
        board_status_led_set(2U, true);
        return;
    }

    if (xTaskCreate(status_task, "auth-status", APP_STATUS_TASK_STACK, NULL,
                    APP_TASK_PRIORITY, NULL) != pdPASS ||
        xTaskCreate(button_task, "button", APP_BUTTON_TASK_STACK, NULL,
                    APP_TASK_PRIORITY, NULL) != pdPASS ||
        xTaskCreate(auth_task, "auth-api", APP_AUTH_TASK_STACK, NULL,
                    APP_TASK_PRIORITY, NULL) != pdPASS) {
        board_status_led_set(2U, true);
    }
}
