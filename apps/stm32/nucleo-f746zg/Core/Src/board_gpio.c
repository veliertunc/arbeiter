#include "board_gpio.h"

#include "stm32f7xx.h"

/* NUCLEO-F746ZG: LD1=PB0, LD2=PB7, LD3=PB14 (active high); B1=PC13. */
static const uint32_t s_led_pins[] = { (1UL << 0U), (1UL << 7U), (1UL << 14U) };

void board_gpio_init(void)
{
    RCC->AHB1ENR |= RCC_AHB1ENR_GPIOBEN | RCC_AHB1ENR_GPIOCEN;
    (void)RCC->AHB1ENR;

    GPIOB->MODER &= ~((3U << (0U * 2U)) | (3U << (7U * 2U)) | (3U << (14U * 2U)));
    GPIOB->MODER |=  ((1U << (0U * 2U)) | (1U << (7U * 2U)) | (1U << (14U * 2U)));
    GPIOB->OTYPER &= ~((1UL << 0U) | (1UL << 7U) | (1UL << 14U));
    GPIOB->OSPEEDR &= ~((3U << (0U * 2U)) | (3U << (7U * 2U)) | (3U << (14U * 2U)));
    GPIOB->PUPDR &= ~((3U << (0U * 2U)) | (3U << (7U * 2U)) | (3U << (14U * 2U)));
    GPIOB->BSRR = ((1UL << 0U) | (1UL << 7U) | (1UL << 14U)) << 16U;

    /* B1 is pulled down on the Nucleo board; PC13 reads high when pressed. */
    GPIOC->MODER &= ~(3U << (13U * 2U));
    GPIOC->PUPDR &= ~(3U << (13U * 2U));
}

bool board_user_button_pressed(void)
{
    return (GPIOC->IDR & (1UL << 13U)) != 0U;
}

void board_status_led_set(unsigned int led, bool on)
{
    if (led >= (sizeof(s_led_pins) / sizeof(s_led_pins[0]))) return;
    GPIOB->BSRR = on ? s_led_pins[led] : ((uint32_t)s_led_pins[led] << 16U);
}

void board_status_led_toggle(unsigned int led)
{
    if (led >= (sizeof(s_led_pins) / sizeof(s_led_pins[0]))) return;
    GPIOB->ODR ^= s_led_pins[led];
}
