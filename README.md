# File System Simulator

Aplikasi "File System Simulator" adalah sebuah simulator sistem berkas (file system) sederhana yang dibangun sebagai tugas mata kuliah Sistem Operasi. Aplikasi ini mendemonstrasikan cara kerja manajemen file dan direktori dalam sistem operasi.

## Deskripsi Proyek

Aplikasi ini menyediakan antarmuka grafis (GUI) untuk mensimulasikan operasi dasar sistem berkas, seperti membuat, membaca, menulis, dan menghapus file atau direktori. Sistem berkas yang diimplementasikan menggunakan struktur data File Allocation Table (FAT) untuk mengelola alokasi blok pada "disk" virtual.

## Teknologi yang Digunakan

- **Bahasa Pemrograman**: Go (Golang)
- **Library GUI**: Fyne.io
- **Metode Alokasi**: File Allocation Table (FAT)

## Fitur Utama

1. **Manajemen File dan Direktori**

   - Membuat file baru
   - Membuat direktori baru
   - Membuka dan mengedit isi file
   - Menghapus file dan direktori

2. **Navigasi Sistem Berkas**

   - Menjelajahi struktur direktori
   - Navigasi ke direktori induk (parent directory)
   - Menampilkan path direktori saat ini

3. **Visualisasi Metadata**
   - Menampilkan ukuran file
   - Menampilkan waktu modifikasi terakhir
   - Menampilkan tipe item (file atau direktori)

## Struktur Sistem Berkas

- **Block Size**: 256 bytes
- **Total Blocks**: 256 blocks
- **Ukuran Disk Total**: 32 KB (256 x 256 bytes)
- **File Allocation Table (FAT)**: Tabel untuk mengelola alokasi blok dan rantai blok

## Implementasi Internal

1. **Struktur Data Utama**

   - `FAT`: Tabel alokasi blok
   - `Disk`: Array dari blok data
   - `DirectoryEntry`: Struktur untuk menyimpan metadata file/direktori
   - `FileSystem`: Struktur untuk mengelola keadaan sistem berkas

2. **Operasi Dasar**
   - `CreateFile`: Membuat file baru
   - `CreateDirectory`: Membuat direktori baru
   - `ReadFromFile`: Membaca konten dari file
   - `WriteToFile`: Menulis konten ke file
   - `DeleteEntry`: Menghapus file atau direktori
   - `ChangeDirectory`: Pindah antar direktori

## Cara Menjalankan Aplikasi

1. Pastikan Go (Golang) telah terinstall di komputer Anda
2. Clone atau download repositori ini
3. Masuk ke direktori proyek
4. Jalankan aplikasi dengan perintah:
   ```
   go run .
   ```

## Antarmuka Pengguna

Aplikasi menyediakan antarmuka grafis yang intuitif dengan:

- Panel navigasi untuk berpindah antar direktori
- Daftar file dan direktori dalam tampilan list
- Tombol untuk operasi umum (membuat file/folder, menghapus, dll)
- Dialog untuk membuat file/folder dan mengedit konten

## Keterbatasan

- Ukuran disk virtual terbatas pada 32 KB
- Tidak mendukung fitur lanjutan seperti permission, symbolic links, dll
- Memori hanya bersifat sementara (tidak persisten setelah aplikasi ditutup)

## Kontributor

- [Nama Mahasiswa]
- [NIM]

## Mata Kuliah

Sistem Operasi - Semester 4

---

_Catatan: Proyek ini dibuat untuk tujuan pembelajaran dan tidak dimaksudkan untuk penggunaan produksi._
