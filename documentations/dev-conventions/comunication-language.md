# Konvensi Bahasa Komunikasi

Dokumen ini mengatur penggunaan bahasa untuk seluruh elemen komunikatif dan informatif dalam proyek Private Cam.

## 1. Lingkup Penggunaan Bahasa Indonesia (Teknis)
Bahasa Indonesia yang bersifat teknis (menggunakan istilah teknis bahasa Inggris jika tidak ada padanan yang pas atau jika terjemahannya terasa janggal) wajib digunakan pada:
*   **Dokumen Markdown (`.md`)**: Seluruh berkas dokumentasi proyek.
*   **Komentar dalam Kode Program**: Penjelasan logika, algoritma, atau instruksi teknis di dalam *source code* (Golang, C++, JavaScript).
*   **Antarmuka Pengguna (UI/Frontend)**: Seluruh teks, label, pesan error, log, dan elemen informatif yang tampil pada halaman web monitoring.

### Contoh Penggunaan Bahasa Indonesia Teknis:
*   *Benar:* "Fungsi ini melakukan broadcast data frame JPEG ke seluruh client yang terhubung menggunakan Go Channel."
*   *Salah (Terlalu memaksakan terjemahan):* "Fungsi ini melakukan siaran data bingkai JPEG ke seluruh klien yang terhubung menggunakan Saluran Go."
*   *Salah (Menggunakan bahasa non-teknis):* "Fungsi ini buat ngirim gambar ke semua orang yang lagi buka web."

## 2. Lingkup Penggunaan Bahasa Inggris
Bahasa Inggris wajib digunakan secara penuh untuk elemen-elemen struktural dan deklaratif dalam kode:
*   **Nama Berkas dan Folder**: Seluruh nama file (misalnya `image_processor.go`, `main.go`) dan folder (misalnya `backend`, `frontend`, `firmware`).
*   **Deklarasi Kode**: Nama *class*, *struct*, *interface*, fungsi, *method*, variabel, konstanta, *package*, dan sejenisnya.

### Contoh Deklarasi Kode:
```go
// Nama fungsi, variabel, dan tipe data menggunakan Bahasa Inggris
// Komentar penjelasan menggunakan Bahasa Indonesia Teknis

// FrameData merepresentasikan struktur data gambar yang dikirim dari ESP32-CAM.
type FrameData struct {
    RawData []byte
    Size    int
}

// BroadcastFrame mengirimkan data gambar ke channel tujuan.
func (server *StreamServer) BroadcastFrame(frame *FrameData) {
    // Kirim data ke channel secara non-blocking
    select {
    case server.FrameChannel <- frame:
    default:
        // Channel penuh, abaikan frame ini
    }
}
```
