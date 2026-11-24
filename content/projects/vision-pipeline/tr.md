## Amaç

Hedef tespitini ve takibini araç üzerinde gerçek zamanlı çalıştırmak, QR kodlu yer hedefini her ışıkta okunur tutmak ve bu sezon tüm araçları ortak bir ağa bağlamak.

## Görüntü işleme hattı

```pipeline
Kamera | sensör + lens + CPL filtre
YOLO tespit | TensorRT, FP16
Tracker | TensorRT, FP16, nesne yeniden kullanımı
Hedef verisi | otonomi yazılımına
```

## Performans (Haziran – Eylül 2026)

- YOLO modelini ve kullandığımız hazır tracker'ı TensorRT'ye çevirdim, tüm sistemi FP16'ya taşıdım.
- Tracker kodunda her karede yeniden kurulan ve çok zaman harcayan bir nesne buldum. Bunu bir kez oluşturulan, her karede temizlenip yeniden doldurulan tek bir örnekle değiştirdim. Kare başına bellek ayırma ve kurulum maliyeti ortadan kalktı, periyodik olarak 40 FPS'e düşmeler bitti.

```bars max=60 Kareden kareye süre time.perf_counter ile ölçüldü. Kamera 60 FPS, yani tavan yaklaşık 60 FPS.
Başlangıç (TensorRT yok) | 13 | 12–14 FPS
YOLO TensorRT'de (FP32) | 25 | ~25 FPS
YOLO + tracker TensorRT'de (FP32) | 37.5 | 35–40 FPS
İkisi de FP16 | 55 | 50–60, ~40'a düşüyor
+ nesne yeniden kullanımı | 57.5 | kararlı ~55–60 FPS | *
```

## Kamera ve optik (Haziran – Eylül 2026)

- Kamera sensörünü, lensi ve filtreyi seçtim.
- Yarışmanın QR kodlu hedefi yerde bir branda üzerinde duruyor ve öğle saatlerinde güneşi yansıtıyor. Bu birçok takım için sorun oldu. Araştırdıktan sonra dairesel polarize (CPL) filtre taktım: metal olmayan yüzeylerden yansıyan ışık büyük ölçüde polarize olduğu için filtre parlamayı bastırıyor. Bizim takım bu sorunu hiç yaşamadı.

## Bu sezon: batman-adv mesh ağı (devam ediyor)

Takımın tüm hava ve kara araçları için tek bir mesh ağ kuruyorum. batman-adv, Linux kernel'inde çalışan bir katman 2 mesh protokolü. Araçlar tek bir ağı paylaşıyor ve birbirlerinin trafiğini aktarabiliyor.
