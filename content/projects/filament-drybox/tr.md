## Amaç

3B yazıcı filamenti havadaki nemi çekiyor ve nemli filament kötü baskı veriyor. Bu kutu filamenti ısıtarak nemini alıyor.

## Donanım ve firmware

```pipeline
BME280 | sıcaklık + nem, I2C
ESP32 | ESP-IDF firmware
Çıkışlar | MOSFET çıkışları, 240×240 SPI LCD
```

- Kartı KiCad'de tasarlayıp ürettirdim.
- ESP-IDF firmware'inin bir kısmını yazdım: BME280'den I2C ile sıcaklık ve nem okuyor, MOSFET çıkışlarını sürüyor ve 1.28 inçlik 240×240 yuvarlak RGB LCD'yi SPI ile sürüyor.
- Ekran, kutunun arayüzüne ulaşmak için bir QR kod gösteriyor.

## Durum

Kartın yaklaşık yarısı monte edildi. Montaj, kutu ve haberleşme henüz bitmedi.
