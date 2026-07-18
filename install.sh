#!/usr/bin/env bash
# /install.sh

# =========================================================================
# Skrip Otomasi Deployment Server PrivateCam.
# Berkas ini melakukan pengecekan ketersediaan sistem Docker & Docker Compose,
# mendeteksi IP lokal host secara otomatis, membangun image Docker berbasis
# multi-stage builder, serta menyalakan kontainer PrivateCam di latar belakang.
# =========================================================================

# Cetak warna untuk output terminal
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Memulai Inisialisasi Server Private Cam ===${NC}"

# 1. Validasi Keberadaan Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}[ERROR] Docker tidak terdeteksi di sistem.${NC}"
    echo -e "${YELLOW}[TIPS] Pasang Docker terlebih dahulu dengan perintah:${NC}"
    echo -e "       sudo apt update && sudo apt install docker.io -y"
    exit 1
else
    echo -e "${GREEN}[OK] Docker terdeteksi di sistem: $(docker --version)${NC}"
fi

# 2. Validasi Keberadaan Docker Compose (v2 atau v1)
DOCKER_COMPOSE_CMD=""
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
    echo -e "${GREEN}[OK] Docker Compose V2 terdeteksi: $(docker compose version)${NC}"
elif command -v docker-compose &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
    echo -e "${GREEN}[OK] Docker Compose V1 terdeteksi: $(docker-compose --version)${NC}"
else
    echo -e "${RED}[ERROR] Docker Compose tidak terdeteksi di sistem.${NC}"
    echo -e "${YELLOW}[TIPS] Pasang Docker Compose terlebih dahulu dengan perintah:${NC}"
    echo -e "       sudo apt install docker-compose-v2 -y"
    exit 1
fi

# 3. Mendapatkan IP Address lokal sistem
LOCAL_IP=$(hostname -I | awk '{print $1}')
if [ -z "$LOCAL_IP" ]; then
    LOCAL_IP="localhost"
fi

echo -e "${BLUE}[INFO] Menyalakan kontainer menggunakan $DOCKER_COMPOSE_CMD...${NC}"

# 4. Menjalankan kontainer di latar belakang
$DOCKER_COMPOSE_CMD up -d --build

if [ $? -eq 0 ]; then
    echo -e "\n${GREEN}===============================================${NC}"
    echo -e "${GREEN}[SUKSES] Kontainer Private Cam berhasil dijalankan!${NC}"
    echo -e "${GREEN}===============================================${NC}"
    echo -e "${BLUE}Berikut adalah informasi akses Anda:${NC}"
    echo -e "  - Dashboard Monitor : ${YELLOW}http://${LOCAL_IP}:49111/${NC}"
    echo -e "  - Endpoint Upload   : ${YELLOW}http://${LOCAL_IP}:49111/upload${NC}"
    echo -e ""
    echo -e "${BLUE}[PENTING] Masukkan URL Endpoint Upload di atas pada berkas:${NC}"
    echo -e "          ${YELLOW}firmware/src/main.cpp${NC} -> ${YELLOW}const char* serverUrl${NC} di bagian atas berkas"
    echo -e "==============================================="
else
    echo -e "${RED}[ERROR] Gagal menjalankan kontainer. Silakan periksa log Docker.${NC}"
    exit 1
fi
