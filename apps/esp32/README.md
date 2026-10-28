# Arbeiter ESP32 Auth Demo

An ESP-IDF application for ESP32 that mounts a FAT SD card over SPI, joins Wi-Fi, synchronizes network time for TLS validation, signs in to the Arbeiter Auth REST API, and checks the current session with `GET /v1/me`.

The application is split by responsibility:

- `sd_card.c`: SPI bus setup, SDSPI/FAT mount and unmount, and a small non-sensitive event log.
- `wifi_station.c`: station-mode Wi-Fi setup and connection wait.
- `auth_client.c`: HTTPS sign-in and current-user requests using `esp_http_client`.
- `app_main.c`: startup and flow coordination.

## Requirements

- ESP-IDF 6.1 or later, with the ESP32 target installed.
- An ESP32 board and a 3.3 V-compatible microSD breakout wired for SPI.
- A FAT-formatted SD card.
- A reachable Arbeiter Auth API with a trusted HTTPS certificate.
- A dedicated test account.

The default GPIOs match common original ESP32 SPI wiring: MOSI 23, MISO 19, SCLK 18, and CS 5. Confirm these against your board and change them in menuconfig when needed. The GPIO matrix permits other valid pins. Do not connect a 5 V SD module directly to ESP32 GPIO.

## Configure and run

From this directory in an ESP-IDF terminal:

```sh
idf.py set-target esp32
idf.py menuconfig
```

Open **Arbeiter ESP32 Auth Demo** and configure the Wi-Fi network, HTTPS API base URL, dedicated test account, and SD SPI pins. Then build and flash:

```sh
idf.py build
idf.py -p PORT flash monitor
```

The device needs outbound DNS, HTTPS, and NTP access. It synchronizes the clock through `pool.ntp.org` before making TLS requests so certificate validity checks can succeed.

The app mounts the card at `/sdcard` without formatting it. It appends uptime, event name, and status code to `/sdcard/arbeiter-auth.log`. It never writes the password, profile response, or session cookie to the card. The session cookie is held in RAM and is lost when the device restarts.

## Security notes

- Keep `sdkconfig` local; it contains Wi-Fi and demo account settings and is ignored by Git. Those settings are compiled into firmware, so this sample is for development with a dedicated test account.
- Use a trusted HTTPS endpoint. TLS server verification uses the ESP x509 certificate bundle; add a private CA to the bundle configuration if the API uses an internal certificate.
- `APP_ALLOW_INSECURE_HTTP` is off by default. Only enable it on an isolated development network; it sends the password and session cookie without encryption.
- Production devices need secure provisioning, flash encryption, secure boot, and a credential lifecycle appropriate to the product. This demo does not provision credentials or persist sessions across resets.
- SD event logging contains no user identifiers or secrets. Protect the removable card as operational data.

## Implementation references

SD initialization uses `spi_bus_initialize` before `esp_vfs_fat_sdspi_mount`; the filesystem helper is configured not to format a card after mount failure. ESP-IDF describes SDSPI as using the SPI master driver and the FAT VFS helper for the mounted card. The HTTP client verifies HTTPS through the configured ESP certificate bundle.
