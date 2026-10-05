// Skrip host SMEditor (ExtendScript, setara ES3).
// Aturan: hanya var, tanpa arrow function, tanpa let/const, tanpa template string.
// Setiap fungsi mengembalikan string JSON: {"ok":true,"data":{...}} atau {"ok":false,"error":"..."}.

var SME_TICKS_PER_SECOND = 254016000000;

// ---------- Util ----------

function smeEscape(value) {
    var s = String(value);
    s = s.replace(/\\/g, '\\\\');
    s = s.replace(/"/g, '\\"');
    s = s.replace(/\r/g, '\\r');
    s = s.replace(/\n/g, '\\n');
    s = s.replace(/\t/g, '\\t');
    return s;
}

function smeFail(message) {
    return '{"ok":false,"error":"' + smeEscape(message) + '"}';
}

function smeNormPath(p) {
    return String(p).replace(/\\/g, '/').toLowerCase();
}

function smeTicks(seconds) {
    return String(Math.round(seconds * SME_TICKS_PER_SECOND));
}

function smeTime(seconds) {
    var t = new Time();
    t.seconds = seconds;
    return t;
}

function smeNear(a, b, tolerance) {
    var d = a - b;
    if (d < 0) { d = -d; }
    return d <= tolerance;
}

// ---------- Tahap 1: sambungan ----------

function smePing() {
    try {
        var hasProject = false;
        var projectName = '';
        var projectPath = '';
        if (app.project && app.project.name) {
            hasProject = true;
            projectName = app.project.name;
            projectPath = app.project.path ? app.project.path : '';
        }
        return '{"ok":true,"data":{' +
            '"version":"' + smeEscape(app.version) + '",' +
            '"hasProject":' + (hasProject ? 'true' : 'false') + ',' +
            '"project":"' + smeEscape(projectName) + '",' +
            '"projectPath":"' + smeEscape(projectPath) + '"' +
            '}}';
    } catch (e) {
        return smeFail(e.toString());
    }
}

// ---------- Tahap 2: sequence dan potongan ----------

// Mencari item project berdasarkan path media, termasuk di dalam bin.
function smeFindByPath(bin, normPath) {
    var i, item, mediaPath, found;
    for (i = 0; i < bin.children.numItems; i++) {
        item = bin.children[i];
        if (item.type === 2) { // BIN
            found = smeFindByPath(item, normPath);
            if (found) { return found; }
        } else {
            mediaPath = '';
            try { mediaPath = item.getMediaPath(); } catch (e1) { mediaPath = ''; }
            if (mediaPath && smeNormPath(mediaPath) === normPath) { return item; }
        }
    }
    return null;
}

function smeFindOrImport(videoPath) {
    var nativePath = new File(videoPath).fsName;
    var norm = smeNormPath(nativePath);
    var item = smeFindByPath(app.project.rootItem, norm);
    if (item) { return item; }
    app.project.importFiles([nativePath], true, app.project.rootItem, false);
    return smeFindByPath(app.project.rootItem, norm);
}

function smeClearTracks(tracks) {
    var t, i, track;
    for (t = 0; t < tracks.numTracks; t++) {
        track = tracks[t];
        for (i = track.clips.numItems - 1; i >= 0; i--) {
            track.clips[i].remove(false, false);
        }
    }
}

// Mengatur titik masuk atau keluar klip sumber. Bentuk argumen waktu berbeda antar versi
// Premiere, jadi beberapa bentuk dicoba dan hasilnya dibaca ulang untuk memastikan.
function smeSetPoint(item, isIn, seconds) {
    var variants = [seconds, smeTicks(seconds), smeTime(seconds)];
    var names = ['detik', 'ticks', 'Time'];
    var i, got;
    for (i = 0; i < variants.length; i++) {
        try {
            if (isIn) { item.setInPoint(variants[i], 4); } else { item.setOutPoint(variants[i], 4); }
        } catch (e1) {
            continue;
        }
        got = null;
        try {
            got = isIn ? item.getInPoint().seconds : item.getOutPoint().seconds;
        } catch (e2) {
            return names[i] + '?';
        }
        if (smeNear(got, seconds, 0.1)) { return names[i]; }
    }
    return '';
}

// Menaruh klip di track pada posisi tertentu, lalu memastikan klip benar-benar ada di sana.
function smePlace(track, item, position) {
    var variants = [position, smeTime(position), smeTicks(position)];
    var names = ['detik', 'Time', 'ticks'];
    var i, before, last;
    for (i = 0; i < variants.length; i++) {
        before = track.clips.numItems;
        try {
            track.overwriteClip(item, variants[i]);
        } catch (e1) {
            continue;
        }
        if (track.clips.numItems <= before) { continue; }
        last = track.clips[track.clips.numItems - 1];
        if (smeNear(last.start.seconds, position, 0.5)) { return names[i]; }
        // Masuk di posisi yang salah: buang dan coba bentuk berikutnya.
        try { last.remove(false, false); } catch (e2) { /* biarkan */ }
    }
    return '';
}

function smeAddMarker(sequence, start, end, label, comments, colorIndex) {
    var marker = sequence.markers.createMarker(start);
    marker.name = label;
    marker.comments = comments;
    marker.end = end;
    try { marker.setColorByIndex(colorIndex); } catch (e) { /* warna tidak wajib */ }
}

// params: { videoPath, sequenceName, pad, gap, durasi, segmen: [{nomor, mulai, selesai, kategori, label, narasi}] }
function smeBuild(params) {
    var item = null;
    try {
        if (!app.project || !app.project.rootItem) {
            return smeFail('Belum ada project Premiere yang terbuka.');
        }
        if (!params || !params.segmen || !params.segmen.length) {
            return smeFail('Tidak ada segmen untuk dipotong.');
        }

        item = smeFindOrImport(params.videoPath);
        if (!item) {
            return smeFail('Video tidak bisa diimpor: ' + params.videoPath);
        }

        var sequence = app.project.createNewSequenceFromClips(params.sequenceName, [item], app.project.rootItem);
        if (!sequence) { sequence = app.project.activeSequence; }
        if (!sequence) {
            return smeFail('Sequence tidak bisa dibuat.');
        }

        smeClearTracks(sequence.videoTracks);
        smeClearTracks(sequence.audioTracks);

        var vTrack = sequence.videoTracks[0];
        var pad = Number(params.pad) || 0;
        var gap = Number(params.gap) || 0;
        var durasi = Number(params.durasi) || 0;

        var position = 0;
        var created = 0;
        var markers = 0;
        var markerError = '';
        var skipped = [];
        var clips = [];
        var colors = {};
        var colorCount = 0;
        var methodPoint = '';
        var methodPlace = '';
        var i, seg, inS, outS, m1, m2, placed, length;

        for (i = 0; i < params.segmen.length; i++) {
            seg = params.segmen[i];
            inS = seg.mulai - pad;
            if (inS < 0) { inS = 0; }
            outS = seg.selesai + pad;
            if (durasi > 0 && outS > durasi) { outS = durasi; }
            length = outS - inS;

            m1 = smeSetPoint(item, true, inS);
            m2 = smeSetPoint(item, false, outS);
            if (!m1 || !m2) {
                skipped.push('{"nomor":' + seg.nomor + ',"alasan":"Titik masuk atau keluar tidak bisa diatur."}');
                continue;
            }
            methodPoint = m1;

            placed = smePlace(vTrack, item, position);
            if (!placed) {
                skipped.push('{"nomor":' + seg.nomor + ',"alasan":"Klip tidak bisa ditaruh di timeline."}');
                continue;
            }
            methodPlace = placed;
            created++;
            clips.push('{"nomor":' + seg.nomor + ',"start":' + position + ',"length":' + length + '}');

            try {
                if (colors[seg.kategori] === undefined) {
                    colors[seg.kategori] = colorCount % 8;
                    colorCount++;
                }
                smeAddMarker(sequence, position, position + length,
                    seg.nomor + '. ' + seg.label,
                    (seg.kategori ? '[' + seg.kategori + '] ' : '') + seg.narasi,
                    colors[seg.kategori]);
                markers++;
            } catch (eMarker) {
                markerError = eMarker.toString();
            }

            position += length + gap;
        }

        var videoClips = vTrack.clips.numItems;
        var audioClips = 0;
        try { audioClips = sequence.audioTracks[0].clips.numItems; } catch (eAudio) { audioClips = -1; }

        return '{"ok":true,"data":{' +
            '"sequence":"' + smeEscape(sequence.name) + '",' +
            '"created":' + created + ',' +
            '"markers":' + markers + ',' +
            '"markerError":"' + smeEscape(markerError) + '",' +
            '"videoClips":' + videoClips + ',' +
            '"audioClips":' + audioClips + ',' +
            '"totalSeconds":' + (position > gap ? position - gap : position) + ',' +
            '"methodPoint":"' + smeEscape(methodPoint) + '",' +
            '"methodPlace":"' + smeEscape(methodPlace) + '",' +
            '"clips":[' + clips.join(',') + '],' +
            '"skipped":[' + skipped.join(',') + ']' +
            '}}';
    } catch (e) {
        return smeFail(e.toString() + (e.line ? ' (baris ' + e.line + ')' : ''));
    } finally {
        // Kembalikan klip sumber ke keadaan tanpa titik masuk dan keluar.
        if (item) {
            try { item.clearInPoint(); } catch (eIn) { /* biarkan */ }
            try { item.clearOutPoint(); } catch (eOut) { /* biarkan */ }
        }
    }
}

// ---------- Tahap 3: subtitle ----------

// Mengimpor satu file (misalnya subtitle.srt) ke panel Project.
function smeImportFile(path) {
    try {
        if (!app.project || !app.project.rootItem) {
            return smeFail('Belum ada project Premiere yang terbuka.');
        }
        var nativePath = new File(path).fsName;
        var ok = app.project.importFiles([nativePath], true, app.project.rootItem, false);
        return '{"ok":true,"data":{"imported":' + (ok ? 'true' : 'false') + '}}';
    } catch (e) {
        return smeFail(e.toString());
    }
}
