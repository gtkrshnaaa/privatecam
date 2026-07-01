# Konvensi Teknis Pengembangan

Dokumen ini mengatur standar praktis pemrograman dan konvensi teknis untuk proyek Private Cam. Seluruh pengembang dan kontribusi wajib mematuhi aturan-aturan berikut.

## 1. Kebersihan Struktur Direktori
* Jaga agar direktori kerja tetap bersih dan bebas dari berkas sementara, tidak terpakai, atau terbengkalai.
* Jangan melakukan commit untuk konfigurasi editor lokal, berkas log, atau hasil *build*. Pastikan berkas-berkas terorganisasi di dalam direktori yang telah ditentukan:
  * `firmware/` – Kode program ESP32-CAM (C++/Arduino).
  * `backend/` – Berkas server Golang.
  * `frontend/` – Berkas dasbor pemantauan UI statis.

## 2. Konvensi Backend (Golang)
* **Deklarasi Variabel**:
  * Sintaks deklarasi variabel pendek (`:=`) dilarang keras untuk digunakan.
  * Gunakan kata kunci `var` standar untuk seluruh deklarasi variabel guna meningkatkan keterbacaan dan memastikan tipe data atau gaya deklarasi tertulis secara eksplisit.
  * Contoh:
    ```go
    // Dilarang
    // x := 10
    
    // Wajib
    var x int = 10
    ```
* Jaga agar fungsi tetap sederhana, terfokus, dan terstruktur dengan baik.

## 3. Konvensi Frontend
* **Gaya Visual (CSS)**:
  * Gunakan CSS murni (*vanilla CSS*) saja.
  * *Framework* CSS pihak ketiga (seperti Tailwind CSS, Bootstrap, Bulma) dilarang keras untuk digunakan.
  * Tulis selektor CSS yang bersih, terorganisasi, dan modular.
* **Komponen Antarmuka (UI Components)**:
  * Pustaka komponen, *UI kit*, atau komponen khusus (seperti shadcn/ui, Radix UI) dilarang keras untuk digunakan.
  * Tulis komponen HTML, CSS, dan JS secara *native* untuk menjaga agar basis kode tetap sederhana, ringan, dan bebas dari dependensi pihak ketiga.
