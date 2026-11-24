## Amaç

Otonom bir robotun şarj istasyonuna düşük hata payıyla yanaşması için Lidar taramasından hedef noktayı çıkarmak. Robot tarama anında `(0, 0)` noktasında kabul ediliyor. Program çevredeki düz yüzeyleri buluyor, bunların kesişimlerinden en uygun yanaşma noktasını seçiyor ve robotun o noktaya uzaklığını veriyor.

## Akış

```pipeline
TOML | URL (libcurl) ya da disk
Lexer + parser | elle yazıldı
Nokta bulutu | filtre, (r, θ) → (x, y)
ANTSAC | en fazla 10 doğru parçası
Kesişim seçimi | yakın nokta, açı, uzaklık
```

## TOML okuyucu

Girdi dosyasını okumak için hazır bir TOML kütüphanesi kullanmadım, lexer'ı ve parser'ı C++'ta sıfırdan yazdım.

```pipeline
Dosya | tek seferde belleğe
Lexer | string_view token'lar
Parser | tablo + anahtar → route id
ranges | filtre, (r, θ) → (x, y)
```

- **Kopyasız token'lar.** Dosya bir kez belleğe okunuyor, her token o buffer'ın içini gösteren bir `string_view`. Okuma sırasında metin kopyalanmıyor.
- **Kural tablosu.** Lexer sıradaki karaktere bakıp (koşul, üye fonksiyon işaretçisi) çiftlerinden oluşan bir tabloda doğru üreticiyi seçiyor. Tam sayı, ondalık ve negatif sayılar, tırnaklı string'ler, çıplak anahtarlar, köşeli ve süslü parantezler, yorumlar, LF ve CRLF satır sonları tanınıyor.
- **Route id.** Parser son üç anlamlı token'ı (`anahtar = değer`) tutuyor. Aktif tablo ve anahtar 32 bitlik tek bir sayıda birleşiyor (`tablo << 16 | anahtar`) ve tek bir `switch` ile doğru alana yazılıyor.
- **Akan dizi.** `ranges` dizisi ayrı bir listeye toplanmıyor. Her değer okunduğu anda sensör aralığına göre filtreleniyor, açısıyla birlikte Kartezyen noktaya çevriliyor. Bellek, açı aralığından hesaplanan eleman sayısı kadar baştan ayrılıyor.
- **Eksik alan kontrolü.** Okunan her zorunlu alan bir bit maskesinde işaretleniyor, dosya sonunda eksik olanlar adıyla raporlanıyor.

## ANTSAC

RANSAC örnek noktaları tamamen rastgele seçiyor. Birbirine yakın birkaç duvarın olduğu taramada bu, farklı duvarlardan gelen noktaları tek bir doğruya bağlayabiliyor. ANTSAC'ta her noktanın bir feromon değeri var ve örnekleme bu değerle orantılı. Doğruya uyan noktaların feromonu artıyor, her iterasyonun sonunda tüm feromonlar bir miktar buharlaşıyor. Böylece iyi örnekler zamanla öne çıkıyor.

![RANSAC ve ANTSAC karşılaştırması](ransac-antsac.png "Aynı tarama üzerinde ağırlıklı RANSAC denemesi (solda) ve ANTSAC (sağda)")

Bulunan parçaların kesişimleri üç ölçütle puanlanıyor: kesişime yakın nokta sayısı (~%30), kesişim açısı (~%35) ve kesişim noktasının iki doğruya uzaklığı (~%35). En yüksek puanlı kesişim yanaşma noktası oluyor.

## Benim kısmım

İki kişilik projede dosya indirme ve argüman ayrıştırma, TOML lexer ve parser'ı, ANTSAC kesişim taraması, yakın nokta hesabı ve en iyi kesişimin seçimi bendeydi. Görselleştirmenin büyük kısmını takım arkadaşım yaptı.
