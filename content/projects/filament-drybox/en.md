## Goal

3D printer filament absorbs moisture from the air, and damp filament prints badly. This box dries the filament by heating it.

## Hardware

I designed the board in KiCad and had it made: an ESP32, two BME280s (inside the box and in the electronics bay), MOSFET outputs for a PTC heater and a fan, and a 1.28-inch 240×240 round SPI LCD.

## Firmware

On ESP-IDF, every part is its own component: sensor, fan, heater, display, Wi-Fi manager, DNS server and web server.

```pipeline
First boot | no saved network
Setup network | Wi-Fi QR code on screen
Captive portal | scan networks, enter password
Save to NVS | restart
Dashboard | readings, JSON API
```

- **Setup:** With no saved network, the box opens its own access point and shows a QR code to join it. A DNS server points every domain at the box, so the setup page opens by itself when a phone connects. The page lists nearby networks; the chosen one is written to NVS and the box restarts.
- **Normal mode:** The box joins the home network, and the same web server now serves a temperature and humidity dashboard and a JSON API.
- **Drivers:** The fan is driven by LEDC PWM at 2.5 kHz with 10-bit resolution, the PTC heater through a GPIO. The sensors are read over I2C; the display is drawn with LVGL.

## Status

The firmware components are ready. About half of the board is assembled; the hardware is waiting because I have not been able to budget for the remaining parts and the enclosure. The control loop that drives the heater from humidity and temperature will be wired up once the board is complete.
