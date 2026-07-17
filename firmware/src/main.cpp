// /firmware/src/main.cpp
/*
 * Firmware untuk Prototype CCTV Privat berbasis ESP32-CAM.
 * Menginisialisasi kamera OV2640, menghubungkan ke Wi-Fi, 
 * dan mengirimkan frame gambar JPEG via HTTP POST ke backend Golang.
 * Kompatibel dengan PlatformIO IDE.
 */

#include <Arduino.h>
#include "esp_camera.h"
#include <WiFi.h>
#include <HTTPClient.h>
#include "soc/soc.h"
#include "soc/rtc_cntl_reg.h"

// ==========================================
// KONFIGURASI WIFI & SERVER
// ==========================================
const char* ssid = "ifana cantik";
const char* password = "111222333";
const char* serverUrl = "http://72.61.213.51:49111/upload";

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

#define FLASH_GPIO_NUM     4

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
  config.pin_sccb_sda = SIOD_GPIO_NUM;
  config.pin_sccb_scl = SIOC_GPIO_NUM;
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

// Fungsi pembantu untuk mengirimkan log aktivitas langsung ke dashboard server
void sendLogToServer(String message) {
  if (WiFi.status() == WL_CONNECTED) {
    HTTPClient http;
    String logUrl = String(serverUrl);
    logUrl.replace("/upload", "/log");
    
    http.begin(logUrl);
    http.addHeader("Content-Type", "text/plain");
    http.setTimeout(3000);
    int httpResponseCode = http.POST(message);
    http.end();
  }
  Serial.println("[ESP32 LOG] " + message);
}


void setup() {
  // Matikan brownout detector agar ESP32 tidak reset saat terjadi drop tegangan kecil akibat transmisi Wi-Fi
  WRITE_PERI_REG(RTC_CNTL_BROWN_OUT_REG, 0);

  // Inisialisasi Serial Monitor untuk debugging
  Serial.begin(115200);
  Serial.setDebugOutput(true);
  Serial.println();

  // Matikan LED flash secara default untuk menghemat daya dan mencegah brownout
  pinMode(FLASH_GPIO_NUM, OUTPUT);
  digitalWrite(FLASH_GPIO_NUM, LOW);


  // Inisialisasi Wi-Fi
  setupWiFi();

  // Kirim log awal boot berhasil ke server
  sendLogToServer("ESP32-CAM berhasil boot dan terhubung ke Wi-Fi.");

  // Inisialisasi Driver Kamera
  if (!initCamera()) {
    sendLogToServer("Gagal mengaktifkan Kamera! Silakan restart board.");
    return;
  }
  
  sendLogToServer("Kamera berhasil diaktifkan. Memulai streaming gambar...");
}

void loop() {
  // Pastikan Wi-Fi tetap terhubung
  if (WiFi.status() != WL_CONNECTED) {
    setupWiFi();
    sendLogToServer("Koneksi Wi-Fi pulih kembali.");
  }

  camera_fb_t * fb = NULL;
  
  // Ambil frame gambar dari kamera
  fb = esp_camera_fb_get();
  if (!fb) {
    sendLogToServer("Gagal mengambil gambar dari kamera (Sensor/Frame error).");
    delay(1000);
    return;
  }

  // Kirim data biner gambar ke server Golang via HTTP POST
  HTTPClient http;
  
  // Inisialisasi HTTP request ke server tujuan
  http.begin(serverUrl);
  http.addHeader("Content-Type", "image/jpeg");
  http.setTimeout(5000);

  // Kirim data biner JPEG
  int httpResponseCode = http.POST(fb->buf, fb->len);

  // Kembalikan frame buffer segera setelah POST untuk menghemat memori
  esp_camera_fb_return(fb);

  if (httpResponseCode > 0) {
    Serial.printf("Gambar terkirim. Respon Server [%d]\n", httpResponseCode);
  } else {
    String errMsg = "Gagal kirim gambar. Error: " + http.errorToString(httpResponseCode);
    sendLogToServer(errMsg);
  }

  // Akhiri koneksi HTTP
  http.end();

  // Jeda antar pengiriman frame (500 ms untuk ~2 FPS, lebih stabil untuk VPS jarak jauh)
  delay(500);
}
