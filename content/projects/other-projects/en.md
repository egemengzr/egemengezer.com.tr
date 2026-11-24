## MavCs · a MAVLink library for .NET

*C#, .NET 9, xUnit · Nov 2025 – Jan 2026*

I started this when MAVSDK's multirotor focus caused problems on our fixed-wing UAV and I could not find a good open-source MAVLink library for C#. I wrote MAVLink v1/v2 framing and CRC handling, attribute-based auto-discovery of messages and serializers via `[MavMessage(Id, CrcExtra)]`, and type-safe serializer interfaces; I implemented about 20 messages by hand and added xUnit round-trip tests. ```pipeline
Raw bytes | incoming data
Frame parser | MAVLink v1/v2, CRC + CRC_EXTRA
Serializer discovery | [MavMessage(Id, CrcExtra)]
Typed message | a C# object
```

The next step was a generator that builds message definitions from the MAVLink XML files; the project is paused there for lack of time.

## Smart filament drybox

*ESP32 (ESP-IDF), C, KiCad, BME280 · Mar 2026 – Present*

Electronics for a box that heats 3D-printer filament to remove moisture. I designed the board in KiCad and had it manufactured, and wrote part of the ESP-IDF firmware: it reads temperature and humidity from a BME280 over I2C, drives the MOSFET outputs, and drives a 1.28-inch 240×240 round RGB LCD over SPI, which shows a QR code for reaching the box's interface. ```pipeline
BME280 | temperature + humidity, I2C
ESP32 | ESP-IDF firmware
Outputs | MOSFET outputs, 240×240 SPI LCD
```

About half of the board is assembled; assembly, the enclosure and communication are not finished yet.

## Lidar-based autonomous docking

*C++, Lidar · Sep – Oct 2025*

An algorithm that computes the path an autonomous robot follows to dock precisely at its charging station using Lidar data, with a C++ tokenizer for the TOML-formatted inputs.
