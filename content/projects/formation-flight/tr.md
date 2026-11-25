## Amaç

İki sabit kanatlı uçağın formasyon uçması: lider kendi görev rotasını uçarken takipçi onun 45 m arkasında ve 5 m yukarısında kalıyor. İki kişilik bir işti; 96 commit'in 80'i benim.

## Nasıl çalışıyor

```pipeline
Telemetri | iki uçak, iki ayrı thread
Anlık kopya | tutarlı veriyle hesap
Hedef nokta | liderin 45 m arkası, 5 m üstü
Yön ve hız | PID ve hız profili
GUIDED komutları | hız, yön, irtifa
```

- **Modlar:** Lider AUTO'da görevini uçuyor, takipçi GUIDED'da. Takipçiye ArduPlane'in GUIDED hız, yön ve irtifa komutları gidiyor.
- **Yön:** Takipçinin hedef noktaya olan yanal sapması bir PID'e giriyor, çıktı en fazla 45°'lik yön düzeltmesi.
- **Hız:** Liderin yer hızı, rüzgâr vektörü ve basınç/sıcaklıktan hesaplanan EAS→TAS oranıyla hava hızına çevriliyor. Üstüne mesafeye göre bir fark ekleniyor: geride kalınca hızlan, yaklaşınca fren. Sonuç 14–23 m/s aralığında kalıyor.
- **İletişim kopunca:** Kayıtlı yön ve irtifada uçmaya devam ediyor. Süre dolarsa takipçi RTL ile eve dönüyor.
- **Sahada değişiklik:** Lider/takipçi rolleri arayüzden tek komutla değiştirilebiliyor. Parametreler her döngüde dosyadan okunduğu için kazançlar programı yeniden başlatmadan ayarlanabiliyor.

![Takipçinin hız profili](speed-tr.svg "Takipçinin hedef hızına eklenen fark, lidere uzaklığa göre (config.param değerleriyle, hız PID'inden önce).")

## Sonuç

Simülasyonda iki uçak formasyonu tuttu. Sahada ise çok rüzgârlı bir günde takipçi formasyona geri dönemedi, yerinde bekleyip durdu. Sorunun rüzgâr mı, yoksa uçaklar arası haberleşme mi olduğunu o uçuşta ayıramadık.
