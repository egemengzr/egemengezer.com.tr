## Goal

3D printer filament absorbs moisture from the air, and damp filament prints badly. This box dries the filament by heating it.

## Hardware and firmware

```pipeline
BME280 | temperature + humidity, I2C
ESP32 | ESP-IDF firmware
Outputs | MOSFET outputs, 240×240 SPI LCD
```

- Designed the board in KiCad and had it manufactured.
- Wrote part of the ESP-IDF firmware: it reads temperature and humidity from a BME280 over I2C, drives the MOSFET outputs and drives a 1.28-inch 240×240 round RGB LCD over SPI.
- The display shows a QR code that leads to the box's interface.

## Status

About half of the board is assembled. Assembly, the enclosure and communication are not finished yet.
