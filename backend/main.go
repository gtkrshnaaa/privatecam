// /backend/main.go
/*
 * Backend Server Golang untuk Sistem CCTV Privat (PrivateCam).
 * File ini mengelola alur upload gambar dari ESP32-CAM, penyiaran video stream (MJPEG),
 * otentikasi admin berbasis SQLite (username: admin, password: password), manajemen sesi cookie,
 * dan penyajian aset statis untuk frontend dasbor.
 */
package main


import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

// StreamServer mengelola data streaming gambar dari ESP32-CAM ke banyak web client.
type StreamServer struct {
	// FrameChannel menerima data gambar mentah (JPEG) dari kamera.
	FrameChannel chan []byte
	// Clients menyimpan daftar channel aktif milik client yang sedang menonton.
	Clients map[chan []byte]bool
	// ClientsMu mengamankan akses map Clients dari akses konkuren.
	ClientsMu sync.Mutex
	// LastFrameTime mencatat waktu terakhir kali frame diterima.
	LastFrameTime time.Time
	// LastFrameTimeMu mengamankan akses ke LastFrameTime.
	LastFrameTimeMu sync.Mutex
}

// NewStreamServer membuat instance baru dari StreamServer.
func NewStreamServer() *StreamServer {
	var server *StreamServer = &StreamServer{
		FrameChannel: make(chan []byte, 30),
		Clients:      make(map[chan []byte]bool),
	}
	return server
}

// Start menjalankan loop utama untuk membagikan frame ke seluruh client.
func (server *StreamServer) Start() {
	var frame []byte
	for {
		frame = <-server.FrameChannel

		server.ClientsMu.Lock()
		var clientChan chan []byte
		for clientChan = range server.Clients {
			select {
			case clientChan <- frame:
			default:
				// Lewati jika channel client penuh untuk mencegah lag
			}
		}
		server.ClientsMu.Unlock()
	}
}

// UpdateLastFrameTime memperbarui catatan waktu penerimaan frame terakhir.
func (server *StreamServer) UpdateLastFrameTime() {
	server.LastFrameTimeMu.Lock()
	server.LastFrameTime = time.Now()
	server.LastFrameTimeMu.Unlock()
}

// GetLastFrameTime mengambil catatan waktu penerimaan frame terakhir.
func (server *StreamServer) GetLastFrameTime() time.Time {
	server.LastFrameTimeMu.Lock()
	var t time.Time = server.LastFrameTime
	server.LastFrameTimeMu.Unlock()
	return t
}

// Global server instance
var streamServer *StreamServer = NewStreamServer()

// Database global instance
var db *sql.DB

// Penyimpanan Log Aktivitas Terpusat
var systemLogs []string
var systemLogsMu sync.Mutex

func addSystemLog(msg string) {
	systemLogsMu.Lock()
	defer systemLogsMu.Unlock()
	if len(systemLogs) >= 50 {
		systemLogs = systemLogs[1:]
	}
	t := time.Now().Format("15:04:05")
	systemLogs = append(systemLogs, fmt.Sprintf("[%s] %s", t, msg))
}

// Inisialisasi database SQLite
func initDB() {
	var err error
	var dbPath string = os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "privatecam.db"
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("Gagal membuka database: %v", err)
	}

	var createTableQuery string = `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password TEXT
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Gagal membuat tabel users: %v", err)
	}

	// Cek apakah user admin default sudah ada
	var row *sql.Row = db.QueryRow("SELECT id FROM users WHERE username = ?", "admin")
	var id int
	err = row.Scan(&id)
	if err == sql.ErrNoRows {
		// Hashing password default "password"
		var hashedPasswordBytes []byte
		hashedPasswordBytes, err = bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
		if err != nil {
			log.Fatalf("Gagal melakukan hashing password: %v", err)
		}
		var hashedPassword string = string(hashedPasswordBytes)

		_, err = db.Exec("INSERT INTO users (username, password) VALUES (?, ?)", "admin", hashedPassword)
		if err != nil {
			log.Fatalf("Gagal memasukkan user admin default: %v", err)
		}
		log.Println("User admin default (username: admin, password: password) berhasil dibuat di database.")
		addSystemLog("[Server] User admin default berhasil dibuat.")
	} else if err != nil {
		log.Fatalf("Gagal melakukan verifikasi pengguna admin: %v", err)
	}
	addSystemLog("[Server] Database SQLite terkoneksi.")
}
}

// Map penyimpanan sesi secara memori (token -> username)
var sessions map[string]string = make(map[string]string)
var sessionsMu sync.RWMutex

func addSession(token string, username string) {
	sessionsMu.Lock()
	sessions[token] = username
	sessionsMu.Unlock()
}

func removeSession(token string) {
	sessionsMu.Lock()
	delete(sessions, token)
	sessionsMu.Unlock()
}

func checkSession(token string) bool {
	sessionsMu.RLock()
	var exists bool
	_, exists = sessions[token]
	sessionsMu.RUnlock()
	return exists
}

// Middleware otentikasi untuk melindungi endpoint dasbor
func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rute publik yang tidak memerlukan sesi login
		if r.URL.Path == "/login" || r.URL.Path == "/style.css" || r.URL.Path == "/upload" || r.URL.Path == "/log" {
			next.ServeHTTP(w, r)
			return
		}

		var cookie *http.Cookie
		var err error
		cookie, err = r.Cookie("session_token")
		if err != nil {
			// Kembalikan error 401 jika akses via API, atau redirect ke login jika akses via browser
			if r.URL.Path == "/status" || r.URL.Path == "/stream" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if !checkSession(cookie.Value) {
			if r.URL.Path == "/status" || r.URL.Path == "/stream" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleUpload menerima kiriman gambar biner (JPEG) dari ESP32-CAM via HTTP POST.
func handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	var data []byte
	var err error
	data, err = io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Gagal membaca body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(data) == 0 {
		http.Error(w, "Data kosong", http.StatusBadRequest)
		return
	}

	// Update waktu frame terakhir dan kirim data ke channel streaming
	var isFirstFrame bool = false
	streamServer.LastFrameTimeMu.Lock()
	if streamServer.LastFrameTime.IsZero() {
		isFirstFrame = true
	}
	streamServer.LastFrameTime = time.Now()
	streamServer.LastFrameTimeMu.Unlock()

	if isFirstFrame {
		addSystemLog("[Server] Menerima unggahan frame pertama dari ESP32-CAM.")
	}

	select {
	case streamServer.FrameChannel <- data:
	default:
		// Drop frame jika channel utama penuh
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Sukses")
}

// handleStream menyiarkan data gambar kontinu menggunakan format MJPEG ke browser.
func handleStream(w http.ResponseWriter, r *http.Request) {
	var clientChan chan []byte = make(chan []byte, 10)

	// Daftarkan client ke daftar aktif
	streamServer.ClientsMu.Lock()
	streamServer.Clients[clientChan] = true
	var clientCount int = len(streamServer.Clients)
	streamServer.ClientsMu.Unlock()

	addSystemLog(fmt.Sprintf("[Server] Klien baru terhubung ke stream. Total klien: %d", clientCount))

	// Bersihkan pendaftaran saat client putus koneksi
	defer func() {
		streamServer.ClientsMu.Lock()
		delete(streamServer.Clients, clientChan)
		var remainingClients int = len(streamServer.Clients)
		streamServer.ClientsMu.Unlock()
		close(clientChan)
		
		addSystemLog(fmt.Sprintf("[Server] Klien terputus dari stream. Sisa klien: %d", remainingClients))
	}()

	// Atur header HTTP untuk MJPEG Streaming
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("Cache-Control", "no-cache, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	var flusher http.Flusher
	var ok bool
	flusher, ok = w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming tidak didukung", http.StatusInternalServerError)
		return
	}

	var frame []byte
	for {
		select {
		case frame = <-clientChan:
			// Tulis header bagian frame JPEG
			fmt.Fprintf(w, "--frame\r\n")
			fmt.Fprintf(w, "Content-Type: image/jpeg\r\n")
			fmt.Fprintf(w, "Content-Length: %d\r\n\r\n", len(frame))
			
			// Tulis data biner gambar
			_, _ = w.Write(frame)
			fmt.Fprintf(w, "\r\n")
			
			// Kirim data langsung ke client
			flusher.Flush()
		case <-r.Context().Done():
			// Client memutuskan koneksi
			return
		}
	}
}

// handleStatus memberikan informasi status server dalam format JSON.
func handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	streamServer.ClientsMu.Lock()
	var activeClients int = len(streamServer.Clients)
	streamServer.ClientsMu.Unlock()

	var lastActive string = "Never"
	var lastTime time.Time = streamServer.GetLastFrameTime()
	if !lastTime.IsZero() {
		lastActive = lastTime.Format(time.RFC3339)
	}

	systemLogsMu.Lock()
	logsCopy := make([]string, len(systemLogs))
	copy(logsCopy, systemLogs)
	systemLogsMu.Unlock()

	var responseData map[string]interface{} = map[string]interface{}{
		"status":          "online",
		"active_clients":  activeClients,
		"last_frame_time": lastActive,
		"server_time":     time.Now().Format(time.RFC3339),
		"logs":            logsCopy,
	}

	var err error
	var jsonBytes []byte
	jsonBytes, err = json.Marshal(responseData)
	if err != nil {
		http.Error(w, `{"error": "Gagal serialisasi JSON"}`, http.StatusInternalServerError)
		return
	}

	_, _ = w.Write(jsonBytes)
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// handleLogin melayani halaman login (GET) dan memverifikasi kredensial (POST)
func handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		http.ServeFile(w, r, "frontend/login.html")
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
		return
	}

	var req LoginRequest
	var err error
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Format request salah", http.StatusBadRequest)
		return
	}

	// Verifikasi ketersediaan user di database
	var storedPassword string
	var row *sql.Row = db.QueryRow("SELECT password FROM users WHERE username = ?", req.Username)
	err = row.Scan(&storedPassword)
	if err == sql.ErrNoRows {
		http.Error(w, "Username atau password salah", http.StatusUnauthorized)
		return
	} else if err != nil {
		http.Error(w, "Kesalahan database", http.StatusInternalServerError)
		return
	}

	// Bandingkan password hash
	err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(req.Password))
	if err != nil {
		http.Error(w, "Username atau password salah", http.StatusUnauthorized)
		return
	}

	// Generate token sesi unik
	var tokenBytes []byte = make([]byte, 16)
	_, err = rand.Read(tokenBytes)
	if err != nil {
		http.Error(w, "Gagal memproses sesi", http.StatusInternalServerError)
		return
	}
	var token string = hex.EncodeToString(tokenBytes)

	// Simpan sesi login
	addSession(token, req.Username)

	// Set session cookie
	var cookie *http.Cookie = &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, cookie)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success": true, "message": "Login berhasil"}`))
}

// handleLogout menghapus cookie sesi aktif
func handleLogout(w http.ResponseWriter, r *http.Request) {
	var cookie *http.Cookie
	var err error
	cookie, err = r.Cookie("session_token")
	if err == nil {
		removeSession(cookie.Value)
	}

	// Set expired cookie
	var expiredCookie *http.Cookie = &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Now().Add(-1 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	}
	http.SetCookie(w, expiredCookie)

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"success": true, "message": "Logout berhasil"}`))
}

func main() {
	// Inisialisasi database SQLite
	initDB()

	// Jalankan thread broadcast server
	go streamServer.Start()

	var mux *http.ServeMux = http.NewServeMux()

	// Daftarkan route HTTP publik & terproteksi
	mux.HandleFunc("/upload", handleUpload)
	mux.HandleFunc("/stream", handleStream)
	mux.HandleFunc("/status", handleStatus)
	mux.HandleFunc("/login", handleLogin)
	mux.HandleFunc("/logout", handleLogout)
	mux.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Gagal membaca body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		
		msg := string(body)
		if msg != "" {
			addSystemLog("[ESP32] " + msg)
			log.Printf("[ESP32 LOG] %s\n", msg)
		}
		w.WriteHeader(http.StatusOK)
	})

	// Layani aset statis frontend & redirect rute utama/obsolete html
	var fileServer http.Handler = http.FileServer(http.Dir("frontend"))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}
		if r.URL.Path == "/login.html" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	// Rute Dasbor terproteksi
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Metode tidak diizinkan", http.StatusMethodNotAllowed)
			return
		}
		http.ServeFile(w, r, "frontend/index.html")
	})

	// Terapkan middleware otentikasi
	var protectedHandler http.Handler = authMiddleware(mux)

	var port string = os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	var serverAddr string = ":" + port
	log.Printf("Server berjalan di alamat http://localhost%s\n", serverAddr)
	
	var err error
	err = http.ListenAndServe(serverAddr, protectedHandler)
	if err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}
