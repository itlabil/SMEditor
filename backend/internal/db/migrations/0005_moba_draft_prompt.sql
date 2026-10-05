-- SM-16: blok tugas MOBA meminta field "draft" (hasil pick dan ban).
-- Hanya menimpa blok yang belum diubah pengguna (is_custom = 0); teks
-- yang sama ada di backend/internal/modules/prompt/defaults.go.
UPDATE prompt_blocks SET body = 'Istilah game ini:
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
- kesimpulan: simpulkan mengapa tim pemenang bisa menang, lalu sebutkan {penghargaan} jika ada.', updated_at = '2026-10-05T12:00:00Z'
WHERE code = 'moba' AND is_custom = 0;
