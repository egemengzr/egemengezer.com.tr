## MavCs · .NET için MAVLink kütüphanesi

*C#, .NET 9, xUnit · Kasım 2025 – Ocak 2026*

MAVSDK'nin çok rotorlu araçlara odaklanması sabit kanatlı İHA'mızda sorun çıkarınca ve C# için iyi bir açık kaynak MAVLink kütüphanesi bulamayınca başladım. MAVLink v1/v2 çerçeveleme ve CRC işlemeyi, `[MavMessage(Id, CrcExtra)]` ile attribute tabanlı mesaj ve serializer keşfini, tip güvenli serializer arayüzlerini yazdım. Yaklaşık 20 mesajı elle uyguladım ve xUnit ile gidiş-dönüş testleri ekledim. ```pipeline
Ham baytlar | gelen veri
Çerçeve ayrıştırıcı | MAVLink v1/v2, CRC + CRC_EXTRA
Serializer keşfi | [MavMessage(Id, CrcExtra)]
Tip güvenli mesaj | C# nesnesi
```

Sıradaki adım, mesaj tanımlarını MAVLink XML dosyalarından üreten bir generator'dı. Proje zaman yetersizliğinden orada duraklatıldı.

## Akıllı filament kurutma kutusu

*ESP32 (ESP-IDF), C, KiCad, BME280 · Mart 2026 – Devam*

3B yazıcı filamentini ısıtarak nemini alan bir kutunun elektroniği. Kartı KiCad'de tasarlayıp ürettirdim ve ESP-IDF firmware'inin bir kısmını yazdım: BME280'den I2C ile sıcaklık ve nem okuyor, MOSFET çıkışlarını sürüyor ve 1.28 inçlik 240×240 yuvarlak RGB LCD'yi SPI ile sürüyor. Ekran, kutunun arayüzüne ulaşmak için bir QR kod gösteriyor. ```pipeline
BME280 | sıcaklık + nem, I2C
ESP32 | ESP-IDF firmware
Çıkışlar | MOSFET çıkışları, 240×240 SPI LCD
```

Kartın yaklaşık yarısı monte edildi. Montaj, kutu ve haberleşme henüz bitmedi.

## Lidar ile otonom yanaşma

*C++, Lidar · Eylül – Ekim 2025*

Otonom bir robotun Lidar verisiyle şarj istasyonuna hassas şekilde yanaşması için izleyeceği yolu hesaplayan bir algoritma. TOML formatındaki girdiler için C++ ile bir tokenizer da yazdım.
