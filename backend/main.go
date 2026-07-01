package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
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
	streamServer.UpdateLastFrameTime()
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
	streamServer.ClientsMu.Unlock()

	// Bersihkan pendaftaran saat client putus koneksi
	defer func() {
		streamServer.ClientsMu.Lock()
		delete(streamServer.Clients, clientChan)
		streamServer.ClientsMu.Unlock()
		close(clientChan)
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

	var responseData map[string]interface{} = map[string]interface{}{
		"status":          "online",
		"active_clients":  activeClients,
		"last_frame_time": lastActive,
		"server_time":     time.Now().Format(time.RFC3339),
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

func main() {
	// Jalankan thread broadcast server
	go streamServer.Start()

	var mux *http.ServeMux = http.NewServeMux()

	// Daftarkan route HTTP
	mux.HandleFunc("/upload", handleUpload)
	mux.HandleFunc("/stream", handleStream)
	mux.HandleFunc("/status", handleStatus)

	// Layani aset statis frontend
	var fileServer http.Handler = http.FileServer(http.Dir("frontend"))
	mux.Handle("/", fileServer)

	var serverAddr string = ":8080"
	log.Printf("Server berjalan di alamat http://localhost%s\n", serverAddr)
	
	var err error
	err = http.ListenAndServe(serverAddr, mux)
	if err != nil {
		log.Fatalf("Server gagal berjalan: %v", err)
	}
}
