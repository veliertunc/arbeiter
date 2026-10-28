# Arbeiter Auth on NUCLEO-F746ZG

Firmware application for the STM32F746ZG on the NUCLEO-F746ZG board. It uses the RJ45 Ethernet PHY through lwIP's socket API, mbedTLS for verified HTTPS, direct CMSIS GPIO drivers, and FreeRTOS tasks. Board support and the FreeRTOS/lwIP/mbedTLS libraries are supplied by STM32CubeF7/CubeIDE rather than copied into this repository.

## Current scope

The auth task waits for Ethernet link, an assigned IPv4 address, and a valid system clock. It then sends `POST /v1/sign-in` and `GET /v1/me` through the RJ45 interface. The session cookie is held in RAM, profile responses are not logged, and TLS validates both the configured CA and hostname. Pressing B1 cycles the status LEDs; green indicates the last auth check succeeded and red indicates a configuration/network/request failure.

The repository's current C SDK (`sdk/cpp/include/arbeiter_auth/c_api.h`) is a C ABI over a C++ implementation that requires libcurl and nlohmann/json; those desktop dependencies cannot be linked into STM32F746 firmware. This app therefore implements the two REST calls it needs in a firmware-native C transport over lwIP sockets and mbedTLS. Never send credentials over plain HTTP.

## Build setup

1. Install STM32CubeIDE and the STM32CubeF7 package for the STM32F7 series.
2. Create/import a CubeIDE project for **NUCLEO-F746ZG** (STM32F746ZGTx), enable FreeRTOS (CMSIS-RTOS v2), Ethernet in RMII mode, LwIP with DHCP and socket/netconn support, DNS, and mbedTLS client/X.509 support. Configure the system clock using the board's HSE clock source.
3. Add this directory's `Core/Inc` and `Core/Src` to the project. Keep CubeMX-generated startup, `SystemClock_Config`, HAL initialization, and FreeRTOS kernel sources in the CubeIDE project.
4. Enable `LWIP_SNTP` and `SNTP_SERVER_DNS` in `lwipopts.h`. This app starts lwIP SNTP against `pool.ntp.org` once the link has an IPv4 address. Configure the SNTP time-update hook to update the C library clock (or provide a valid RTC); the app waits until `time()` is valid because mbedTLS validates certificate dates. Enable the STM32 hardware RNG; the app registers it as mbedTLS's strong entropy source and fails closed if the RNG reports an error. TLS is restricted to TLS 1.2 or newer.
5. Create an ignored `Core/Inc/auth_config.local.h` (the path is in `.gitignore`) with `AUTH_API_HOST`, `AUTH_API_PORT`, `AUTH_EMAIL`, `AUTH_PASSWORD`, and `AUTH_ROOT_CA_PEM` string-literal macros. Configure the API hostname without a scheme/path, and put the trusted CA PEM in `AUTH_ROOT_CA_PEM`. Use a dedicated demo account; credentials are compiled into firmware flash.
6. Ensure the generated project defines `USE_HAL_DRIVER` and `STM32F746xx`; include STM32F7 CMSIS, FreeRTOS, lwIP sockets, and mbedTLS headers/libraries.
7. Call `arbeiter_app_start()` after LwIP initialization and before `vTaskStartScheduler()`, or from an already-running FreeRTOS task. Ensure FreeRTOS dynamic allocation and `configSUPPORT_DYNAMIC_ALLOCATION` are enabled. Build and flash using CubeIDE/ST-LINK.

The on-board Ethernet PHY uses RMII pins; avoid assigning those pins to other peripherals. Connect the RJ45 jack to a network with DHCP and DNS access to the API host. The default board LEDs are LD1 PB0, LD2 PB7, and LD3 PB14; B1 is PC13. Check the board solder-bridge configuration if LD1 is routed to PA5 instead.

## Layout

- `Core/Inc/board_gpio.h`, `Core/Src/board_gpio.c`: direct CMSIS register setup and GPIO access for B1 and the three user LEDs.
- `Core/Inc/arbeiter_app.h`, `Core/Src/arbeiter_app.c`: FreeRTOS event queue, button/status handling, and the auth task.
- `Core/Inc/auth_client.h`, `Core/Src/auth_client.c`: lwIP DNS/TCP sockets, mbedTLS verification, and sign-in/current-user API calls.
- `Core/Inc/auth_config.h`: safe empty defaults with an optional ignored local config override.
- `stm32cube.ioc`: CubeMX project settings to use as a starting point when generating the board project.

This is a board demo, not a production-ready identity device. The `.ioc` file is a CubeMX starting point; CubeMX must generate the matching board startup and middleware sources before this application module can be built. Credentials are compiled into the image, so use only a dedicated test account. Production deployments need secure provisioning, protected storage, an audited certificate/clock update path, watchdog policy, and hardware-in-loop validation.
