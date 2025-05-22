// filesystem_logic.go
package filesystem_logic

import (
	"bytes"           // Akan kita gunakan untuk serialisasi/deserialisasi
	"encoding/binary" // Juga untuk serialisasi/deserialisasi
	"errors"
	"fmt"
	"time"
)

// Konstanta yang sudah kita bahas (bisa disesuaikan nanti)
const (
	BLOCK_SIZE       = 256         // Bytes per block (kecil untuk testing)
	TOTAL_BLOCKS     = 256         // Total blocks on the disk (total disk size = 128*256 = 32KB)
	FAT_FREE         = BlockID(-2) // Tandai blok kosong di FAT dengan -2
	FAT_EOF          = BlockID(-1) // Tandai akhir dari rantai blok file di FAT
	ROOT_DIR_BLOCK   = BlockID(1)  // Blok pertama untuk root directory (konvensi, blok 0 bisa untuk info disk)
	MAX_FILENAME_LEN = 28          // Maksimum panjang nama file (agar DirectoryEntry punya ukuran tetap)
	// Ukuran DirectoryEntry akan: MAX_FILENAME_LEN + 1 (Type) + 4 (StartBlock) + 8 (Size) + 8 (ModTime detik) + 4 (ModTime nanosec)
	// Perkiraan: 28 + 1 + 4 + 8 + 8 + 4 = 53 bytes. Kita bulatkan agar mudah, misal 64 bytes per entry.
	// Ini PENTING untuk serialisasi! Mari kita buat ukuran pasti.
	// Name (28) + Type (1 byte, tapi Go bool/enum bisa butuh padding) + StartBlock (4) + Size (8) + ModTime (12, time.Time internal)
	// Untuk Type, kita pakai int8 agar ukurannya pasti 1 byte saat serialisasi.
	// time.Time akan kita serialize sebagai UnixNano (int64) agar ukurannya pasti 8 bytes.
	// Jadi: Name(28) + Type(1) + StartBlock(4) + Size(8) + ModTimeUnixNano(8) = 49 bytes.
	// Kita akan gunakan ukuran ini.
	DIRECTORY_ENTRY_SIZE = 49
)

type BlockID int32 // Tipe untuk nomor blok

var Disk [][]byte // Representasi disk kita: slice dari blok, setiap blok adalah slice dari byte
var FAT []BlockID // File Allocation Table: slice di mana indeks adalah nomor blok

type FileType int8 // int8 agar ukuran pasti 1 byte

const (
	TYPE_FILE      FileType = 0
	TYPE_DIRECTORY FileType = 1
)

type DirectoryEntry struct {
	Name       [MAX_FILENAME_LEN]byte // Nama file/dir (fixed size array byte)
	Type       FileType               // Tipe entri (file atau direktori)
	StartBlock BlockID                // Blok pertama data di FAT (jika file) atau blok pertama isi direktori (jika direktori)
	Size       int64                  // Ukuran file dalam bytes (untuk direktori, bisa ukuran total entri di dalamnya)
	ModTime    int64                  // Waktu modifikasi terakhir (disimpan sebagai Unix nanoseconds)
}

// Fungsi untuk mengkonversi struct DirectoryEntry menjadi slice byte
func (de *DirectoryEntry) Serialize() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. Tulis Nama (fixed length)
	_, err := buf.Write(de.Name[:]) // Tulis seluruh array nama
	if err != nil {
		return nil, fmt.Errorf("serialize name: %w", err)
	}

	// 2. Tulis Tipe
	err = binary.Write(buf, binary.LittleEndian, de.Type)
	if err != nil {
		return nil, fmt.Errorf("serialize type: %w", err)
	}

	// 3. Tulis StartBlock
	err = binary.Write(buf, binary.LittleEndian, de.StartBlock)
	if err != nil {
		return nil, fmt.Errorf("serialize startblock: %w", err)
	}

	// 4. Tulis Size
	err = binary.Write(buf, binary.LittleEndian, de.Size)
	if err != nil {
		return nil, fmt.Errorf("serialize size: %w", err)
	}

	// 5. Tulis ModTime (sebagai int64 UnixNano)
	err = binary.Write(buf, binary.LittleEndian, de.ModTime)
	if err != nil {
		return nil, fmt.Errorf("serialize modtime: %w", err)
	}
	// Pastikan panjangnya sesuai DIRECTORY_ENTRY_SIZE
	serializedData := buf.Bytes()
	if len(serializedData) != DIRECTORY_ENTRY_SIZE {
		return nil, fmt.Errorf("serialized data length %d does not match DIRECTORY_ENTRY_SIZE %d", len(serializedData), DIRECTORY_ENTRY_SIZE)
	}
	return serializedData, nil
}

// Fungsi untuk mengkonversi slice byte kembali menjadi struct DirectoryEntry
func DeserializeEntry(data []byte) (DirectoryEntry, error) {
	var de DirectoryEntry
	if len(data) < DIRECTORY_ENTRY_SIZE { // Perlu data yang cukup
		return de, errors.New("insufficient data to deserialize directory entry")
	}
	buf := bytes.NewReader(data)

	// 1. Baca Nama
	_, err := buf.Read(de.Name[:])
	if err != nil {
		return de, fmt.Errorf("deserialize name: %w", err)
	}

	// 2. Baca Tipe
	err = binary.Read(buf, binary.LittleEndian, &de.Type)
	if err != nil {
		return de, fmt.Errorf("deserialize type: %w", err)
	}

	// 3. Baca StartBlock
	err = binary.Read(buf, binary.LittleEndian, &de.StartBlock)
	if err != nil {
		return de, fmt.Errorf("deserialize startblock: %w", err)
	}

	// 4. Baca Size
	err = binary.Read(buf, binary.LittleEndian, &de.Size)
	if err != nil {
		return de, fmt.Errorf("deserialize size: %w", err)
	}

	// 5. Baca ModTime
	err = binary.Read(buf, binary.LittleEndian, &de.ModTime)
	if err != nil {
		return de, fmt.Errorf("deserialize modtime: %w", err)
	}

	return de, nil
}

// Fungsi untuk menginisialisasi seluruh "Disk" dan FAT
// Ini seperti memformat disk.
func FormatDisk() error {
	// 1. Inisialisasi Disk: Buat slice Disk dengan TOTAL_BLOCKS elemen.
	//    Setiap elemen Disk[i] adalah slice byte dengan panjang BLOCK_SIZE.
	Disk = make([][]byte, TOTAL_BLOCKS)
	for i := 0; i < TOTAL_BLOCKS; i++ {
		Disk[i] = make([]byte, BLOCK_SIZE) // Setiap blok diisi byte kosong (nilai default 0)
	}
	fmt.Printf("Disk initialized with %d blocks, each %d bytes.\n", TOTAL_BLOCKS, BLOCK_SIZE)

	// 2. Inisialisasi FAT: Buat slice FAT dengan TOTAL_BLOCKS elemen.
	//    Setiap elemen FAT[i] awalnya adalah FAT_FREE (blok kosong).
	FAT = make([]BlockID, TOTAL_BLOCKS)
	for i := 0; i < TOTAL_BLOCKS; i++ {
		FAT[i] = FAT_FREE
	}
	fmt.Println("FAT initialized. All blocks marked as free.")

	// 3. Alokasikan blok untuk Root Directory:
	//    - Pastikan ROOT_DIR_BLOCK valid (tidak melebihi TOTAL_BLOCKS).
	//    - Set FAT[ROOT_DIR_BLOCK] menjadi FAT_EOF (karena root dir awalnya hanya 1 blok dan itu blok terakhirnya).
	if ROOT_DIR_BLOCK >= BlockID(TOTAL_BLOCKS) || ROOT_DIR_BLOCK < 0 {
		return errors.New("invalid ROOT_DIR_BLOCK configuration")
	}
	FAT[ROOT_DIR_BLOCK] = FAT_EOF
	fmt.Printf("Block %d allocated for Root Directory and marked as EOF in FAT.\n", ROOT_DIR_BLOCK)

	// 4. Buat entri "." (direktori saat ini) untuk Root Directory:
	//    - Buat instance DirectoryEntry.
	//    - Isi field-fieldnya:
	//        Name: "." (gunakan helper untuk konversi string ke [MAX_FILENAME_LEN]byte)
	//        Type: TYPE_DIRECTORY
	//        StartBlock: ROOT_DIR_BLOCK (karena "." dari root menunjuk ke root itu sendiri)
	//        Size: 0 (atau bisa dihitung nanti berdasarkan jumlah entri)
	//        ModTime: time.Now().UnixNano()
	var dotEntry DirectoryEntry
	copy(dotEntry.Name[:], ".") // Salin string ke array byte
	dotEntry.Type = TYPE_DIRECTORY
	dotEntry.StartBlock = ROOT_DIR_BLOCK
	dotEntry.Size = 0 // Untuk direktori, size bisa berarti jumlah entri atau ukuran data entri
	dotEntry.ModTime = time.Now().UnixNano()

	// 5. Buat entri ".." (parent directory) untuk Root Directory:
	//    - Buat instance DirectoryEntry.
	//    - Isi field-fieldnya:
	//        Name: ".."
	//        Type: TYPE_DIRECTORY
	//        StartBlock: ROOT_DIR_BLOCK (karena parent dari root adalah root itu sendiri dalam simulasi ini)
	//        Size: 0
	//        ModTime: time.Now().UnixNano()
	var dotDotEntry DirectoryEntry
	copy(dotDotEntry.Name[:], "..")
	dotDotEntry.Type = TYPE_DIRECTORY
	dotDotEntry.StartBlock = ROOT_DIR_BLOCK // Parent dari root adalah root
	dotDotEntry.Size = 0
	dotDotEntry.ModTime = time.Now().UnixNano()

	// 6. Serialize entri "." dan ".." menjadi byte.
	dotBytes, err := dotEntry.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize '.' entry: %w", err)
	}
	dotDotBytes, err := dotDotEntry.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize '..' entry: %w", err)
	}

	// 7. Tulis byte hasil serialisasi ke blok data Root Directory (Disk[ROOT_DIR_BLOCK]):
	//    - Entri pertama (dotBytes) ditulis mulai dari byte ke-0 di Disk[ROOT_DIR_BLOCK].
	//    - Entri kedua (dotDotBytes) ditulis setelah entri pertama.
	//    - Pastikan tidak melebihi BLOCK_SIZE.
	offset := 0
	if offset+len(dotBytes) > BLOCK_SIZE {
		return errors.New("block size too small for '.' entry")
	}
	copy(Disk[ROOT_DIR_BLOCK][offset:], dotBytes)
	offset += len(dotBytes)
	fmt.Printf("Serialized '.' entry (size %d) written to Root Directory block.\n", len(dotBytes))

	if offset+len(dotDotBytes) > BLOCK_SIZE {
		return errors.New("block size too small for '..' entry after '.' entry")
	}
	copy(Disk[ROOT_DIR_BLOCK][offset:], dotDotBytes)
	fmt.Printf("Serialized '..' entry (size %d) written to Root Directory block after '.' entry.\n", len(dotDotBytes))

	// Kita juga perlu menandai ukuran direktori root berdasarkan entri yang ada
	// Misalnya, rootEntry.Size = int64(2 * DIRECTORY_ENTRY_SIZE)
	// Tapi ini akan dikelola oleh fungsi yang memanipulasi direktori nanti.
	// Untuk format, cukup entri . dan .. ada.

	fmt.Println("Disk formatting complete. Root directory initialized with '.' and '..' entries.")
	return nil
}

// Fungsi helper untuk NewFileSystem agar bisa dipanggil dari main.go
type FileSystem struct {
	CurrentDirectoryBlock BlockID
	// Bisa ditambahkan field lain jika perlu state global
}

func NewFileSystem() (*FileSystem, error) {
	err := FormatDisk()
	if err != nil {
		return nil, fmt.Errorf("failed to format disk during NewFileSystem: %w", err)
	}
	fs := &FileSystem{
		CurrentDirectoryBlock: ROOT_DIR_BLOCK,
	}
	return fs, nil
}

// ListEntries: Membaca semua DirectoryEntry dari sebuah direktori.
// Input: directoryStartBlock adalah nomor blok pertama dari direktori yang ingin dibaca.
// Output: Slice dari DirectoryEntry yang ada di direktori tersebut, dan error jika ada.
func ListEntries(directoryStartBlock BlockID) ([]DirectoryEntry, error) {
	// 1. Buat slice kosong untuk menampung hasil DirectoryEntry.
	//    Ini adalah daftar file/folder yang akan kita kembalikan.
	var entries []DirectoryEntry

	// 2. Validasi Awal: Pastikan directoryStartBlock tidak FAT_FREE.
	//    Jika start block-nya saja sudah kosong, berarti direktori ini tidak valid atau kosong.
	if directoryStartBlock == FAT_FREE {
		// Bisa dianggap sebagai direktori kosong atau error, tergantung desain.
		// Untuk sekarang, kita kembalikan slice kosong saja, menandakan tidak ada entri.
		return entries, nil
	}
	if directoryStartBlock < 0 || directoryStartBlock >= BlockID(TOTAL_BLOCKS) {
		return nil, fmt.Errorf("blok awal direktori tidak valid: %d", directoryStartBlock)
	}

	// 3. Iterasi Melalui Rantai Blok Direktori di FAT:
	//    Sebuah direktori (seperti file) bisa saja memakan lebih dari satu blok jika isinya banyak.
	//    Kita perlu mengikuti rantai blok di FAT.
	currentBlock := directoryStartBlock
	for currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		// a. Validasi currentBlock (lagi, untuk keamanan tambahan di dalam loop)
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			return entries, fmt.Errorf("ditemukan nomor blok tidak valid (%d) dalam rantai direktori", currentBlock)
		}

		// b. Ambil data byte dari blok disk saat ini.
		//    Disk[currentBlock] adalah []byte yang berisi data mentah dari blok tersebut.
		blockData := Disk[currentBlock]

		// c. Iterasi di dalam satu blok untuk membaca setiap DirectoryEntry.
		//    Setiap DirectoryEntry punya ukuran DIRECTORY_ENTRY_SIZE byte.
		//    Kita akan membaca blok ini per DIRECTORY_ENTRY_SIZE byte.
		for offset := 0; offset+DIRECTORY_ENTRY_SIZE <= BLOCK_SIZE; offset += DIRECTORY_ENTRY_SIZE {
			// i. Ambil satu potong data seukuran DirectoryEntry.
			entryData := blockData[offset : offset+DIRECTORY_ENTRY_SIZE] // ii. Cek apakah entri ini "kosong" atau dihapus (invaliated).
			//     Konvensi sederhana: jika byte pertama dari nama adalah 0,
			//     kita anggap itu entri kosong / tidak terpakai / dihapus.
			//     Ini penting agar kita tidak membaca data sampah.
			if entryData[0] == 0 {
				// Jika kita menemukan entri yang namanya dimulai dengan byte 0,
				// ini adalah entri kosong atau dihapus, tapi masih mungkin ada entri valid lain setelahnya.
				// Kita lewati entri ini dan lanjutkan ke entri berikutnya dalam blok yang sama.
				continue
			} // iii. Deserialize data byte menjadi struct DirectoryEntry.
			entry, err := DeserializeEntry(entryData)
			if err != nil {
				// Jika ada error saat deserialize satu entri, kita bisa memilih untuk
				// mengabaikannya dan lanjut, atau mengembalikan error.
				// Untuk sekarang, kita catat errornya dan lanjut ke blok berikutnya (jika ada).
				// Atau, bisa juga return error:
				// return entries, fmt.Errorf("gagal deserialize entri di blok %d offset %d: %w", currentBlock, offset, err)
				fmt.Printf("Warning: Gagal deserialize entri di blok %d offset %d: %v. Mungkin akhir dari data valid.\n", currentBlock, offset, err)
				continue // Skip this entry and continue with the next one
			}

			// iv. Tambahkan entri yang berhasil di-deserialize ke slice hasil.
			entries = append(entries, entry)
		}

		// Label untuk 'goto' agar bisa lanjut ke blok berikutnya
		// d. Ambil nomor blok berikutnya dari FAT untuk direktori ini.
		currentBlock = FAT[currentBlock] // Pindah ke blok selanjutnya dalam rantai
	} // Akhir dari loop 'for currentBlock'

	// 4. Kembalikan daftar entri yang sudah terkumpul.
	return entries, nil
}

// filesystem_logic.go
// (Lanjutan dari kode sebelumnya)

// findFreeBlock: Mencari blok kosong pertama di FAT.
// Mengembalikan BlockID dari blok kosong tersebut, atau error jika tidak ada blok kosong (disk penuh).
func findFreeBlock() (BlockID, error) {
	// Kita iterasi melalui seluruh FAT.
	// Ingat, blok 0 bisa jadi punya arti khusus (misalnya superblok) atau tidak digunakan.
	// Di desain kita, ROOT_DIR_BLOCK adalah 1. Kita bisa mulai cari dari blok setelah itu,
	// atau dari awal jika blok 0 juga bisa dipakai untuk data umum.
	// Untuk sederhana, kita cari dari blok ke-0. Jika ada blok khusus,
	// kita harus pastikan tidak mengalokasikannya secara tidak sengaja di sini.
	// Mari kita asumsikan untuk saat ini, blok 0 bisa saja dipakai jika FAT_FREE.
	// Namun, lebih aman untuk memulai pencarian dari blok setelah yang sudah pasti dipakai (misal, setelah ROOT_DIR_BLOCK).
	// Untuk fungsi umum findFreeBlock, iterasi dari awal FAT itu logis.
	// Kita akan skip blok 0 jika itu adalah SUPER_BLOCK_ID atau semacamnya.
	// Karena ROOT_DIR_BLOCK kita = 1, kita bisa mulai cari dari blok 2, atau bahkan 0 jika FAT[0] bisa FAT_FREE.
	// Mari kita cari dari semua blok untuk generalitas.
	for i := BlockID(0); i < BlockID(TOTAL_BLOCKS); i++ {
		if FAT[i] == FAT_FREE {
			// Ditemukan blok kosong!
			return i, nil // Kembalikan nomor bloknya
		}
	}
	// Jika loop selesai dan tidak ada blok kosong yang ditemukan, berarti disk penuh.
	return -1, errors.New("disk penuh, tidak ada blok kosong ditemukan")
}

// filesystem_logic.go
// (Lanjutan dari kode sebelumnya)

// addEntryToDirectory: Menambahkan sebuah DirectoryEntry baru ke dalam direktori induk.
// Ia akan mencari slot kosong di blok-blok data direktori induk.
// Untuk saat ini, TIDAK menangani kasus jika direktori induk perlu blok baru (itu fitur lanjutan).
func addEntryToDirectory(parentDirStartBlock BlockID, newEntry DirectoryEntry) error {
	if parentDirStartBlock < 0 || parentDirStartBlock >= BlockID(TOTAL_BLOCKS) || FAT[parentDirStartBlock] == FAT_FREE {
		return errors.New("blok awal direktori induk tidak valid atau belum dialokasikan")
	}

	// Serialize entri baru menjadi byte
	entryBytes, err := newEntry.Serialize()
	if err != nil {
		return fmt.Errorf("gagal serialize entri baru: %w", err)
	}
	if len(entryBytes) != DIRECTORY_ENTRY_SIZE { // Validasi tambahan
		return fmt.Errorf("ukuran byte entri baru (%d) tidak sesuai dengan DIRECTORY_ENTRY_SIZE (%d)", len(entryBytes), DIRECTORY_ENTRY_SIZE)
	}

	// Iterasi melalui rantai blok direktori induk
	currentBlock := parentDirStartBlock
	for currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			return fmt.Errorf("nomor blok tidak valid (%d) dalam rantai direktori induk", currentBlock)
		}

		blockData := Disk[currentBlock] // Ambil data dari blok saat ini

		// Cari slot kosong di dalam blok ini
		for offset := 0; offset+DIRECTORY_ENTRY_SIZE <= BLOCK_SIZE; offset += DIRECTORY_ENTRY_SIZE {
			// Cek apakah slot ini kosong (nama dimulai dengan byte 0)
			potentialEmptySlot := blockData[offset : offset+DIRECTORY_ENTRY_SIZE]
			if potentialEmptySlot[0] == 0 { // Byte pertama dari nama adalah 0, berarti slot kosong
				// Ditemukan slot kosong! Tulis entri baru di sini.
				copy(Disk[currentBlock][offset:], entryBytes) // Salin byte entri baru ke disk
				fmt.Printf("Entri '%s' ditambahkan ke blok %d direktori induk, offset %d.\n",
					string(newEntry.Name[:bytes.IndexByte(newEntry.Name[:], 0)]), parentDirStartBlock, offset)
				// Kita juga perlu update ModTime direktori induk
				// Ini bisa dilakukan oleh fungsi yang memanggil addEntryToDirectory, atau di sini
				// (Untuk sekarang kita skip update ModTime induk agar sederhana)
				return nil // Berhasil menambahkan entri
			}
		}
		// Jika blok ini penuh (tidak ada slot kosong), pindah ke blok berikutnya dari direktori induk
		// prevBlock := currentBlock // Commented out - will be used in future implementations
		currentBlock = FAT[currentBlock]

		// FITUR LANJUTAN (BELUM DIIMPLEMENTASIKAN DI SINI):
		// Jika currentBlock sekarang FAT_EOF (artinya blok prevBlock adalah yang terakhir dan penuh),
		// dan kita masih belum menemukan slot, kita seharusnya:
		// 1. Cari blok kosong baru dengan findFreeBlock().
		// 2. Alokasikan blok baru itu di FAT, dengan FAT[prevBlock] = blokBaru, dan FAT[blokBaru] = FAT_EOF.
		// 3. Kemudian tulis entri baru kita ke blokBaru tersebut.
		// Untuk versi saat ini, jika semua blok yang ada penuh, kita akan error.
		if currentBlock == FAT_EOF {
			// Kode untuk mengalokasikan blok baru untuk direktori induk akan ada di sini
			// Untuk sementara, kita anggap ini error "direktori induk penuh dan tidak bisa diperluas otomatis"
			return errors.New("direktori induk penuh, tidak dapat menambahkan entri baru (fitur perluasan blok direktori belum ada)")
		}
	}
	// Jika loop selesai karena currentBlock menjadi FAT_FREE (seharusnya tidak terjadi jika FAT dikelola dengan baik)
	// atau kondisi lain yang tidak terduga.
	return errors.New("tidak dapat menemukan slot untuk menambahkan entri di direktori induk (mungkin rantai FAT rusak atau direktori penuh)")
}

// filesystem_logic.go
// (Lanjutan dari kode sebelumnya)

// CreateDirectory: Membuat direktori baru di dalam parentDirStartBlock.
func CreateDirectory(parentDirStartBlock BlockID, newDirName string) error {
	// 1. Validasi Nama Direktori Baru
	if len(newDirName) == 0 {
		return errors.New("nama direktori tidak boleh kosong")
	}
	if len(newDirName) > MAX_FILENAME_LEN {
		return fmt.Errorf("nama direktori terlalu panjang (maks %d karakter)", MAX_FILENAME_LEN)
	}
	// (Bisa ditambahkan validasi karakter ilegal jika perlu)

	// 2. Cek Apakah Nama Sudah Ada di Direktori Induk
	//    Kita gunakan ListEntries yang sudah kita buat!
	parentEntries, err := ListEntries(parentDirStartBlock)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori induk: %w", err)
	}
	for _, entry := range parentEntries {
		// Konversi nama dari byte array ke string untuk perbandingan
		entryName := string(entry.Name[:bytes.IndexByte(entry.Name[:], 0)]) // Berhenti di null terminator
		if entryName == newDirName {
			return fmt.Errorf("direktori atau file dengan nama '%s' sudah ada", newDirName)
		}
	}

	// 3. Cari Blok Kosong untuk Data Direktori Baru
	newDirDataBlock, err := findFreeBlock()
	if err != nil {
		return fmt.Errorf("gagal membuat direktori (tidak ada blok kosong): %w", err)
	}

	// 4. Alokasikan Blok Tersebut di FAT untuk Direktori Baru
	//    Direktori baru awalnya hanya 1 blok dan itu blok terakhirnya.
	FAT[newDirDataBlock] = FAT_EOF
	fmt.Printf("Blok %d dialokasikan untuk direktori baru '%s'.\n", newDirDataBlock, newDirName)

	// 5. Buat dan Tulis Entri "." dan ".." untuk Direktori Baru Ini
	//    a. Entri "." (menunjuk ke dirinya sendiri)
	var dotEntry DirectoryEntry
	copy(dotEntry.Name[:], ".")
	dotEntry.Type = TYPE_DIRECTORY
	dotEntry.StartBlock = newDirDataBlock           // Menunjuk ke blok data direktori baru ini
	dotEntry.Size = int64(2 * DIRECTORY_ENTRY_SIZE) // Awalnya berisi . dan ..
	dotEntry.ModTime = time.Now().UnixNano()
	dotBytes, _ := dotEntry.Serialize() // Error handling diabaikan untuk ringkas, idealnya dicek

	//    b. Entri ".." (menunjuk ke direktori induknya)
	var dotDotEntry DirectoryEntry
	copy(dotDotEntry.Name[:], "..")
	dotDotEntry.Type = TYPE_DIRECTORY
	dotDotEntry.StartBlock = parentDirStartBlock // Menunjuk ke blok awal direktori induk
	dotDotEntry.Size = 0                         // Size untuk ".." bisa 0 atau size induk, untuk simpel 0 dulu
	dotDotEntry.ModTime = time.Now().UnixNano()
	dotDotBytes, _ := dotDotEntry.Serialize() // Error handling diabaikan

	//    c. Tulis kedua entri ini ke blok data direktori baru (Disk[newDirDataBlock])
	offset := 0
	copy(Disk[newDirDataBlock][offset:], dotBytes)
	offset += len(dotBytes)
	copy(Disk[newDirDataBlock][offset:], dotDotBytes)
	offset += len(dotDotBytes)
	fmt.Printf("Entri '.' dan '..' ditulis ke blok data direktori '%s'.\n", newDirName)
	// --- TAMBAHAN BARU: Tandai akhir entri di blok direktori baru ---
	// Jika masih ada ruang di blok setelah entri "." dan "..",
	// pastikan byte pertama dari slot entri berikutnya adalah 0.
	// Ini menandakan ke ListEntries bahwa tidak ada entri lagi di blok ini.
	if offset < BLOCK_SIZE {
		// Inicializa seluruh sisa blok dengan 0 untuk memastikan tidak ada data sampah
		for i := offset; i < BLOCK_SIZE; i++ {
			Disk[newDirDataBlock][i] = 0
		}
	}
	// --- AKHIR TAMBAHAN BARU ---

	// 6. Buat DirectoryEntry untuk Direktori Baru Ini (yang akan disimpan di direktori induk)
	var dirEntryForParent DirectoryEntry
	copy(dirEntryForParent.Name[:], newDirName)
	dirEntryForParent.Type = TYPE_DIRECTORY
	dirEntryForParent.StartBlock = newDirDataBlock           // Menunjuk ke blok data yang baru dialokasikan
	dirEntryForParent.Size = int64(2 * DIRECTORY_ENTRY_SIZE) // Ukuran awal karena ada . dan ..
	dirEntryForParent.ModTime = time.Now().UnixNano()

	// 7. Tambahkan Entri Direktori Baru Ini ke Direktori Induk
	err = addEntryToDirectory(parentDirStartBlock, dirEntryForParent)
	if err != nil {
		// Jika gagal menambahkan ke induk (misalnya induk penuh),
		// kita idealnya harus membatalkan alokasi newDirDataBlock di FAT (rollback).
		// Untuk sekarang, kita hanya kembalikan error.
		FAT[newDirDataBlock] = FAT_FREE // Rollback sederhana: bebaskan lagi bloknya
		return fmt.Errorf("gagal menambahkan entri direktori '%s' ke induk: %w", newDirName, err)
	}

	fmt.Printf("Direktori '%s' berhasil dibuat.\n", newDirName)
	return nil
}

// filesystem_logic.go
// (Lanjutan dari kode sebelumnya)

// CreateFile: Membuat file baru di dalam parentDirStartBlock.
func CreateFile(parentDirStartBlock BlockID, newFileName string) error {
	// 1. Validasi Nama File Baru
	if len(newFileName) == 0 {
		return errors.New("nama file tidak boleh kosong")
	}
	if len(newFileName) > MAX_FILENAME_LEN {
		return fmt.Errorf("nama file terlalu panjang (maks %d karakter)", MAX_FILENAME_LEN)
	}
	// (Bisa ditambahkan validasi karakter ilegal jika perlu, misal '/')

	// 2. Cek Apakah Nama Sudah Ada di Direktori Induk
	//    Gunakan ListEntries yang sudah ada.
	parentEntries, err := ListEntries(parentDirStartBlock)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori induk saat membuat file: %w", err)
	}
	for _, entry := range parentEntries {
		entryName := string(entry.Name[:bytes.IndexByte(entry.Name[:], 0)]) // Konversi nama ke string
		if entryName == newFileName {
			return fmt.Errorf("file atau direktori dengan nama '%s' sudah ada", newFileName)
		}
	}

	// 3. Cari Blok Kosong untuk Data Awal File Baru
	//    Meskipun file awalnya 0 byte, kita alokasikan 1 blok untuknya dan tandai EOF.
	//    Ini akan mempermudah operasi tulis nanti dan memberikan StartBlock yang valid.
	newFileDataBlock, err := findFreeBlock()
	if err != nil {
		return fmt.Errorf("gagal membuat file (tidak ada blok kosong untuk data file): %w", err)
	}

	// 4. Alokasikan Blok Tersebut di FAT untuk File Baru
	//    File baru (kosong) hanya 1 blok (yang belum tentu diisi data) dan itu blok terakhirnya.
	FAT[newFileDataBlock] = FAT_EOF
	fmt.Printf("Blok %d dialokasikan untuk file baru '%s'.\n", newFileDataBlock, newFileName)

	// 5. Buat DirectoryEntry untuk File Baru Ini (yang akan disimpan di direktori induk)
	var fileEntryForParent DirectoryEntry
	copy(fileEntryForParent.Name[:], newFileName)      // Salin nama file
	fileEntryForParent.Type = TYPE_FILE                // Set tipe sebagai FILE
	fileEntryForParent.StartBlock = newFileDataBlock   // Menunjuk ke blok data yang baru dialokasikan
	fileEntryForParent.Size = 0                        // File baru ukurannya 0 byte
	fileEntryForParent.ModTime = time.Now().UnixNano() // Waktu modifikasi saat ini

	// 6. Tambahkan Entri File Baru Ini ke Direktori Induk
	//    Gunakan fungsi addEntryToDirectory yang sudah kita buat.
	err = addEntryToDirectory(parentDirStartBlock, fileEntryForParent)
	if err != nil {
		// Jika gagal menambahkan ke induk (misalnya induk penuh),
		// batalkan alokasi newFileDataBlock di FAT (rollback).
		FAT[newFileDataBlock] = FAT_FREE // Bebaskan lagi bloknya
		return fmt.Errorf("gagal menambahkan entri file '%s' ke direktori induk: %w", newFileName, err)
	}

	fmt.Printf("File '%s' berhasil dibuat.\n", newFileName)
	return nil
}

// freeBlockChain: Membebaskan rantai blok di FAT mulai dari startBlock.
// Semua blok dalam rantai akan di-set menjadi FAT_FREE.
func freeBlockChain(startBlock BlockID) error {
	if startBlock < 0 || startBlock >= BlockID(TOTAL_BLOCKS) {
		// Jika startBlock adalah FAT_EOF atau FAT_FREE atau tidak valid, anggap tidak ada yang perlu dibebaskan.
		// FAT_EOF (-1) dan FAT_FREE (-2) memang < 0.
		if startBlock == FAT_EOF || startBlock == FAT_FREE {
			return nil
		}
		return fmt.Errorf("startBlock (%d) tidak valid untuk freeBlockChain", startBlock)
	}

	currentBlock := startBlock
	for currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			// Seharusnya tidak terjadi jika FAT konsisten, tapi sebagai pengaman
			return fmt.Errorf("ditemukan blok tidak valid (%d) saat membebaskan rantai", currentBlock)
		}
		nextBlock := FAT[currentBlock]
		FAT[currentBlock] = FAT_FREE // Bebaskan blok saat ini
		// fmt.Printf("Blok %d dibebaskan.\n", currentBlock) // Untuk debug
		currentBlock = nextBlock
	}
	return nil
}

// updateEntryInDirectory: Mengupdate DirectoryEntry yang sudah ada di direktori induk.
// Mencari entri dengan nama yang sama dan menimpanya dengan updatedEntry.
func updateEntryInDirectory(parentDirStartBlock BlockID, updatedEntry DirectoryEntry) error {
	if parentDirStartBlock < 0 || parentDirStartBlock >= BlockID(TOTAL_BLOCKS) || FAT[parentDirStartBlock] == FAT_FREE {
		return errors.New("blok awal direktori induk tidak valid atau belum dialokasikan untuk update")
	}

	updatedEntryBytes, err := updatedEntry.Serialize()
	if err != nil {
		return fmt.Errorf("gagal serialize updatedEntry: %w", err)
	}

	updatedEntryName := string(updatedEntry.Name[:bytes.IndexByte(updatedEntry.Name[:], 0)])

	currentBlock := parentDirStartBlock
	for currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			return fmt.Errorf("nomor blok tidak valid (%d) dalam rantai direktori induk saat update", currentBlock)
		}

		blockData := Disk[currentBlock]
		for offset := 0; offset+DIRECTORY_ENTRY_SIZE <= BLOCK_SIZE; offset += DIRECTORY_ENTRY_SIZE {
			entryData := blockData[offset : offset+DIRECTORY_ENTRY_SIZE]
			if entryData[0] == 0 { // Slot kosong, berarti entri yang dicari tidak ada di sisa blok ini
				goto nextBlockInUpdate // Lanjut ke blok berikutnya jika ada
			}

			// Deserialize untuk perbandingan nama
			existingEntry, errDeserialize := DeserializeEntry(entryData)
			if errDeserialize != nil {
				// Abaikan entri yang rusak, lanjutkan pencarian
				fmt.Printf("Warning: Gagal deserialize entri di blok %d offset %d saat update: %v\n", currentBlock, offset, errDeserialize)
				continue
			}

			existingEntryName := string(existingEntry.Name[:bytes.IndexByte(existingEntry.Name[:], 0)])

			if existingEntryName == updatedEntryName {
				// Ditemukan entri yang cocok! Timpa dengan data baru.
				copy(Disk[currentBlock][offset:], updatedEntryBytes)
				// fmt.Printf("Entri '%s' diupdate di blok %d direktori induk, offset %d.\n", updatedEntryName, currentBlock, offset)
				return nil // Berhasil update
			}
		}
	nextBlockInUpdate:
		currentBlock = FAT[currentBlock]
	}

	return fmt.Errorf("entri dengan nama '%s' tidak ditemukan di direktori induk untuk diupdate", updatedEntryName)
}

// WriteToFile: Menulis data ke sebuah file. Mode saat ini adalah OVERWRITE.
// Membebaskan blok lama, lalu mengalokasikan blok baru sesuai kebutuhan data.
func WriteToFile(fileEntry *DirectoryEntry, parentDirStartBlock BlockID, dataToWrite []byte) error {
	// 1. Validasi Awal
	if fileEntry == nil {
		return errors.New("fileEntry tidak boleh nil")
	}
	if fileEntry.Type != TYPE_FILE {
		return errors.New("hanya bisa menulis ke entri bertipe FILE")
	}
	if parentDirStartBlock < 0 || parentDirStartBlock >= BlockID(TOTAL_BLOCKS) || FAT[parentDirStartBlock] == FAT_FREE {
		return errors.New("blok awal direktori induk tidak valid atau belum dialokasikan untuk file")
	}

	fileNameForLog := string(fileEntry.Name[:bytes.IndexByte(fileEntry.Name[:], 0)])
	fmt.Printf("Menulis ke file '%s'. Ukuran data: %d bytes.\n", fileNameForLog, len(dataToWrite))

	// 2. Bebaskan Blok Lama yang Mungkin Digunakan File Ini (Mode Overwrite)
	//    fileEntry.StartBlock menyimpan blok pertama dari data file lama.
	//    Jika fileEntry.StartBlock adalah FAT_FREE atau FAT_EOF, berarti file belum punya blok data.
	if fileEntry.StartBlock != FAT_FREE && fileEntry.StartBlock != FAT_EOF {
		// fmt.Printf("Membebaskan blok lama dari file '%s' mulai dari blok %d.\n", fileNameForLog, fileEntry.StartBlock)
		err := freeBlockChain(fileEntry.StartBlock)
		if err != nil {
			return fmt.Errorf("gagal membebaskan blok lama file '%s': %w", fileNameForLog, err)
		}
	}
	fileEntry.StartBlock = FAT_EOF // Reset StartBlock, akan diisi jika ada data
	fileEntry.Size = 0             // Reset Size

	// 3. Jika tidak ada data untuk ditulis (misalnya, ingin membuat file kosong atau mengosongkan file)
	if len(dataToWrite) == 0 {
		fmt.Printf("Tidak ada data untuk ditulis ke '%s'. File akan menjadi kosong.\n", fileNameForLog)
		// StartBlock sudah FAT_EOF, Size sudah 0. Tinggal update ModTime.
		fileEntry.ModTime = time.Now().UnixNano()
		// Update entri ini di direktori induknya
		errUpdate := updateEntryInDirectory(parentDirStartBlock, *fileEntry)
		if errUpdate != nil {
			return fmt.Errorf("gagal update entri untuk file kosong '%s' di direktori induk: %w", fileNameForLog, errUpdate)
		}
		return nil // Selesai
	}

	// 4. Hitung Jumlah Blok yang Dibutuhkan
	//    (panjang_data + ukuran_blok - 1) / ukuran_blok  (integer division untuk pembulatan ke atas)
	numBlocksNeeded := (len(dataToWrite) + BLOCK_SIZE - 1) / BLOCK_SIZE
	// fmt.Printf("Data membutuhkan %d blok.\n", numBlocksNeeded)

	var allocatedBlocks []BlockID        // Untuk menyimpan daftar blok yang berhasil dialokasikan
	var firstBlockOfFile = FAT_EOF       // Akan diisi dengan blok pertama yang dialokasikan
	var previousAllocatedBlock = FAT_EOF // Untuk chaining di FAT

	// 5. Alokasikan Blok Baru dan Tulis Data per Blok
	for i := 0; i < numBlocksNeeded; i++ {
		newBlock, err := findFreeBlock()
		if err != nil {
			// Gagal alokasi blok. Perlu rollback: bebaskan semua blok yang sudah dialokasikan di loop ini.
			for _, allocatedBlock := range allocatedBlocks {
				FAT[allocatedBlock] = FAT_FREE
			}
			return fmt.Errorf("disk penuh saat mencoba alokasi blok ke-%d untuk file '%s': %w", i+1, fileNameForLog, err)
		}

		FAT[newBlock] = FAT_EOF // Awalnya, setiap blok baru adalah EOF sampai ada blok berikutnya
		allocatedBlocks = append(allocatedBlocks, newBlock)
		// fmt.Printf("Blok %d dialokasikan untuk file '%s'.\n", newBlock, fileNameForLog)

		if i == 0 {
			firstBlockOfFile = newBlock
		}

		if previousAllocatedBlock != FAT_EOF {
			FAT[previousAllocatedBlock] = newBlock // Hubungkan blok sebelumnya ke blok baru ini
		}
		previousAllocatedBlock = newBlock

		// Tentukan bagian data yang akan ditulis ke blok ini
		startByte := i * BLOCK_SIZE
		endByte := (i + 1) * BLOCK_SIZE
		if endByte > len(dataToWrite) {
			endByte = len(dataToWrite)
		}
		dataChunk := dataToWrite[startByte:endByte]

		// Salin dataChunk ke Disk[newBlock]
		// Pastikan blok di-clear dulu jika ada sisa data lama (meskipun findFreeBlock seharusnya mengembalikan blok "bersih")
		// copy sudah menimpa, jadi tidak perlu clear manual jika blok baru.
		copy(Disk[newBlock][:len(dataChunk)], dataChunk) // Hanya salin sejumlah dataChunk
		// Jika len(dataChunk) < BLOCK_SIZE, sisa Disk[newBlock] akan tetap 0 (jika blok baru) atau data lama (jika blok dipakai ulang).
		// Untuk keamanan, kita bisa clear sisa bloknya:
		if len(dataChunk) < BLOCK_SIZE {
			for k := len(dataChunk); k < BLOCK_SIZE; k++ {
				Disk[newBlock][k] = 0 // Set sisa byte di blok menjadi 0
			}
		}
		// fmt.Printf("%d bytes ditulis ke blok %d.\n", len(dataChunk), newBlock)
	}

	// 6. Update Informasi di DirectoryEntry file
	fileEntry.StartBlock = firstBlockOfFile
	fileEntry.Size = int64(len(dataToWrite))
	fileEntry.ModTime = time.Now().UnixNano()

	// 7. Tulis Ulang (Update) DirectoryEntry yang Sudah Diperbarui ke Direktori Induk
	errUpdate := updateEntryInDirectory(parentDirStartBlock, *fileEntry)
	if errUpdate != nil {
		// Gagal update entri di induk. Perlu rollback: bebaskan semua blok yang baru dialokasikan.
		for _, allocatedBlock := range allocatedBlocks {
			FAT[allocatedBlock] = FAT_FREE
		}
		return fmt.Errorf("gagal update entri file '%s' di direktori induk setelah menulis data: %w", fileNameForLog, errUpdate)
	}

	fmt.Printf("Data berhasil ditulis ke file '%s'. Total blok dipakai: %d.\n", fileNameForLog, len(allocatedBlocks))
	return nil
}

// ReadFromFile: Membaca seluruh konten data dari sebuah file.
// Input: fileEntry adalah DirectoryEntry dari file yang ingin dibaca.
// Output: Slice byte yang berisi data file, dan error jika ada.
func ReadFromFile(fileEntry DirectoryEntry) ([]byte, error) {
	// 1. Validasi Awal
	if fileEntry.Type != TYPE_FILE {
		return nil, errors.New("hanya bisa membaca dari entri bertipe FILE")
	}

	fileNameForLog := string(fileEntry.Name[:bytes.IndexByte(fileEntry.Name[:], 0)])
	// fmt.Printf("Membaca dari file '%s'. Ukuran diharapkan: %d bytes, StartBlock: %d.\n",
	// 	fileNameForLog, fileEntry.Size, fileEntry.StartBlock)

	// 2. Handle File Kosong atau Belum Ada Data
	//    Jika ukuran file 0, atau StartBlock tidak menunjuk ke blok data yang valid,
	//    kembalikan slice byte kosong.
	if fileEntry.Size == 0 {
		// fmt.Printf("File '%s' kosong (size 0).\n", fileNameForLog)
		return []byte{}, nil // File kosong, tidak ada data untuk dibaca
	}
	if fileEntry.StartBlock == FAT_EOF || fileEntry.StartBlock == FAT_FREE ||
		fileEntry.StartBlock < 0 || fileEntry.StartBlock >= BlockID(TOTAL_BLOCKS) {
		// Jika StartBlock tidak valid tapi size > 0, ini kondisi aneh/inkonsisten.
		// Tapi untuk kasus umum file kosong yang StartBlock-nya FAT_EOF/FAT_FREE, ini benar.
		// fmt.Printf("File '%s' tidak memiliki blok data yang dialokasikan (StartBlock=%d).\n", fileNameForLog, fileEntry.StartBlock)
		if fileEntry.Size > 0 { // Inkonsistensi jika size > 0 tapi start block tidak valid
			return nil, fmt.Errorf("inkonsistensi metadata file '%s': size %d tapi StartBlock %d", fileNameForLog, fileEntry.Size, fileEntry.StartBlock)
		}
		return []byte{}, nil
	}

	// 3. Siapkan Buffer untuk Menampung Data Hasil Baca
	//    Kita gunakan bytes.Buffer untuk menggabungkan data dari beberapa blok.
	var fileDataBuffer bytes.Buffer
	bytesToRead := fileEntry.Size        // Berapa banyak byte lagi yang perlu kita baca
	currentBlock := fileEntry.StartBlock // Mulai dari blok pertama file

	// 4. Iterasi Melalui Rantai Blok File di FAT
	for bytesToRead > 0 && currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		// a. Validasi currentBlock (keamanan tambahan)
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			return nil, fmt.Errorf("ditemukan nomor blok tidak valid (%d) saat membaca file '%s'", currentBlock, fileNameForLog)
		}

		// b. Ambil data byte dari blok disk saat ini.
		blockData := Disk[currentBlock]

		// c. Tentukan berapa banyak byte yang akan dibaca dari blok ini.
		//    Bisa jadi sisa bytesToRead lebih kecil dari BLOCK_SIZE (jika ini blok terakhir).
		chunkSize := int64(BLOCK_SIZE)
		if bytesToRead < chunkSize {
			chunkSize = bytesToRead
		}

		// d. Tulis bagian data dari blok ini ke buffer hasil.
		//    Kita hanya mengambil sebanyak chunkSize dari blockData.
		_, err := fileDataBuffer.Write(blockData[:chunkSize])
		if err != nil {
			// Seharusnya tidak terjadi dengan bytes.Buffer, tapi baik untuk ada.
			return nil, fmt.Errorf("gagal menulis ke buffer saat membaca blok %d file '%s': %w", currentBlock, fileNameForLog, err)
		}
		// fmt.Printf("Membaca %d bytes dari blok %d untuk file '%s'.\n", chunkSize, currentBlock, fileNameForLog)

		// e. Update jumlah byte yang masih harus dibaca.
		bytesToRead -= chunkSize

		// f. Ambil nomor blok berikutnya dari FAT.
		currentBlock = FAT[currentBlock]
	} // Akhir dari loop 'for bytesToRead > 0 ...'

	// 5. Validasi Akhir: Apakah kita sudah membaca semua byte sesuai ukuran file?
	if bytesToRead > 0 {
		// Ini berarti rantai FAT berhenti (EOF atau FREE) sebelum kita selesai membaca semua data
		// sesuai dengan fileEntry.Size. Ini menandakan ada korupsi/inkonsistensi.
		fmt.Printf("Warning: File '%s' mungkin terpotong. Diharapkan %d bytes, tapi rantai FAT berhenti saat sisa %d bytes.\n",
			fileNameForLog, fileEntry.Size, bytesToRead)
		// Tergantung kebijakan, kita bisa kembalikan error atau data yang sudah terbaca sejauh ini.
		// Kita kembalikan yang sudah terbaca.
	}

	// 6. Kembalikan data yang sudah terkumpul dari buffer.
	// fmt.Printf("Selesai membaca file '%s'. Total bytes dibaca: %d.\n", fileNameForLog, fileDataBuffer.Len())
	return fileDataBuffer.Bytes(), nil
}

// invalidateEntryInParent: Menemukan entri dengan nama tertentu di direktori induk
// dan menandainya sebagai tidak valid/dihapus dengan mengubah Name[0] menjadi 0.
func invalidateEntryInParent(parentDirStartBlock BlockID, entryNameToInvalidate string) error {
	if parentDirStartBlock < 0 || parentDirStartBlock >= BlockID(TOTAL_BLOCKS) || FAT[parentDirStartBlock] == FAT_FREE {
		return errors.New("blok awal direktori induk tidak valid atau belum dialokasikan untuk invalidasi")
	}

	// Iterasi melalui rantai blok direktori induk
	currentBlock := parentDirStartBlock
	entryFoundAndInvalidated := false

	for currentBlock != FAT_EOF && currentBlock != FAT_FREE {
		if currentBlock < 0 || currentBlock >= BlockID(TOTAL_BLOCKS) {
			return fmt.Errorf("nomor blok tidak valid (%d) dalam rantai direktori induk saat invalidasi", currentBlock)
		}

		blockData := Disk[currentBlock] // Ambil data dari blok saat ini

		// Cari entri di dalam blok ini
		for offset := 0; offset+DIRECTORY_ENTRY_SIZE <= BLOCK_SIZE; offset += DIRECTORY_ENTRY_SIZE {
			entryDataSlice := blockData[offset : offset+DIRECTORY_ENTRY_SIZE] // Ini slice, jadi modifikasi akan ke Disk

			if entryDataSlice[0] == 0 { // Slot sudah kosong, tidak mungkin ini entri yang kita cari
				continue // Lanjut ke slot berikutnya
			}

			// Kita perlu deserialize sementara untuk cek nama, meskipun kita hanya akan modifikasi byte Name[0]
			// Ini agak tidak efisien, tapi paling mudah untuk sekarang.
			// Alternatif: langsung bandingkan byte nama.
			tempEntry, errDeserialize := DeserializeEntry(entryDataSlice)
			if errDeserialize != nil {
				// Abaikan entri yang rusak
				continue
			}

			currentEntryName := string(tempEntry.Name[:bytes.IndexByte(tempEntry.Name[:], 0)])

			if currentEntryName == entryNameToInvalidate {
				// Ditemukan entri yang cocok! Invalidate dengan set Name[0] = 0.
				// Kita modifikasi langsung slice yang merujuk ke Disk.
				Disk[currentBlock][offset] = 0 // Byte pertama dari nama di-set 0
				// Atau, jika ingin lebih "bersih" terhadap seluruh field nama (opsional):
				// for k := 0; k < MAX_FILENAME_LEN; k++ {
				// 	Disk[currentBlock][offset+k] = 0
				// }

				fmt.Printf("Entri '%s' diinvalidaasi dari blok %d direktori induk, offset %d.\n",
					entryNameToInvalidate, currentBlock, offset)
				entryFoundAndInvalidated = true
				// Kita bisa 'return nil' di sini jika yakin nama unik.
				// Atau lanjutkan loop jika ada kemungkinan nama duplikat (seharusnya tidak ada di desain kita).
				// Untuk sekarang, kita anggap nama unik dan langsung keluar setelah invalidasi.
				// Jika direktori bisa multi-blok, kita mungkin perlu 'goto endInvalidateLoop' setelah ini.
				// Tapi karena kita akan return, tidak masalah.
				return nil
			}
		}
		// Pindah ke blok berikutnya dari direktori induk
		currentBlock = FAT[currentBlock]
	}

	if !entryFoundAndInvalidated {
		return fmt.Errorf("entri dengan nama '%s' tidak ditemukan di direktori induk untuk diinvalidasi", entryNameToInvalidate)
	}
	return nil // Seharusnya sudah return di dalam loop jika ketemu
}

// filesystem_logic.go
// (Lanjutan dari kode sebelumnya)

// DeleteEntry: Menghapus file atau direktori (kosong).
func DeleteEntry(parentDirStartBlock BlockID, entryName string) error {
	// 1. Validasi Nama
	if len(entryName) == 0 {
		return errors.New("nama entri untuk dihapus tidak boleh kosong")
	}
	if entryName == "." || entryName == ".." {
		return errors.New("tidak dapat menghapus entri '.' atau '..'")
	}

	// 2. Cari Entri yang Akan Dihapus di Direktori Induk
	parentEntries, err := ListEntries(parentDirStartBlock)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori induk saat mencari entri '%s': %w", entryName, err)
	}

	var entryToDelete DirectoryEntry
	found := false
	for _, entry := range parentEntries {
		currentEntryName := string(entry.Name[:bytes.IndexByte(entry.Name[:], 0)])
		if currentEntryName == entryName {
			entryToDelete = entry // Salin structnya
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("entri '%s' tidak ditemukan di direktori induk", entryName)
	}

	// 3. Proses Berdasarkan Tipe Entri
	if entryToDelete.Type == TYPE_FILE {
		// Jika file, bebaskan rantai blok datanya
		fmt.Printf("Menghapus file '%s'. Membebaskan blok mulai dari %d.\n", entryName, entryToDelete.StartBlock)
		err = freeBlockChain(entryToDelete.StartBlock)
		if err != nil {
			return fmt.Errorf("gagal membebaskan blok data file '%s': %w", entryName, err)
		}
		// Set StartBlock ke FAT_FREE atau FAT_EOF untuk menandakan tidak ada blok lagi
		// Ini tidak perlu karena entri akan diinvalidasi. Metadata lama tidak masalah.
	} else if entryToDelete.Type == TYPE_DIRECTORY {
		// Jika direktori, cek apakah kosong (hanya berisi "." dan "..")
		fmt.Printf("Mencoba menghapus direktori '%s' (StartBlock: %d).\n", entryName, entryToDelete.StartBlock)

		// Pastikan StartBlock direktori yang akan dihapus itu valid sebelum ListEntries
		if entryToDelete.StartBlock == FAT_FREE || entryToDelete.StartBlock == FAT_EOF ||
			entryToDelete.StartBlock < 0 || entryToDelete.StartBlock >= BlockID(TOTAL_BLOCKS) {
			// Ini kasus aneh, direktori tanpa blok data yang valid. Anggap "kosong" dan bisa dihapus entrinya.
			fmt.Printf("Direktori '%s' tidak memiliki blok data valid, dianggap kosong.\n", entryName)
		} else {
			subEntries, errListSub := ListEntries(entryToDelete.StartBlock)
			if errListSub != nil {
				return fmt.Errorf("gagal membaca isi direktori '%s' untuk pemeriksaan kekosongan: %w", entryName, errListSub)
			}
			// --- TAMBAHKAN DEBUG PRINT DI SINI ---
			fmt.Printf("DEBUG: Isi subEntries untuk '%s':\n", entryName)
			for k, se := range subEntries {
				seNameIdx := bytes.IndexByte(se.Name[:], 0)
				var seNameStr string
				if seNameIdx == -1 {
					seNameStr = string(se.Name[:])
				} else {
					seNameStr = string(se.Name[:seNameIdx])
				}
				fmt.Printf("  SubEntri %d: Nama='%s', Tipe=%d\n", k, seNameStr, se.Type)
			}
			// --- AKHIR DEBUG PRINT ---			// Direktori kosong jika hanya ada "." dan ".." atau tidak ada sama sekali (seharusnya minimal . dan .. jika sudah diinisialisasi)
			// Kita hitung entri yang BUKAN "." atau ".."
			realEntryCount := 0
			for _, subEntry := range subEntries {
				// Ambil nama dengan aman (INI BAGIAN YANG DIPERBAIKI)
				idx := bytes.IndexByte(subEntry.Name[:], 0)
				var subEntryName string
				if idx == -1 { // Jika tidak ada null byte (nama mengisi seluruh array)
					// Ini seharusnya tidak terjadi jika MAX_FILENAME_LEN > panjang nama aktual seperti "." atau ".."
					// Tapi sebagai pengaman:
					subEntryName = string(subEntry.Name[:])
				} else {
					subEntryName = string(subEntry.Name[:idx])
				}

				if subEntryName != "." && subEntryName != ".." {
					realEntryCount++
				}
			}

			// --- TAMBAHKAN DEBUG PRINT DI LUAR LOOP ---
			fmt.Printf("DEBUG: realEntryCount akhir untuk '%s' adalah %d.\n", entryName, realEntryCount) // --- AKHIR DEBUG PRINT ---

			if realEntryCount > 0 {
				return fmt.Errorf("direktori '%s' tidak kosong (berisi %d entri selain . dan ..), tidak dapat dihapus", entryName, realEntryCount)
			}
		}

		// Jika direktori kosong (atau dianggap kosong), bebaskan blok datanya
		fmt.Printf("Direktori '%s' kosong. Membebaskan blok mulai dari %d.\n", entryName, entryToDelete.StartBlock)
		err = freeBlockChain(entryToDelete.StartBlock)
		if err != nil {
			return fmt.Errorf("gagal membebaskan blok data direktori '%s': %w", entryName, err)
		}
	} else {
		return fmt.Errorf("tipe entri tidak dikenal untuk '%s'", entryName)
	}

	// 4. Invalidate/Hapus Entri dari Direktori Induk
	err = invalidateEntryInParent(parentDirStartBlock, entryName)
	if err != nil {
		// Jika gagal menginvalidasi dari induk, ini masalah.
		// Blok data mungkin sudah terlanjur dibebaskan. Idealnya ada mekanisme transaksi/rollback yang lebih baik.
		// Untuk sekarang, kita kembalikan errornya.
		return fmt.Errorf("berhasil membebaskan blok data untuk '%s', TAPI gagal menginvalidasi entri dari direktori induk: %w", entryName, err)
	}

	fmt.Printf("Entri '%s' berhasil dihapus.\n", entryName)
	return nil
}

// ChangeDirectory: Mengubah direktori kerja saat ini (CurrentDirectoryBlock) di FileSystem.
// Menerima pointer ke FileSystem agar bisa memodifikasinya.
func ChangeDirectory(fs *FileSystem, targetName string) error {
	if fs == nil {
		return errors.New("FileSystem instance tidak boleh nil")
	}

	// 1. Handle kasus khusus targetName
	if targetName == "/" { // Pindah ke root directory
		fs.CurrentDirectoryBlock = ROOT_DIR_BLOCK
		fmt.Printf("Direktori diubah ke root (Blok %d).\n", fs.CurrentDirectoryBlock)
		return nil
	}

	if targetName == "." { // Direktori saat ini, tidak ada perubahan
		// fmt.Printf("Direktori tetap di Blok %d.\n", fs.CurrentDirectoryBlock)
		return nil
	}

	// 2. Ambil semua entri dari direktori saat ini untuk mencari targetName
	currentEntries, err := ListEntries(fs.CurrentDirectoryBlock)
	if err != nil {
		return fmt.Errorf("gagal membaca direktori saat ini (Blok %d) untuk cd: %w", fs.CurrentDirectoryBlock, err)
	}

	// 3. Cari targetName (baik itu ".." atau nama direktori spesifik)
	var targetEntry *DirectoryEntry // Gunakan pointer agar bisa nil jika tidak ketemu
	for i, entry := range currentEntries {
		entryNameStr := string(entry.Name[:bytes.IndexByte(entry.Name[:], 0)])
		if entryNameStr == targetName {
			// Periksa apakah ini direktori (untuk nama biasa atau "..")
			if entry.Type == TYPE_DIRECTORY {
				targetEntry = &currentEntries[i] // Ambil alamat dari elemen slice
				break
			} else if targetName != ".." { // Jika nama biasa tapi bukan direktori
				return fmt.Errorf("'%s' bukan direktori", targetName)
			}
            // Jika targetName adalah ".." dan tipenya bukan direktori, itu aneh, tapi ListEntries harusnya hanya return dir untuk ".."
		}
	}

	// 4. Proses hasil pencarian
	if targetEntry == nil {
		return fmt.Errorf("direktori '%s' tidak ditemukan di direktori saat ini (Blok %d)", targetName, fs.CurrentDirectoryBlock)
	}

	// Jika ditemukan dan merupakan direktori, ubah CurrentDirectoryBlock
	// targetEntry.StartBlock adalah blok awal dari direktori tujuan (baik itu ".." atau nama direktori lain)
	if targetEntry.StartBlock < 0 || targetEntry.StartBlock >= BlockID(TOTAL_BLOCKS) || FAT[targetEntry.StartBlock] == FAT_FREE {
		// Ini seharusnya tidak terjadi jika entri valid, kecuali untuk ".." di root yang StartBlock-nya ROOT_DIR_BLOCK
        // atau jika metadata korup.
		return fmt.Errorf("StartBlock untuk direktori tujuan '%s' (Blok %d) tidak valid atau belum dialokasikan", targetName, targetEntry.StartBlock)
	}

	fs.CurrentDirectoryBlock = targetEntry.StartBlock
	fmt.Printf("Direktori diubah ke '%s' (Blok %d).\n", targetName, fs.CurrentDirectoryBlock)
	return nil
}