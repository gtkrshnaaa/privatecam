# Private Cam - Prototype CCTV Privat dengan ESP32-CAM & Golang

Proyek ini adalah implementasi sistem CCTV pintar mandiri (*self-hosted*) yang aman, efisien, dan sepenuhnya berada di bawah kendali pengguna. Dengan menggunakan mikrokontroler **ESP32-CAM** untuk menangkap gambar dan server backend **Golang** di **VPS/Docker**, aliran video (*video stream*) dikirim langsung ke infrastruktur pribadi tanpa ketergantungan pada layanan *cloud* pihak ketiga.

---

## 1. Latar Belakang
CCTV komersial saat ini sangat bergantung pada ekosistem *cloud* pihak ketiga yang memicu kekhawatiran terkait privasi data, potensi kebocoran rekaman, serta biaya langganan bulanan. Di sisi lain, CCTV murni lokal kehilangan fitur pemantauan jarak jauh. 

Proyek ini menawarkan jalan tengah: **tetap terkoneksi ke internet global, tetapi menggunakan infrastruktur pribadi (self-hosted)** yang dijembatani oleh VPS Linux pribadi atau mesin Docker lokal.

---

## 2. Daftar Alat & Komponen (Bill of Materials)
*   **Modul ESP32-CAM + Kamera OV2640**: Unit utama untuk menangkap gambar, memproses data, dan mengirimkannya via Wi-Fi.
*   **Shield ESP32-CAM-MB**: Board bawah dengan IC CH340/FT232 untuk melakukan *flashing code* via USB sekaligus sebagai regulator daya operasional.
*   **Kabel Data USB (Micro USB / Type-C)**: Penghubung board MB ke komputer (untuk pemrogaman) atau ke adaptor (untuk operasional).
*   **Adaptor Charger HP 5V (Minimal 1A - 2A)**: Sumber daya utama agar CCTV dapat menyala secara mandiri (*stand-alone*).

---

## 3. Skema Koneksi Perangkat
Perakitan fisik tidak membutuhkan kabel jumper atau papan sirkuit tambahan karena modul dirancang untuk saling menancap langsung:
1.  **Pemasangan Kamera**: Buka klip hitam soket pita pada board ESP32-CAM, masukkan kabel pita kamera OV2640 (pin tembaga menghadap board), lalu kunci kembali klipnya.
2.  **Penggabungan Board**: Sejajarkan pin *male* ESP32-CAM dengan soket *female* pada Shield MB (antena Wi-Fi dan port USB searah), lalu tekan perlahan hingga rapat.
3.  **Fase Operasional**:
    *   *Fase Flashing*: Hubungkan kabel USB dari Shield MB langsung ke komputer untuk menyuplai daya sekaligus mengunggah program (*firmware*).
    *   *Fase Deploy*: Cabut dari komputer, lalu hubungkan kabel USB ke Adaptor Charger HP 5V untuk berjalan secara mandiri dan otomatis terhubung ke Wi-Fi.

---

## 4. Struktur Folder Project (Monorepo)
```text
cctv-esp32cam-golang/
├── firmware/          # Kode program ESP32-CAM (C++/Arduino)
├── backend/           # Server HTTP & API Streaming (Golang)
├── frontend/          # Tampilan Dasbor Monitoring (HTML/CSS/JS)
├── Dockerfile         # Konfigurasi multi-stage build backend & frontend
├── docker-compose.yml # Konfigurasi container deployment CCTV
└── install.sh         # Skrip Bash otomasi instalasi & deployment di Ubuntu
```

---

## 5. Alur Logika & Arsitektur Data (MJPEG Streaming)
Aliran data berjalan secara linear dari perangkat keras ke server, lalu didistribusikan ke browser:
1.  **Inisialisasi & Koneksi (ESP32-CAM)**: Modul mengaktifkan kamera OV2640 (resolusi VGA), terhubung ke Wi-Fi lokal, dan melakukan *handshake* awal dengan server Golang.
2.  **Pengiriman Gambar Kontinu**: ESP32-CAM masuk ke siklus looping tanpa henti: menangkap gambar -> menyimpan ke memori internal (*frame buffer*) sebagai biner JPEG -> mengirim data biner tersebut via HTTP POST ke endpoint upload Golang -> mengosongkan *frame buffer* untuk siklus berikutnya.
3.  **Pemrosesan Server (Golang)**:
    *   *Upload Handler*: Menerima kiriman gambar JPEG dari ESP32-CAM.
    *   *Concurrency & Channels*: Data dialirkan secara *non-blocking* menggunakan Go Channel untuk disiarkan (*broadcast*).
    *   *Streaming Handler*: Menyiarkan gambar secara *real-time* ke web frontend dengan header HTTP khusus (`multipart/x-mixed-replace`) dengan koneksi yang dijaga tetap terbuka (*keep-alive*).
4.  **Tampilan Dashboard (Web Frontend)**: Browser pengguna mengakses alamat server streaming Golang. Karena menggunakan metode MJPEG, browser secara otomatis mengenali dan merender kumpulan gambar yang bergerak tersebut secara langsung pada tag `<img>` standar tanpa memerlukan pustaka JavaScript yang berat.

---

## 6. Docker & Otomasi Deployment
*   **Dockerization (Multi-stage Build)**: Menggunakan image Golang resmi untuk melakukan kompilasi kode backend menjadi berkas biner ringan, kemudian memindahkan berkas biner tersebut ke image minimalis (Alpine Linux) bersama berkas statis frontend. Hasilnya adalah kontainer Docker yang sangat hemat penggunaan RAM dan CPU.
*   **Docker Compose**: Memetakan port eksternal (misal `8080`) ke kontainer dan mengatur kebijakan *restart* otomatis.
*   **Skrip Otomasi (`install.sh`)**:
    1.  Memeriksa dan memasang Docker / Docker Compose jika belum terinstal di sistem Ubuntu.
    2.  Menyiapkan berkas lingkungan (*environment configuration*).
    3.  Menjalankan kontainer di latar belakang dan menampilkan informasi URL IP Address yang perlu dimasukkan ke dalam konfigurasi *firmware* ESP32-CAM.
