# PRD: Editor Highlight Esport

Oct 2, 2026 · @Erwin Dianto

## Ringkasan dan tujuan

Sistem ini terdiri dari dua bagian: app web lokal yang menyiapkan bahan, dan plugin Premiere yang memotong video asli secara otomatis dari hasil highlight AI.

App web mengunduh video, membuat transcript, menyediakan prompt, dan memvalidasi JSON highlight. Plugin membaca JSON itu dan menyusun potongan di timeline Premiere. Perapian potongan, dubbing, musik, transisi, dan export dikerjakan di Premiere.

Tujuan:

- Memangkas waktu mencari dan memotong momen penting di video 20–40 menit.
- Potongan tetap bisa ditarik dan digeser di timeline Premiere tanpa re-encode.
- Script narasi bahasa Indonesia per fase tersedia di timeline dan sebagai file `.txt`.

Pengguna hanya satu orang, yaitu pemilik app.

## Ruang lingkup

Versi 1 mencakup highlight landscape 16:9, dengan pemotongan dilakukan plugin di Premiere.

| App web | Plugin Premiere | Di luar versi 1 |
| --- | --- | --- |
| Project: buat, daftar, buka, hapus | Pilih video asli dan file highlight | Editor timeline (potong, susun ulang) dan export di web |
| Download video 1080p, fallback 720p | Buat sequence sesuai resolusi dan fps video | Mode short 9:16 |
| Transcript multi-bahasa dengan timestamp | Susun potongan otomatis dengan tambahan waktu dan jeda | Panggilan API AI otomatis |
| Unduh transcript `.json` dan `.txt` | Marker berisi label dan narasi per potongan | Dubbing, musik, transisi otomatis |
| Template prompt yang bisa disalin | Laporan segmen yang dilewati | Login, multi-user, hosting publik |
| Validasi JSON highlight, simpan sebagai file |  |  |
| Unduh script narasi `.txt` |  |  |
| Pratinjau video per segmen highlight (tanpa edit) |  |  |

## Alur pengguna

Satu project berisi satu game: enam langkah di app web, lalu tiga langkah di Premiere.

1. Buat project: isi nama, pilih game dari daftar mode, tempel URL YouTube.
2. App mengunduh video dan menampilkan progres.
3. App membuat transcript; bahasa dideteksi otomatis atau dipilih manual.
4. Unduh transcript `.txt` dan salin prompt dari halaman project.
5. Tempel keduanya ke AI (Claude, GPT, Gemini), lalu salin JSON jawabannya.
6. Tempel JSON ke app. Setelah lolos validasi, app menyimpan `highlight.json` dan `narasi.txt` di folder project.
7. Salin atau pindahkan video asli dan `highlight.json` ke folder project Premiere.
8. Di panel plugin, pilih kedua file itu dan klik Buat Sequence.
9. Rapikan potongan di timeline, lalu lanjutkan dubbing dan export di Premiere.

Status project di app web: `baru`, `mengunduh`, `transcript`, `menunggu highlight`, `siap untuk Premiere`. Langkah yang gagal bisa diulang tanpa mengulang langkah sebelumnya.

## Kebutuhan fungsional

### Project

- Daftar project menampilkan nama, game, status, durasi video, dan ukuran file.
- Hapus project meminta konfirmasi SweetAlert, lalu menghapus baris database dan seluruh folder project: video, audio, transcript, dan hasil export.
- Job yang sedang berjalan dihentikan dulu sebelum folder dihapus.

### Download

- Memakai yt-dlp, dengan prioritas video H.264 dan audio AAC dalam MP4 pada 1080p, lalu 720p.
- Jika hanya tersedia VP9 atau AV1, video tetap diunduh dan dibuatkan file proxy H.264 untuk preview. Export tetap memakai file asli.
- Progres (persen, kecepatan, sisa waktu) tampil langsung dan bisa dibatalkan.
- Menyimpan judul, durasi, resolusi, fps, dan thumbnail.

### Transcript

- Audio diekstrak ke WAV 16 kHz mono, lalu diproses Whisper lokal.
- Pilihan bahasa: otomatis, Indonesia, Inggris, Filipina, atau kode bahasa lain.
- Hasil berisi waktu mulai, waktu selesai, dan teks per kalimat.
- Unduh sebagai `.json` (lengkap) atau `.txt` (ringkas, untuk ditempel ke AI).
- Transcript bisa diulang dengan bahasa atau model lain.

### Prompt

- Halaman project menampilkan prompt sesuai game, dengan tombol salin.
- Judul video, durasi, dan nama tim (opsional) terisi otomatis ke dalam prompt.
- Prompt bisa diedit dan disimpan sebagai template sendiri.

### Validasi highlight

- Kolom tempel menerima JSON dari AI, termasuk yang terbungkus blok kode.
- Validasi: skema benar, `mulai` lebih kecil dari `selesai`, waktu tidak melewati durasi video, segmen tidak tumpang tindih, dan jika ada `draft`, pick tiap tim maksimal 5.
- Kesalahan ditunjukkan per segmen supaya bisa diperbaiki atau diminta ulang ke AI.
- Hasil yang lolos ditampilkan sebagai daftar segmen untuk dibaca, lalu disimpan sebagai `highlight.json`.
- Tiap segmen pada daftar bisa diputar sebagai pratinjau dari video asli, dari waktu mulai sampai selesai; hanya pemutaran, tanpa fitur edit (potong, susun ulang, dan export tetap dilakukan di Premiere).
- Tiap segmen pada daftar bisa diubah lewat tombol "Ubah" (label, narasi, kategori, waktu mulai, waktu selesai) atau dihapus dengan konfirmasi SweetAlert. Perubahan diperiksa dengan validasi yang sama seperti saat menempel JSON; jika lolos, `highlight.json` dan `narasi.txt` ditulis ulang, jika tidak, kesalahan tampil di form dan file tidak berubah.
- Jika highlight punya `draft`, kartu "Hasil draft" di atas daftar segmen menampilkan pick dan ban kedua tim, dan bisa diubah lewat tombol "Ubah" dengan validasi yang sama. Untuk genre MOBA kartu tetap tampil walau belum ada draft, supaya bisa diisi manual.
- Script narasi dibuat dari JSON yang sama dan disimpan sebagai `narasi.txt`: hasil draft (jika ada) di bagian atas, lalu berurutan per segmen.
- Tombol buka folder project memudahkan menyalin video dan file highlight ke Premiere.

### Salin ke folder

- Tombol hanya aktif saat status `siap_premiere`.
- Pengguna mengisi path folder tujuan lewat input teks; nilai terakhir disimpan sebagai setting `export_dir` dan menjadi isian default berikutnya.
- App membuat subfolder bernama project (dibersihkan dari karakter yang tidak sah di Windows dan Linux) di dalam tujuan, lalu menyalin `source.mp4`, `highlight.json`, dan `narasi.txt` ke sana. Field `video` di `highlight.json` hasil salinan tetap sama dengan nama file video yang disalin, karena keduanya disalin apa adanya tanpa diubah.
- Penyalinan berjalan sebagai job `export` lewat worker, dengan progres dan bisa dibatalkan; job ini tidak mengubah status project.
- Jika subfolder tujuan sudah ada, pengguna diminta konfirmasi sebelum menimpa. Jika folder tujuan tidak ada atau tidak bisa ditulis, app menolak dengan pesan yang jelas sebelum membuat job apa pun.
- Tujuannya: setelah disalin, project di app boleh dihapus tanpa membuat media offline di project Premiere.

### Plugin Premiere

- Panel berisi: pilih video asli, pilih `highlight.json`, nama sequence, tambahan waktu, dan jeda antar potongan.
- Tambahan waktu memperpanjang tiap potongan di awal dan akhir, default 1 detik, sebagai ruang untuk transisi dan koreksi jika potongan AI kurang pas.
- Jeda antar potongan di timeline, default 1 detik.
- Plugin mengimpor video ke project jika belum ada, lalu membuat sequence dengan resolusi dan fps yang sama dengan video.
- Tiap segmen menjadi satu potongan video beserta audionya, disusun berurutan di track pertama.
- Tiap potongan diberi marker: nama berisi label, komentar berisi narasi, warna mengikuti kategori.
- Plugin memeriksa ulang JSON. Segmen yang tidak valid dilewati dan dilaporkan di panel.
- Tambahan waktu dibatasi supaya tidak melewati awal atau akhir video.
- Menjalankan ulang membuat sequence baru; sequence lama tidak diubah.
- Tandai manual tidak perlu fitur khusus: momen yang terlewat dipotong langsung di timeline Premiere.

## Format data

Transcript `.txt` dipakai untuk AI karena paling hemat token; satu baris satu kalimat:

```
[00:00:12] Welcome back to game three of the grand final
[00:00:17] and we are heading into the draft phase
```

Transcript `.json` menyimpan data lengkap:

```json
{
  "bahasa": "en",
  "durasi": 1834.5,
  "segmen": [
    { "mulai": 12.0, "selesai": 16.8, "teks": "Welcome back to game three of the grand final" }
  ]
}
```

JSON highlight yang diminta dari AI dan diterima app:

```json
{
  "game": "mlbb",
  "ringkasan": "Tim A menang lewat war Lord kedua di menit 16.",
  "segmen": [
    {
      "mulai": "00:01:05",
      "selesai": "00:02:10",
      "kategori": "draft",
      "label": "Draft pick kedua tim",
      "alasan": "Caster membahas pick terakhir dan komposisi tim",
      "narasi": "Tim A mengamankan hero incaran di pick pertama..."
    }
  ]
}
```

| Field | Aturan |
| --- | --- |
| `mulai`, `selesai` | Format `HH:MM:SS`, diambil dari waktu di transcript |
| `kategori` | Sesuai genre; daftarnya ada di bagian Template prompt. MOBA: `draft`, `early`, `mid`, `end`, `kesimpulan` |
| `label` | Judul pendek, maksimal 8 kata |
| `alasan` | Kutipan atau konteks dari transcript, untuk pengecekan |
| `narasi` | Bahasa Indonesia, maksimal sekitar 2 kata per detik durasi segmen |
| `draft` | Opsional, hanya diminta prompt MOBA. Jika ada, `pick` tiap tim maksimal 5 hero (urutan pick) dan `ban` boleh kosong |

Untuk genre MOBA, JSON boleh berisi field `draft` di tingkat atas, sejajar dengan `segmen`:

```json
{
  "draft": {
    "tim_a": { "nama": "ONIC", "pick": ["Fanny", "Kaja"], "ban": ["Ling"] },
    "tim_b": { "nama": "RRQ", "pick": ["Lancelot"], "ban": [] }
  }
}
```

Nama hero ditulis dengan ejaan resmi game; hero yang tidak disebut caster tidak diisi. Plugin Premiere hanya membaca `segmen`, jadi `draft` tidak memengaruhinya.

File `highlight.json` yang disimpan app web memakai struktur yang sama, ditambah `video` (nama file video asli) dan `durasi` (detik), supaya plugin bisa mencocokkan file dan memeriksa batas waktu.

## Template prompt

App menyediakan 12 mode game dalam empat genre, ditambah mode umum. Bagian dalam kurung kurawal diisi otomatis oleh app.

Tiap prompt dirakit app dari tiga bagian: kerangka bersama, blok tugas sesuai genre, dan istilah khusus game yang dipilih.

### Kerangka bersama

```
Kamu adalah editor video highlight esport {nama_game} yang berpengalaman.
Saya melampirkan transcript komentar caster dari satu pertandingan, lengkap dengan waktu.

Data pertandingan:
- Judul video: {judul}
- Durasi video: {durasi}
- Tim: {tim_a} vs {tim_b}
- Perkiraan total durasi highlight: {target_durasi} menit

{blok_tugas}

Aturan segmen:
- Satu momen = satu segmen.
- Waktu hanya boleh diambil dari timestamp yang ada di transcript. Jangan mengarang waktu.
- Mulai segmen 5-8 detik sebelum aksi dimulai, dan akhiri 3-5 detik setelah aksi selesai.
- Segmen harus berurutan sesuai waktu dan tidak tumpang tindih.
- Lewati jeda, iklan, replay yang berulang, dan obrolan caster yang tidak terkait permainan.
- Jangan memasukkan momen lemah hanya untuk memenuhi jumlah. Jika suatu bagian tidak punya cukup momen, tulis yang ada dan sebutkan di "ringkasan".

Aturan narasi:
- Tulis narasi dubbing bahasa Indonesia untuk setiap segmen.
- Gaya caster yang santai tapi jelas, tanpa kata kasar.
- Panjang maksimal 2 kata per detik durasi segmen.
- Jelaskan apa yang terjadi dan dampaknya bagi jalannya pertandingan.
- Hanya sebut nama tim, pemain, karakter, dan angka yang benar-benar ada di transcript.

Format jawaban:
Balas HANYA dengan JSON valid, tanpa teks lain, dengan struktur:
{
  "game": "{kode_game}",
  "ringkasan": "2-3 kalimat jalannya pertandingan",
  "segmen": [
    {
      "mulai": "HH:MM:SS",
      "selesai": "HH:MM:SS",
      "kategori": "{daftar_kategori}",
      "label": "judul pendek maksimal 8 kata",
      "alasan": "kutipan atau konteks dari transcript",
      "narasi": "script dubbing bahasa Indonesia"
    }
  ]
}
```

### Blok tugas MOBA

Dipakai untuk Mobile Legends, Honor of Kings, Arena of Valor, Wild Rift, League of Legends, dan Dota 2. Kategori: `draft`, `early`, `mid`, `end`, `kesimpulan`.

Blok ini juga meminta field `draft` (hasil pick dan ban, lihat Format data). Teks di bawah adalah versi terbaru, dipasang lewat migrasi `0005_moba_draft_prompt.sql` hanya jika blok MOBA belum diubah pengguna (`is_custom = 0`); blok yang sudah diubah pengguna dibiarkan, dan "kembalikan ke bawaan" memakai teks ini.

```
Istilah game ini:
- Objektif: {objektif}
- Bangunan: {bangunan}
- Penghargaan akhir: {penghargaan}

Susun highlight dalam lima fase, berurutan:
1. draft: 1 segmen berisi hasil akhir ban dan pick kedua tim. Boleh 2 segmen jika fase ban dan pick terpisah jauh.
2. early: momen penting awal game, misalnya first blood, objektif pertama, invade, atau gank yang berhasil.
3. mid: momen penting pertengahan game, misalnya team fight, perebutan objektif, bangunan penting yang hancur, atau pick off.
4. end: momen penentu akhir game, misalnya war objektif besar, wipe out, comeback, pertahanan base, sampai base hancur.
5. kesimpulan: 1 segmen berisi hasil akhir, skor, {penghargaan}, dan statistik yang disebut caster.

Jumlah momen untuk fase early, mid, dan end, masing-masing:
- 2 momen jika fase itu sepi.
- 3 momen jika fase itu ramai.
- 4 momen hanya jika semuanya benar-benar penting.

Durasi segmen 15-90 detik. Segmen draft boleh sampai 120 detik.

Hasil draft:
Tambahkan field "draft" di JSON jawaban, sejajar dengan "ringkasan" dan "segmen", dengan struktur:
"draft": {
  "tim_a": {"nama": "nama tim pertama", "pick": ["hero pick pertama", "hero pick kedua"], "ban": ["hero yang di-ban"]},
  "tim_b": {"nama": "nama tim kedua", "pick": ["hero pick pertama"], "ban": []}
}
- Isi dari transcript. Tulis "pick" sesuai urutan pick tim itu, maksimal 5 hero per tim. "ban" boleh kosong.
- Tulis nama hero dengan ejaan resmi di game. Jika transcript salah dengar nama hero, perbaiki hanya jika jelas hero mana yang dimaksud.
- Jangan mengarang hero yang tidak disebut caster. Isi hanya yang ada; daftar boleh kurang dari 5 atau kosong.
- Jika caster sama sekali tidak membahas draft, hilangkan field "draft".

Narasi khusus:
- draft: sebutkan dulu semua hero yang di-pick tiap tim, sama dengan isi field "draft", baru setelah itu komentari komposisinya.
- kesimpulan: simpulkan mengapa tim pemenang bisa menang, lalu sebutkan {penghargaan} jika ada.
```

Istilah yang diisi per game:

| Game | Objektif | Bangunan | Penghargaan akhir |
| --- | --- | --- | --- |
| Mobile Legends | Turtle, Lord | Turret, base | MVP |
| Honor of Kings | Tyrant, Overlord, Tempest Dragon | Tower, crystal | MVP |
| Arena of Valor | Abyssal Dragon, Dark Slayer | Tower, core | MVP |
| Wild Rift | Dragon, Rift Herald, Baron Nashor | Turret, Nexus | MVP |
| League of Legends | Dragon, Rift Herald, Baron Nashor, Elder Dragon | Turret, inhibitor, Nexus | Player of the Game |
| Dota 2 | Roshan dan Aegis, Tormentor | Tower, barracks, Ancient | Statistik akhir dan net worth |

### Blok tugas battle royale

Dipakai untuk PUBG Mobile (istilah menang: Winner Winner Chicken Dinner) dan Free Fire (Booyah). Kategori: `drop`, `fight`, `rotasi`, `zona-akhir`, `hasil`.

```
Istilah menang di game ini: {istilah_menang}
Tim yang difokuskan (opsional): {tim_fokus}

Susun highlight dalam empat bagian, berurutan:
1. drop: 1 segmen pembuka (map, jalur pesawat, drop yang diperebutkan), hanya jika caster membahasnya.
2. awal dan tengah match: 2-4 momen per bagian, dengan kategori fight (pertempuran antar tim, wipe squad, clutch, kill beruntun) atau rotasi (rotasi berisiko, perebutan compound).
3. zona-akhir: 1-2 segmen pertarungan di zona terakhir sampai tersisa satu tim.
4. hasil: 1 segmen berisi {istilah_menang}, jumlah kill, dan klasemen jika disebut caster.

Jika {tim_fokus} diisi, utamakan momen yang melibatkan tim itu.
Lewati looting dan perjalanan tanpa kejadian.
Durasi segmen 10-75 detik. Segmen zona-akhir boleh sampai 150 detik.
```

### Blok tugas FPS taktis

Dipakai untuk Valorant dan Call of Duty Mobile. Kategori: `pembuka`, `ronde`, `clutch`, `penentu`, `hasil`.

```
Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen berisi map, komposisi {istilah_karakter} kedua tim, dan sisi awal (attack atau defense).
2. paruh pertama dan paruh kedua: 2-4 ronde terbaik per paruh, dengan kategori ronde (pistol round, ace, multi kill, ronde eco yang menang) atau clutch (1 lawan banyak, defuse atau plant di detik terakhir).
3. penentu: 1-3 segmen berisi ronde match point, overtime, atau ronde yang membalik keadaan.
4. hasil: 1 segmen berisi skor akhir, MVP, dan statistik yang disebut caster.

Selalu sebutkan skor ronde di narasi jika caster menyebutnya.
Durasi segmen 15-60 detik. Satu ronde penuh boleh sampai 120 detik.
```

Untuk Valorant, `{istilah_karakter}` diisi agent; untuk Call of Duty Mobile diisi loadout dan operator.

### Blok tugas sepak bola

Dipakai untuk eFootball dan EA Sports FC. Kategori: `pembuka`, `gol`, `peluang`, `penentu`, `hasil`.

```
Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen berisi tim yang dipakai, formasi, dan pemain kunci.
2. babak pertama dan babak kedua: semua gol masuk sebagai kategori gol. Tambahkan 2-4 momen lain per babak sebagai kategori peluang (peluang emas, penyelamatan, tiang gawang, kartu merah, penalti).
3. penentu: perpanjangan waktu atau adu penalti, jika ada.
4. hasil: 1 segmen berisi skor akhir dan statistik yang disebut caster.

Setiap segmen gol dimulai dari awal serangan, bukan dari tendangan terakhir.
Selalu sebutkan skor terbaru di narasi setelah gol.
Durasi segmen 15-60 detik.
```

### Blok tugas umum

Dipakai untuk game lain. Kategori: `pembuka`, `momen`, `penentu`, `hasil`.

```
Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen perkenalan pertandingan dan kedua pihak.
2. momen: 2-4 momen per bagian awal, tengah, dan akhir pertandingan, yaitu yang paling ditekankan caster.
3. penentu: 1-2 segmen momen yang menentukan hasil.
4. hasil: 1 segmen berisi hasil akhir dan kesimpulan caster.

Durasi segmen 15-90 detik.
```

Mode baru bisa ditambahkan dari halaman pengaturan dengan menyalin blok tugas yang ada, lalu mengubah kategori dan istilahnya.

## Arsitektur teknis

App web adalah satu binary Go yang melayani API dan frontend di `localhost`. Plugin Premiere adalah panel terpisah yang hanya membaca file, tanpa koneksi ke app web.

| Bagian | Pilihan | Catatan |
| --- | --- | --- |
| Backend | Go dengan Gin | Pola yang sama dengan project Go/Vue lain |
| Frontend | Vue 3, Vite, Tailwind, SweetAlert2 | Hasil build ditanam ke binary Go |
| Database | SQLite | Satu file, tanpa server database |
| Download | yt-dlp | Dipanggil sebagai proses; perlu diperbarui berkala |
| Transcript | Whisper lokal (faster-whisper atau whisper.cpp) | Model `medium`, GPU bila tersedia, fallback CPU |
| Video | ffmpeg dan ffprobe | Ekstrak audio, baca metadata, buat proxy |
| Progres | Server-Sent Events | Untuk download dan transcript |
| Preview | HTTP range request | Supaya seek di video panjang tetap cepat |

Pekerjaan panjang berjalan sebagai job antrean, satu per satu, dengan status tersimpan di database supaya bertahan saat app ditutup.

Struktur penyimpanan per project:

```
data/
  app.db
  projects/{id}/
    source.mp4
    proxy.mp4        (hanya jika sumber bukan H.264)
    audio.wav
    transcript.json
    transcript.txt
    highlight.json
    narasi.txt
    thumbnail.jpg
```

Tabel utama: `projects`, `jobs`, dan `prompt_templates`. Segmen highlight cukup disimpan sebagai file.

Lokasi yt-dlp, ffmpeg, model Whisper, dan folder data diatur di halaman pengaturan.

### Plugin Premiere

Plugin dibuat sebagai panel CEP (HTML, JavaScript, dan ExtendScript), karena versi Premiere yang dipakai belum pasti dan kemungkinan bukan versi baru. CEP berjalan di banyak versi lama, sedangkan UXP butuh Premiere 25.6 atau lebih baru. Jika nanti Premiere diperbarui, panel bisa dipindahkan ke UXP.

| Bagian | Isi |
| --- | --- |
| Panel | HTML dan JavaScript: pilih file, isi opsi, tampilkan laporan |
| Skrip host | ExtendScript: impor video, buat sequence, sisipkan potongan, buat marker |
| Pemasangan | Salin folder plugin ke folder ekstensi CEP dan aktifkan mode debug, tanpa perlu tanda tangan Adobe |
| Sistem operasi | Windows atau macOS; Premiere tidak berjalan di Ubuntu |

## Kebutuhan non-fungsional

- App web: spesifikasi minimal GTX 1650, RAM 8 GB, Intel i3 generasi 10.
- Plugin: PC Windows dengan Premiere terpasang.
- App web hanya mendengarkan di `127.0.0.1`; internet hanya dipakai untuk mengunduh video.
- Aksi merusak seperti hapus project memakai konfirmasi SweetAlert.
- Antarmuka app web dan panel plugin berbahasa Indonesia.

## Risiko

| Risiko | Penanganan |
| --- | --- |
| Plugin tidak bisa diuji sebelum dipasang di Premiere Anda | Uji bertahap di PC Anda; perbaiki dari pesan error |
| Fungsi skrip berbeda antar versi Premiere | Pastikan versi dulu; cadangan berupa file timeline XML yang diimpor tanpa plugin |
| Caster tidak menyebut momen penting, atau video tanpa caster | Potong manual di timeline Premiere |
| AI mengarang waktu atau nama | Aturan prompt, field `alasan`, validasi di app web dan di plugin |
| Transcript bahasa Filipina kurang akurat | Pilih bahasa manual; tambah potongan manual di Premiere |
| yt-dlp gagal setelah YouTube berubah | Tombol perbarui yt-dlp di pengaturan |
| Video sumber sulit diputar di Premiere (VP9 atau AV1) | Prioritas unduh H.264; jika tidak tersedia, konversi ke H.264 |

## Tahapan pengerjaan

1. App web: kerangka, project (buat, daftar, hapus), dan download dengan progres.
2. App web: transcript, unduh `.json` dan `.txt`, serta halaman prompt.
3. App web: validasi JSON, simpan `highlight.json` dan `narasi.txt`.
4. Plugin: panel dasar, impor video, dan buat sequence.
5. Plugin: susun potongan dengan tambahan waktu dan jeda, lalu marker narasi.
6. Tahap berikutnya: mode short 9:16.

## Pertanyaan terbuka

- Premiere versi berapa yang terpasang? Lihat di menu Help, About Premiere Pro. Jawaban ini memastikan pilihan CEP.
- Berapa target durasi highlight per game? Angka ini menjadi nilai default `{target_durasi}` di prompt.
- Apakah nama tim diisi manual saat membuat project, atau cukup diambil dari judul video?
