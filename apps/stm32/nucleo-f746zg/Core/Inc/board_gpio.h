#ifndef ARBEITER_BOARD_GPIO_H
#define ARBEITER_BOARD_GPIO_H

#include <stdbool.h>
#include <stdint.h>

void board_gpio_init(void);
bool board_user_button_pressed(void);
void board_status_led_set(unsigned int led, bool on);
void board_status_led_toggle(unsigned int led);

#endif
