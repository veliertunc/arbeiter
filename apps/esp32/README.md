# ESP32 TFT User Manager

An ESP-IDF application for managing a local user directory on an SD card. The interface runs on an SPI ILI9341 display with an XPT2046 resistive touchscreen using LVGL 9. Users can be created, listed, edited, and deleted. Data is stored in SQLite at `/sdcard/app.db`.

This is a local directory demo. It does not create accounts in the Go authentication service, and it does not store passwords or authentication tokens.

## Hardware and wiring

The SD card, TFT, and touch controller share the ESP32 SPI2 bus. Each uses its own chip-select pin. Default GPIOs are a starting point for common original ESP32 modules; check your module’s schematic and update them in `idf.py menuconfig` before powering the display.

| Signal | Default GPIO |
|---|---:|
| SPI MOSI | 23 |
| SPI MISO | 19 |
| SPI SCLK | 18 |
| SD card CS | 5 |
| ILI9341 CS | 15 |
| ILI9341 D/C | 2 |
| ILI9341 reset | 4 |
| TFT backlight | 32 |
| XPT2046 CS | 33 |
| XPT2046 PENIRQ | 34 |

Connect all modules to 3.3 V logic and common ground. Do not let two SPI devices share the same CS. If touch is mirrored, offset, or rotated, adjust the swap/mirror settings in menuconfig; resistive panels may need controller-specific calibration. Set the IRQ pin to `-1` to poll the touch controller over SPI.

Use a FAT-formatted card. The app will not format a card after a mount failure. Keep the card inserted and powered while the app is running; do not remove it during a database operation.

## Build and flash

Requirements: ESP-IDF 5.5.4 or later, Component Manager, and an ESP32 target with enough flash for the firmware. An 8 MB module is recommended because SQLite and the graphics stack add significant firmware size.

From this directory in an ESP-IDF terminal:

```sh
idf.py set-target esp32
idf.py menuconfig
```

Under **Arbeiter ESP32 User Admin**, confirm the SPI and TFT/touch pins for your hardware. Then build and flash:

```sh
idf.py build
idf.py -p PORT flash monitor
```

The project manifest pins LVGL 9.5, Espressif’s LVGL port and ILI9341 driver, the XPT2046 touch driver, and an ESP-IDF SQLite component. The Component Manager resolves these into the ignored `managed_components/` directory.

## Data and modules

- `sd_card.c`: mounts FAT over SPI2 without auto-formatting and reserves a transfer buffer for the TFT draw chunks.
- `display/tft_display.c`: initializes the ILI9341, XPT2046, and LVGL 9 port on the shared SPI bus.
- `users/user_repository.c`: SQLite schema and parameterized CRUD operations against `/sdcard/app.db`.
- `users/user_service.c`: name, email, and role validation.
- `ui/user_screen.c`: LVGL list, paging, create/edit form, and delete confirmation.
- `app_main.c`: startup and component coordination.

The `users` table stores an integer ID, display name, unique case-insensitive email, role (`member`, `admin`, or `owner`), and timestamps. It never stores passwords. SQLite uses rollback journaling and full synchronous writes for the FAT volume; the database is serialized behind a mutex.

## Operational notes

- Copy `/sdcard/app.db` regularly while the card is not being used, and use a known-good card. This demo does not include database backup/restore or recovery UI.
- Keep the firmware partition large enough for the selected ESP-IDF, LVGL, and SQLite configuration. Boards with a small default OTA partition may need a custom partition table; changing to a single large app partition removes OTA rollback capability.
- User records are local to this card and are not synchronized with the auth API.
- The XPT2046 driver’s coordinate mapping is panel-dependent. Adjust axis options and controller calibration values if buttons do not line up with touches.

## Dependency notes

LVGL 9 and the Espressif display/touch components are managed ESP-IDF dependencies. The SQLite component is pinned to a registry release that advertises ESP-IDF 5.5.4/6.1 support; build it with your selected target and inspect the resolved lockfile before adopting it in a deployed product.
