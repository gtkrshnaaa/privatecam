// /firmware/firmware.ino
/*
 * Firmware untuk Prototype CCTV Privat berbasis ESP32-CAM.
 * Menginisialisasi kamera OV2640, menghubungkan ke Wi-Fi, 
 * dan mengirimkan frame gambar JPEG via HTTP POST ke backend Golang.
 */

#include "esp_camera.h"
#include <WiFi.h>
#include <HTTPClient.h>

// ==========================================
// KONFIGURASI WIFI & SERVER (Sesuaikan di sini)
// ==========================================
const char* ssid = "NAMA_WIFI_ANDA";
const char* password = "PASSWORD_WIFI_ANDA";
const char* serverUrl = "http://ALAMAT_IP_SERVER_GOLANG:49111/upload";

// ==========================================
// KONFIGURASI PIN ESP32-CAM (AI-THINKER MODULE)
// ==========================================
#define PWDN_GPIO_NUM     32
#define RESET_GPIO_NUM    -1
#define XCLK_GPIO_NUM      0
#define SIOD_GPIO_NUM     26
#define SIOC_GPIO_NUM     27

#define Y9_GPIO_NUM       35
#define Y8_GPIO_NUM       34
#define Y7_GPIO_NUM       39
#define Y6_GPIO_NUM       36
#define Y5_GPIO_NUM       21
#define Y4_GPIO_NUM       19
#define Y3_GPIO_NUM       18
#define Y2_GPIO_NUM        5
#define VSYNC_GPIO_NUM    25
#define HREF_GPIO_NUM     23
#define PCLK_GPIO_NUM     22

// Fungsi untuk menginisialisasi modul kamera
bool initCamera() {
  camera_config_t config;
  config.ledc_channel = LEDC_CHANNEL_0;
  config.ledc_timer = LEDC_TIMER_0;
  config.pin_d0 = Y2_GPIO_NUM;
  config.pin_d1 = Y3_GPIO_NUM;
  config.pin_d2 = Y4_GPIO_NUM;
  config.pin_d3 = Y5_GPIO_NUM;
  config.pin_d4 = Y6_GPIO_NUM;
  config.pin_d5 = Y7_GPIO_NUM;
  config.pin_d6 = Y8_GPIO_NUM;
  config.pin_d7 = Y9_GPIO_NUM;
  config.pin_xclk = XCLK_GPIO_NUM;
  config.pin_pclk = PCLK_GPIO_NUM;
  config.pin_vsync = VSYNC_GPIO_NUM;
  config.pin_href = HREF_GPIO_NUM;
  config.pin_sscb_sda = SIOD_GPIO_NUM;
  config.pin_sscb_scl = SIOC_GPIO_NUM;
  config.pin_pwdn = PWDN_GPIO_NUM;
  config.pin_reset = RESET_GPIO_NUM;
  config.xclk_freq_hz = 20000000;
  config.pixel_format = PIXFORMAT_JPEG;

  // Atur resolusi dan kualitas gambar
  // VGA (640x480) cocok untuk transfer data yang stabil pada VPS berspesifikasi rendah
  if(psramFound()){
    config.frame_size = FRAMESIZE_VGA;
    config.jpeg_quality = 12; // Rentang 10-63, semakin rendah nilai semakin baik kualitasnya
    config.fb_count = 2;
  } else {
    config.frame_size = FRAMESIZE_SVGA;
    config.jpeg_quality = 12;
    config.fb_count = 1;
  }

  // Inisialisasi kamera
  esp_err_t err = esp_camera_init(&config);
  if (err != ESP_OK) {
    Serial.printf("Gagal inisialisasi kamera dengan error 0x%x", err);
    return false;
  }
  
  // Opsi sensor kamera untuk membalikkan gambar jika terbalik
  sensor_t * s = esp_camera_sensor_get();
  s->set_vflip(s, 1);   // Balik gambar vertikal (1 = aktif, 0 = nonaktif)
  s->set_hmirror(s, 1); // Balik gambar horizontal (1 = aktif, 0 = nonaktif)

  return true;
}

// Fungsi untuk menghubungkan ke jaringan Wi-Fi
void setupWiFi() {
  Serial.print("Menghubungkan ke ");
  Serial.println(ssid);
  
  WiFi.begin(ssid, password);
  
  // Terus ulangi koneksi sampai terhubung
  while (WiFi.status() != WL_CONNECTED) {
    delay(500);
    Serial.print(".");
  }
  
  Serial.println("");
  Serial.println("Wi-Fi Terhubung.");
  Serial.print("Alamat IP ESP32-CAM: ");
  Serial.println(WiFi.localIP());
}

void setup() {
  // Inisialisasi Serial Monitor untuk debugging
  Serial.begin(115200);
  Serial.setDebugOutput(true);
  Serial.println();

  // Inisialisasi Wi-Fi
  setupWiFi();

  // Inisialisasi Driver Kamera
  if (!initCamera()) {
    Serial.println("Kamera gagal diaktifkan. Silakan restart board.");
    return;
  }
  
  Serial.println("Inisialisasi Kamera Sukses. Siap mengirim gambar...");
}

void loop() {
  // Pastikan Wi-Fi tetap terhubung
  if (WiFi.status() != WL_CONNECTED) {
    setupWiFi();
  }

  camera_fb_t * fb = NULL;
  
  // Ambil frame gambar dari kamera
  fb = esp_camera_fb_get();
  if (!fb) {
    Serial.println("Gagal mengambil gambar dari kamera.");
    delay(1000);
    return;
  }

  // Kirim data biner gambar ke server Golang via HTTP POST
  HTTPClient http;
  
  // Inisialisasi HTTP request ke server tujuan
  http.begin(serverUrl);
  http.addHeader("Content-Type", "image/jpeg");

  // Kirim data biner JPEG
  int httpResponseCode = http.POST(fb->buf, fb->len);

  if (httpResponseCode > 0) {
    // Unggahan sukses
    String response = http.getString();
    Serial.printf("Gambar terkirim. Respon Server [%d]: %s\n", httpResponseCode, response.c_str());
  } else {
    // Terjadi kesalahan koneksi
    Serial.printf("Gagal mengirim gambar. Error: %s\n", http.errorToString(httpResponseCode).c_str());
  }

  // Akhiri koneksi HTTP
  http.end();

  // Kembalikan frame buffer agar siap digunakan kembali
  esp_camera_fb_return(fb);

  // Berikan jeda waktu antar pengiriman frame (misal 150 ms untuk ~6-7 FPS)
  delay(150);
}
