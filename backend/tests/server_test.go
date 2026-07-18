// /backend/tests/server_test.go
/*
 * Berkas Pengujian Integrasi (Integration Test Suite) - Backend PrivateCam.
 * Berkas ini melakukan build otomatis pada server Golang, menjalankannya pada port pengujian,
 * dan memvalidasi seluruh fungsionalitas HTTP API secara end-to-end termasuk
 * manajemen sesi, otentikasi login admin, perlindungan middleware, pengunggahan frame, dan logout.
 */
package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/cookiejar"
	"os"
	"os/exec"
	"testing"
	"time"
)

func TestServerIntegration(t *testing.T) {
	// 1. Build server binary terlebih dahulu
	t.Log("Melakukan kompilasi binary server untuk pengujian...")
	var cmdBuild *exec.Cmd = exec.Command("go", "build", "-o", "privatecam-test-server", "../main.go")
	var err error
	err = cmdBuild.Run()
	if err != nil {
		t.Fatalf("Gagal melakukan kompilasi server binary: %v", err)
	}
	
	// Pastikan membersihkan file binary dan database setelah pengetesan
	defer func() {
		_ = os.Remove("privatecam-test-server")
		_ = os.Remove("privatecam.db")
	}()

	// 2. Jalankan server binary di background dengan port kustom
	t.Log("Menyalakan server pengujian pada port 49222...")
	var cmdStart *exec.Cmd = exec.Command("./privatecam-test-server")
	cmdStart.Env = append(os.Environ(), "PORT=49222")
	err = cmdStart.Start()
	if err != nil {
		t.Fatalf("Gagal menjalankan server pengujian: %v", err)
	}
	
	// Hentikan proses server saat pengetesan selesai
	defer func() {
		if cmdStart.Process != nil {
			_ = cmdStart.Process.Kill()
		}
	}()

	// Jeda waktu sejenak untuk memastikan server telah sepenuhnya siap menerima koneksi
	time.Sleep(1500 * time.Millisecond)

	// Inisialisasi HTTP Client dengan penampung cookie (cookie jar)
	var jar *cookiejar.Jar
	jar, err = cookiejar.New(nil)
	if err != nil {
		t.Fatalf("Gagal membuat cookie jar: %v", err)
	}
	var client *http.Client = &http.Client{Jar: jar}

	// HTTP Client khusus yang tidak mengikuti pengalihan (redirect) secara otomatis
	var clientNoRedirect *http.Client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 3. 1. RedirectDasborTanpaLogin: Akses rute utama (/) tanpa login (Harus dialihkan ke /login.html)
	t.Run("RedirectDasborTanpaLogin", func(t *testing.T) {
		var resp *http.Response
		resp, err = clientNoRedirect.Get("http://localhost:49222/")
		if err != nil {
			t.Fatalf("Gagal mengakses rute utama: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("Harusnya mendapatkan status 303 (See Other), malah mendapatkan: %d", resp.StatusCode)
		}

		var location string = resp.Header.Get("Location")
		if location != "/login.html" {
			t.Errorf("Harusnya dialihkan ke '/login.html', malah ke: %s", location)
		}
	})

	// 4. 2. StatusTerproteksiTanpaLogin: Akses rute /status tanpa login (Harus ditolak dengan status 401)
	t.Run("StatusTerproteksiTanpaLogin", func(t *testing.T) {
		var resp *http.Response
		resp, err = clientNoRedirect.Get("http://localhost:49222/status")
		if err != nil {
			t.Fatalf("Gagal mengakses /status: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Harusnya mendapatkan status 401 Unauthorized, malah mendapatkan: %d", resp.StatusCode)
		}
	})

	// 5. 3. LoginGagalKredensialSalah: Login dengan kredensial salah (Harus ditolak dengan status 401)
	t.Run("LoginGagalKredensialSalah", func(t *testing.T) {
		var payload map[string]string = map[string]string{
			"username": "admin",
			"password": "salahpassword",
		}
		var jsonPayload []byte
		jsonPayload, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("Gagal melakukan marshal JSON: %v", err)
		}

		var resp *http.Response
		resp, err = client.Post("http://localhost:49222/login", "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Gagal memanggil rute login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Harusnya login ditolak dengan 401, malah mendapatkan: %d", resp.StatusCode)
		}
	})

	// 6. 4. LoginSukses: Login dengan kredensial benar (Harus sukses dengan status 200 dan menyimpan cookie)
	t.Run("LoginSukses", func(t *testing.T) {
		var payload map[string]string = map[string]string{
			"username": "admin",
			"password": "password",
		}
		var jsonPayload []byte
		jsonPayload, err = json.Marshal(payload)
		if err != nil {
			t.Fatalf("Gagal melakukan marshal JSON: %v", err)
		}

		var resp *http.Response
		resp, err = client.Post("http://localhost:49222/login", "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			t.Fatalf("Gagal memanggil rute login: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Harusnya login sukses dengan status 200, malah mendapatkan: %d", resp.StatusCode)
		}

		// Pastikan cookie session_token diterbitkan di header Set-Cookie
		var cookies []*http.Cookie = jar.Cookies(resp.Request.URL)
		var foundCookie bool = false
		var cookie *http.Cookie
		for _, cookie = range cookies {
			if cookie.Name == "session_token" && cookie.Value != "" {
				foundCookie = true
				break
			}
		}

		if !foundCookie {
			t.Error("Gagal mendeteksi cookie session_token setelah login sukses.")
		}
	})

	// 7. 5. UploadFrameBerhasil: Pengunggahan frame JPEG (Bypass login di rute /upload)
	t.Run("UploadFrameBerhasil", func(t *testing.T) {
		var mockJPEG []byte = []byte("fake-jpeg-binary-data")
		var resp *http.Response
		resp, err = client.Post("http://localhost:49222/upload", "image/jpeg", bytes.NewBuffer(mockJPEG))
		if err != nil {
			t.Fatalf("Gagal melakukan unggahan frame: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Harusnya pengunggahan sukses dengan status 200, malah mendapatkan: %d", resp.StatusCode)
		}
	})

	// 8. 6. StatusTerupdateSetelahUpload: Akses rute status setelah pengunggahan (Frame terupdate tidak boleh "Never")
	t.Run("StatusTerupdateSetelahUpload", func(t *testing.T) {
		var resp *http.Response
		resp, err = client.Get("http://localhost:49222/status")
		if err != nil {
			t.Fatalf("Gagal mengambil status: %v", err)
		}
		defer resp.Body.Close()

		var data map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			t.Fatalf("Gagal melakukan decoding JSON status: %v", err)
		}

		if data["last_frame_time"] == "Never" {
			t.Error("Kolom 'last_frame_time' harusnya sudah terisi dengan catatan waktu kirim, bukan 'Never' lagi.")
		}
	})

	// 9. 7. LogoutSukses: Proses keluar akun (Logout)
	t.Run("LogoutSukses", func(t *testing.T) {
		var resp *http.Response
		resp, err = client.Post("http://localhost:49222/logout", "application/json", nil)
		if err != nil {
			t.Fatalf("Gagal memanggil rute logout: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Harusnya logout sukses dengan status 200, malah mendapatkan: %d", resp.StatusCode)
		}

		// Validasi bahwa akses dasbor terkunci kembali setelah logout
		var respCheck *http.Response
		respCheck, err = clientNoRedirect.Get("http://localhost:49222/dashboard")
		if err != nil {
			t.Fatalf("Gagal mengecek dasbor paska logout: %v", err)
		}
		defer respCheck.Body.Close()

		if respCheck.StatusCode != http.StatusSeeOther {
			t.Errorf("Akses dasbor paska logout harusnya dialihkan (303), malah mendapatkan status: %d", respCheck.StatusCode)
		}

		var location string = respCheck.Header.Get("Location")
		if location != "/login.html" {
			t.Errorf("Harusnya dialihkan ke '/login.html', malah ke: %s", location)
		}
	})
}
