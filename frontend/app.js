// /frontend/app.js
/*
 * Logika Interaksi Klien (JS) - Dasbor Utama PrivateCam.
 * Menangani inisialisasi aliran video MJPEG dari server backend, polling berkala status sistem,
 * manajemen sesi login (pengalihan otomatis ke login.html apabila 401 Unauthorized),
 * kontrol layar penuh, pencatatan log aktivitas secara real-time, dan fungsi logout.
 */
// State Management
let cameraOnline = false;
let lastKnownFrameTime = null;

// DOM Elements
const mjpegStream = document.getElementById('mjpegStream');
const streamPlaceholder = document.getElementById('streamPlaceholder');
const cameraStatusBadge = document.getElementById('cameraStatusBadge');
const serverStatusBadge = document.getElementById('serverStatusBadge');
const statClients = document.getElementById('statClients');
const statLastFrame = document.getElementById('statLastFrame');
const logContainer = document.getElementById('logContainer');
const btnRefresh = document.getElementById('btnRefresh');
const btnFullscreen = document.getElementById('btnFullscreen');
const btnLogout = document.getElementById('btnLogout');
const videoContainer = document.querySelector('.video-container');

// Menambahkan entri log ke kontainer log
function addLog(message, type = 'system') {
    const entry = document.createElement('div');
    entry.className = `log-entry ${type}`;
    const now = new Date();
    const timeStr = now.toTimeString().split(' ')[0];
    entry.textContent = `[${timeStr}] ${message}`;
    logContainer.appendChild(entry);
    logContainer.scrollTop = logContainer.scrollHeight;
}

// Inisialisasi aliran video
function startStream() {
    addLog("Memulai pemuatan aliran video MJPEG...", "system");
    mjpegStream.src = '/stream';
    mjpegStream.onload = () => {
        addLog("Aliran video MJPEG berhasil dimuat.", "upload");
        mjpegStream.classList.remove('hidden');
        streamPlaceholder.classList.add('hidden');
    };
    mjpegStream.onerror = () => {
        addLog("Aliran video terputus atau gagal dimuat.", "error");
        stopStream();
    };
}

// Menghentikan aliran video dan kembali ke placeholder
function stopStream() {
    mjpegStream.src = '';
    mjpegStream.classList.add('hidden');
    streamPlaceholder.classList.remove('hidden');
}

// Memperbarui status kamera berdasarkan data server
function updateCameraStatus(statusData) {
    const lastFrameStr = statusData.last_frame_time;
    
    if (lastFrameStr === "Never") {
        if (cameraOnline) {
            cameraOnline = false;
            stopStream();
            addLog("Kamera belum pernah mengirim data.", "warning");
        }
        cameraStatusBadge.textContent = "Offline (Menunggu Data)";
        cameraStatusBadge.className = "feed-badge offline";
        statLastFrame.textContent = "Belum ada data";
        return;
    }

    const lastTime = new Date(lastFrameStr);
    const serverTime = new Date(statusData.server_time);
    const diffSeconds = (serverTime - lastTime) / 1000;
    
    // Format waktu lokal untuk tampilan statistik
    statLastFrame.textContent = lastTime.toLocaleTimeString();

    // Jika frame terakhir dikirim kurang dari 8 detik yang lalu, anggap kamera online
    const isOnline = diffSeconds < 8;

    if (isOnline !== cameraOnline) {
        cameraOnline = isOnline;
        if (cameraOnline) {
            cameraStatusBadge.textContent = "Online";
            cameraStatusBadge.className = "feed-badge online";
            addLog("Kamera terdeteksi ONLINE. Memuat feed...", "upload");
            startStream();
        } else {
            cameraStatusBadge.textContent = "Offline (Koneksi Terputus)";
            cameraStatusBadge.className = "feed-badge offline";
            addLog("Kamera kehilangan koneksi (timeout).", "error");
            stopStream();
        }
    }
    
    lastKnownFrameTime = lastFrameStr;
}

// Polling status server secara berkala
async function fetchServerStatus() {
    try {
        const response = await fetch('/status');
        if (response.status === 401) {
            window.location.href = '/login';
            return;
        }
        
        if (!response.ok) {
            throw new Error(`HTTP error: ${response.status}`);
        }
        
        const data = await response.json();
        
        // Perbarui badge server ke online
        serverStatusBadge.innerHTML = `<span class="status-dot green"></span><span class="status-text">Server Hubungkan: Online</span>`;
        
        // Perbarui statistik
        statClients.textContent = data.active_clients;
        
        // Perbarui status kamera
        updateCameraStatus(data);
        
    } catch (err) {
        // Perbarui badge server ke offline
        serverStatusBadge.innerHTML = `<span class="status-dot red"></span><span class="status-text">Server Hubungkan: Putus</span>`;
        statClients.textContent = "-";
        
        if (cameraOnline) {
            cameraOnline = false;
            stopStream();
            addLog("Koneksi ke backend server terputus.", "error");
        }
        
        cameraStatusBadge.textContent = "Menghubungkan ke Server...";
        cameraStatusBadge.className = "feed-badge offline";
    }
}

// Event Listeners
btnRefresh.addEventListener('click', () => {
    addLog("Melakukan penyegaran aliran video secara manual...", "system");
    stopStream();
    setTimeout(() => {
        mjpegStream.src = '/stream?t=' + new Date().getTime();
        mjpegStream.classList.remove('hidden');
        streamPlaceholder.classList.add('hidden');
    }, 300);
});

btnFullscreen.addEventListener('click', () => {
    if (!document.fullscreenElement) {
        videoContainer.requestFullscreen()
            .then(() => {
                addLog("Mengaktifkan mode layar penuh.", "system");
            })
            .catch(err => {
                addLog(`Gagal memuat layar penuh: ${err.message}`, "error");
            });
    } else {
        document.exitFullscreen();
    }
});

// Deteksi perubahan layar penuh untuk merapikan log
document.addEventListener('fullscreenchange', () => {
    if (!document.fullscreenElement) {
        addLog("Keluar dari mode layar penuh.", "system");
    }
});

// Logout handler
btnLogout.addEventListener('click', async () => {
    addLog("Melakukan proses keluar (logout)...", "system");
    try {
        const response = await fetch('/logout', { method: 'POST' });
        if (response.ok) {
            window.location.href = '/login';
        } else {
            addLog("Gagal keluar dari sesi.", "error");
        }
    } catch (err) {
        addLog("Gagal terhubung untuk logout.", "error");
    }
});

// Menjalankan inisialisasi awal
fetchServerStatus();
// Polling setiap 3 detik
setInterval(fetchServerStatus, 3000);
