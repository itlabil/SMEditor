package prompt

// defaultBlock is what "reset to default" restores a block to. The text
// here must mirror the latest seed of each block exactly:
// backend/internal/db/migrations/0002_seed_prompt_blocks.sql, with the moba
// block superseded by 0005_moba_draft_prompt.sql (SM-16). Committed
// migrations are never edited (per .agents/skills/sm-database), so once a
// user customizes a block the original seed text is no longer recoverable
// from the database itself, and lives here instead.
type defaultBlock struct {
	Body       string
	Categories []string
}

var defaultBlocks = map[string]defaultBlock{
	"frame": {
		Body: `Kamu adalah editor video highlight esport {nama_game} yang berpengalaman.
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
}`,
		Categories: []string{},
	},
	"moba": {
		Body: `Istilah game ini:
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
- kesimpulan: simpulkan mengapa tim pemenang bisa menang, lalu sebutkan {penghargaan} jika ada.`,
		Categories: []string{"draft", "early", "mid", "end", "kesimpulan"},
	},
	"br": {
		Body: `Istilah menang di game ini: {istilah_menang}
Tim yang difokuskan (opsional): {tim_fokus}

Susun highlight dalam empat bagian, berurutan:
1. drop: 1 segmen pembuka (map, jalur pesawat, drop yang diperebutkan), hanya jika caster membahasnya.
2. awal dan tengah match: 2-4 momen per bagian, dengan kategori fight (pertempuran antar tim, wipe squad, clutch, kill beruntun) atau rotasi (rotasi berisiko, perebutan compound).
3. zona-akhir: 1-2 segmen pertarungan di zona terakhir sampai tersisa satu tim.
4. hasil: 1 segmen berisi {istilah_menang}, jumlah kill, dan klasemen jika disebut caster.

Jika {tim_fokus} diisi, utamakan momen yang melibatkan tim itu.
Lewati looting dan perjalanan tanpa kejadian.
Durasi segmen 10-75 detik. Segmen zona-akhir boleh sampai 150 detik.`,
		Categories: []string{"drop", "fight", "rotasi", "zona-akhir", "hasil"},
	},
	"fps": {
		Body: `Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen berisi map, komposisi {istilah_karakter} kedua tim, dan sisi awal (attack atau defense).
2. paruh pertama dan paruh kedua: 2-4 ronde terbaik per paruh, dengan kategori ronde (pistol round, ace, multi kill, ronde eco yang menang) atau clutch (1 lawan banyak, defuse atau plant di detik terakhir).
3. penentu: 1-3 segmen berisi ronde match point, overtime, atau ronde yang membalik keadaan.
4. hasil: 1 segmen berisi skor akhir, MVP, dan statistik yang disebut caster.

Selalu sebutkan skor ronde di narasi jika caster menyebutnya.
Durasi segmen 15-60 detik. Satu ronde penuh boleh sampai 120 detik.`,
		Categories: []string{"pembuka", "ronde", "clutch", "penentu", "hasil"},
	},
	"bola": {
		Body: `Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen berisi tim yang dipakai, formasi, dan pemain kunci.
2. babak pertama dan babak kedua: semua gol masuk sebagai kategori gol. Tambahkan 2-4 momen lain per babak sebagai kategori peluang (peluang emas, penyelamatan, tiang gawang, kartu merah, penalti).
3. penentu: perpanjangan waktu atau adu penalti, jika ada.
4. hasil: 1 segmen berisi skor akhir dan statistik yang disebut caster.

Setiap segmen gol dimulai dari awal serangan, bukan dari tendangan terakhir.
Selalu sebutkan skor terbaru di narasi setelah gol.
Durasi segmen 15-60 detik.`,
		Categories: []string{"pembuka", "gol", "peluang", "penentu", "hasil"},
	},
	"umum": {
		Body: `Susun highlight dalam empat bagian, berurutan:
1. pembuka: 1 segmen perkenalan pertandingan dan kedua pihak.
2. momen: 2-4 momen per bagian awal, tengah, dan akhir pertandingan, yaitu yang paling ditekankan caster.
3. penentu: 1-2 segmen momen yang menentukan hasil.
4. hasil: 1 segmen berisi hasil akhir dan kesimpulan caster.

Durasi segmen 15-90 detik.`,
		Categories: []string{"pembuka", "momen", "penentu", "hasil"},
	},
}
