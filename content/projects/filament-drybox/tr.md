## Amaç

3B yazıcı filamenti havadaki nemi çekiyor ve nemli filament kötü baskı veriyor. Bu kutu filamenti ısıtarak nemini alıyor.

## Donanım

Kartı KiCad'de tasarlayıp ürettirdim: ESP32, iki BME280 (kutunun içi ve elektronik bölmesi), PTC ısıtıcı ve fan için MOSFET çıkışları, 1.28 inçlik 240×240 yuvarlak SPI LCD.

## Firmware

ESP-IDF üzerinde her parça ayrı bir bileşen: sensör, fan, ısıtıcı, ekran, Wi-Fi yöneticisi, DNS sunucusu ve web sunucusu.

```pipeline
İlk açılış | kayıtlı ağ yok
Kurulum ağı | ekranda Wi-Fi QR kodu
Captive portal | ağları tara, şifreyi gir
NVS'ye kaydet | yeniden başla
Panel | ölçümler, JSON API
```

- **Kurulum:** Kayıtlı ağ yoksa kutu kendi erişim noktasını açıyor ve ekranda bağlanmak için QR kod gösteriyor. DNS sunucusu her alan adını kutuya yönlendirdiği için telefon bağlanınca kurulum sayfası kendiliğinden açılıyor. Sayfa çevredeki ağları listeliyor; seçilen ağ NVS'ye yazılıp kutu yeniden başlıyor.
- **Normal mod:** Kutu ev ağına bağlanıyor, aynı web sunucusu bu kez sıcaklık ve nem panelini ve bir JSON API'si sunuyor.
- **Sürücüler:** Fan LEDC ile 2,5 kHz, 10 bit PWM'le, PTC ısıtıcı GPIO ile sürülüyor. Sensörler I2C'den okunuyor; ekran LVGL ile çiziliyor.

## Durum

Firmware bileşenleri hazır. Kartın yaklaşık yarısı monte edildi; kalan parçalar ve kutu için bütçe ayıramadığım için donanım bekliyor. Isıtıcıyı nem ve sıcaklığa göre süren kontrol döngüsü kart tamamlanınca bağlanacak.
