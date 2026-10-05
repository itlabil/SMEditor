// Panel SMEditor Highlight - tahap 1 (SM-11): sambungan ke Premiere dan baca highlight.json.
// Panel memakai API bawaan CEP (window.cep.fs dan window.__adobe_cep__), tanpa pustaka tambahan.
(function () {
  'use strict';

  var state = { jsonPath: '', videoPath: '', baseName: '', data: null, valid: [], problems: [] };

  function $(id) { return document.getElementById(id); }

  function showMessage(text) {
    var el = $('message');
    if (!text) { el.hidden = true; el.textContent = ''; return; }
    el.textContent = text;
    el.hidden = false;
  }

  // ---------- Sambungan ke Premiere ----------

  function evalHost(script) {
    return new Promise(function (resolve) {
      if (!window.__adobe_cep__) {
        resolve({ ok: false, error: 'Panel ini tidak berjalan di dalam Premiere.' });
        return;
      }
      window.__adobe_cep__.evalScript(script, function (raw) {
        if (!raw || raw === 'EvalScript error.') {
          resolve({ ok: false, error: 'Skrip host gagal dijalankan (EvalScript error).' });
          return;
        }
        try {
          resolve(JSON.parse(raw));
        } catch (e) {
          resolve({ ok: false, error: 'Jawaban host tidak bisa dibaca: ' + raw });
        }
      });
    });
  }

  function checkHost() {
    var el = $('host-status');
    evalHost('smePing()').then(function (res) {
      if (!res.ok) {
        el.textContent = 'Tidak tersambung: ' + res.error;
        el.className = 'bad';
        return;
      }
      var d = res.data;
      el.textContent = 'Tersambung ke Premiere ' + d.version +
        (d.hasProject ? ' - project: ' + d.project : ' - belum ada project terbuka');
      el.className = d.hasProject ? 'ok' : 'bad';
    });
  }

  // ---------- Util path dan waktu ----------

  function dirName(p) { return p.replace(/[\\\/][^\\\/]*$/, ''); }
  function joinPath(dir, name) {
    var sep = dir.indexOf('\\') !== -1 ? '\\' : '/';
    return dir + sep + name;
  }

  // Nama dasar untuk sequence dan subtitle. "Nama Project.highlight.json" menjadi "Nama Project".
  // File lama bernama "highlight.json" memakai nama foldernya.
  function baseNameOf(jsonPath) {
    var file = jsonPath.replace(/^.*[\\\/]/, '');
    var base = file.replace(/\.highlight\.json$/i, '').replace(/\.json$/i, '');
    if (!base || base.toLowerCase() === 'highlight') {
      base = dirName(jsonPath).replace(/^.*[\\\/]/, '');
    }
    return base;
  }

  function parseTime(text) {
    var m = /^(\d{1,2}):([0-5]\d):([0-5]\d)$/.exec(String(text || '').trim());
    if (!m) { return null; }
    return Number(m[1]) * 3600 + Number(m[2]) * 60 + Number(m[3]);
  }

  function formatTime(sec) {
    sec = Math.max(0, Math.round(sec));
    var h = Math.floor(sec / 3600);
    var m = Math.floor((sec % 3600) / 60);
    var s = sec % 60;
    function pad(n) { return (n < 10 ? '0' : '') + n; }
    return pad(h) + ':' + pad(m) + ':' + pad(s);
  }

  // ---------- Baca file ----------

  function fileExists(path) {
    var res = window.cep.fs.stat(path);
    return res.err === window.cep.fs.NO_ERROR;
  }

  function readText(path) {
    var res = window.cep.fs.readFile(path);
    if (res.err !== window.cep.fs.NO_ERROR) {
      throw new Error('File tidak bisa dibaca (kode ' + res.err + '): ' + path);
    }
    return String(res.data).replace(/^﻿/, '');
  }

  function pickFile(title, initialPath, types) {
    var res = window.cep.fs.showOpenDialog(false, false, title, initialPath || '', types);
    if (res.err !== window.cep.fs.NO_ERROR || !res.data || !res.data.length) { return ''; }
    return res.data[0];
  }

  // ---------- Validasi (sama dengan aturan di docs/flow.md 4.5) ----------

  function validate(data) {
    var valid = [];
    var problems = [];
    var durasi = Number(data.durasi) || 0;
    var lastEnd = -1;

    if (!data.segmen || !data.segmen.length) {
      problems.push({ nomor: 0, alasan: 'Tidak ada segmen di dalam file.' });
      return { valid: valid, problems: problems };
    }

    data.segmen.forEach(function (seg, i) {
      var nomor = i + 1;
      var mulai = parseTime(seg.mulai);
      var selesai = parseTime(seg.selesai);
      var alasan = '';

      if (mulai === null || selesai === null) {
        alasan = 'Format waktu harus HH:MM:SS.';
      } else if (mulai >= selesai) {
        alasan = 'Waktu mulai harus lebih kecil dari selesai.';
      } else if (durasi > 0 && selesai > durasi) {
        alasan = 'Waktu selesai melewati durasi video (' + formatTime(durasi) + ').';
      } else if (mulai < lastEnd) {
        alasan = 'Tumpang tindih dengan segmen sebelumnya.';
      }

      if (alasan) {
        problems.push({ nomor: nomor, alasan: alasan });
        return;
      }
      lastEnd = selesai;
      valid.push({
        nomor: nomor,
        mulai: mulai,
        selesai: selesai,
        kategori: String(seg.kategori || ''),
        label: String(seg.label || 'Segmen ' + nomor),
        narasi: String(seg.narasi || '')
      });
    });

    return { valid: valid, problems: problems };
  }

  // ---------- Tampilan ----------

  function render() {
    $('json-path').textContent = state.jsonPath || 'belum dipilih';
    var videoEl = $('video-path');
    if (state.videoPath) {
      videoEl.textContent = state.videoPath;
      videoEl.className = 'path';
    } else {
      videoEl.textContent = state.data ? 'tidak ditemukan di folder yang sama' : 'belum dipilih';
      videoEl.className = state.data ? 'path bad' : 'path';
    }
    $('btn-pick-video').hidden = !(state.data && !state.videoPath);

    var summary = $('summary');
    var build = $('build');
    if (!state.data) { summary.hidden = true; build.hidden = true; return; }
    summary.hidden = false;
    build.hidden = false;
    $('btn-build').disabled = !(state.videoPath && state.valid.length);

    var total = 0;
    state.valid.forEach(function (s) { total += s.selesai - s.mulai; });
    $('summary-line').textContent = state.valid.length + ' segmen siap dipotong, total ' +
      formatTime(total) + (state.data.game ? ' - game: ' + state.data.game : '');
    $('ringkasan').textContent = state.data.ringkasan || '';

    var problemList = $('problem-list');
    problemList.innerHTML = '';
    state.problems.forEach(function (p) {
      var li = document.createElement('li');
      li.textContent = (p.nomor ? 'Segmen ' + p.nomor + ': ' : '') + p.alasan;
      problemList.appendChild(li);
    });
    $('problems').hidden = state.problems.length === 0;

    var list = $('segment-list');
    list.innerHTML = '';
    state.valid.forEach(function (s) {
      var li = document.createElement('li');

      var time = document.createElement('span');
      time.className = 'seg-time';
      time.textContent = formatTime(s.mulai) + ' - ' + formatTime(s.selesai);
      li.appendChild(time);

      if (s.kategori) {
        var cat = document.createElement('span');
        cat.className = 'seg-cat';
        cat.textContent = s.kategori;
        li.appendChild(cat);
      }

      var label = document.createElement('div');
      label.textContent = s.label + ' (' + (s.selesai - s.mulai) + ' detik)';
      li.appendChild(label);

      if (s.narasi) {
        var narasi = document.createElement('div');
        narasi.className = 'seg-narasi';
        narasi.textContent = s.narasi;
        li.appendChild(narasi);
      }
      list.appendChild(li);
    });
  }

  // ---------- Aksi ----------

  function loadHighlight(path) {
    showMessage('');
    state.jsonPath = path;
    state.videoPath = '';
    state.data = null;
    state.valid = [];
    state.problems = [];

    try {
      var data = JSON.parse(readText(path));
      if (!data || typeof data !== 'object') { throw new Error('Isi file bukan objek JSON.'); }
      state.data = data;

      var result = validate(data);
      state.valid = result.valid;
      state.problems = result.problems;

      if (data.video) {
        var candidate = joinPath(dirName(path), String(data.video));
        if (fileExists(candidate)) { state.videoPath = candidate; }
      }
    } catch (e) {
      showMessage('highlight.json tidak bisa dibaca: ' + e.message);
    }
    state.baseName = baseNameOf(path);
    $('seq-name').value = 'Highlight - ' + state.baseName;
    $('report').hidden = true;
    render();
  }

  // Mengubah objek menjadi literal JavaScript yang aman untuk ExtendScript:
  // semua karakter non-ASCII ditulis sebagai \uXXXX.
  function toLiteral(obj) {
    return JSON.stringify(obj).replace(/[\u007f-￿]/g, function (c) {
      return '\\u' + ('0000' + c.charCodeAt(0).toString(16)).slice(-4);
    });
  }

  function addReport(text, warn) {
    var li = document.createElement('li');
    li.textContent = text;
    if (warn) { li.className = 'warn'; }
    $('report-list').appendChild(li);
  }

  function buildSequence() {
    showMessage('');
    var button = $('btn-build');
    var params = {
      videoPath: state.videoPath,
      sequenceName: $('seq-name').value.trim() || 'Highlight',
      pad: Math.max(0, Number($('pad').value) || 0),
      gap: Math.max(0, Number($('gap').value) || 0),
      durasi: Number(state.data.durasi) || 0,
      segmen: state.valid
    };

    button.disabled = true;
    button.textContent = 'Sedang membuat...';
    $('report').hidden = true;
    $('report-list').innerHTML = '';

    evalHost('smeBuild(' + toLiteral(params) + ')').then(function (res) {
      button.disabled = false;
      button.textContent = 'Buat Sequence';
      if (!res.ok) { showMessage('Gagal membuat sequence: ' + res.error); return; }

      var d = res.data;
      $('report').hidden = false;
      addReport('Sequence "' + d.sequence + '" dibuat.');
      addReport(d.created + ' dari ' + params.segmen.length + ' potongan ditaruh, total ' + formatTime(d.totalSeconds) + '.');
      addReport('Isi timeline: ' + d.videoClips + ' klip video, ' + d.audioClips + ' klip audio.',
        d.audioClips !== d.videoClips);
      addReport(d.markers + ' marker dibuat.' + (d.markerError ? ' Error marker: ' + d.markerError : ''),
        d.markers !== d.created);
      d.skipped.forEach(function (s) {
        addReport('Segmen ' + s.nomor + ' dilewati: ' + s.alasan, true);
      });
      addReport('Info teknis: titik=' + d.methodPoint + ', taruh=' + d.methodPlace + '.');
      if ($('make-srt').checked) { writeSubtitle(d.clips || []); }
    });
  }

  // ---------- Subtitle dari narasi ----------

  var WORDS_PER_SECOND = 2.3; // perkiraan kecepatan membaca narasi
  var MAX_CUE_CHARS = 60;

  function srtTime(sec) {
    var ms = Math.max(0, Math.round(sec * 1000));
    var h = Math.floor(ms / 3600000); ms -= h * 3600000;
    var m = Math.floor(ms / 60000); ms -= m * 60000;
    var s = Math.floor(ms / 1000); ms -= s * 1000;
    function pad(n, width) { n = String(n); while (n.length < width) { n = '0' + n; } return n; }
    return pad(h, 2) + ':' + pad(m, 2) + ':' + pad(s, 2) + ',' + pad(ms, 3);
  }

  // Memecah narasi menjadi baris pendek; akhir kalimat diutamakan sebagai batas.
  function splitCues(text) {
    var words = text.replace(/\s+/g, ' ').trim().split(' ');
    var cues = [];
    var current = '';
    words.forEach(function (w) {
      if (!w) { return; }
      if (current && current.length + 1 + w.length > MAX_CUE_CHARS) {
        cues.push(current);
        current = w;
      } else {
        current = current ? current + ' ' + w : w;
      }
      if (/[.!?]$/.test(w) && current.length > MAX_CUE_CHARS / 2) {
        cues.push(current);
        current = '';
      }
    });
    if (current) { cues.push(current); }
    return cues;
  }

  function buildSrt(clips) {
    var byNomor = {};
    state.valid.forEach(function (s) { byNomor[s.nomor] = s; });

    var lines = [];
    var count = 0;
    clips.forEach(function (clip) {
      var seg = byNomor[clip.nomor];
      if (!seg || !seg.narasi) { return; }

      var wordCount = seg.narasi.trim().split(/\s+/).length;
      var lead = Math.min(0.5, clip.length / 4);
      var speak = Math.min(clip.length - lead - 0.2, wordCount / WORDS_PER_SECOND);
      if (speak <= 0) { return; }

      var cues = splitCues(seg.narasi);
      var totalChars = 0;
      cues.forEach(function (c) { totalChars += c.length; });
      if (!totalChars) { return; }

      var t = clip.start + lead;
      cues.forEach(function (text) {
        var dur = speak * (text.length / totalChars);
        count++;
        lines.push(String(count), srtTime(t) + ' --> ' + srtTime(t + dur), text, '');
        t += dur;
      });
    });
    return { text: lines.join('\r\n'), count: count };
  }

  function writeSubtitle(clips) {
    var srt = buildSrt(clips);
    if (!srt.count) { addReport('Subtitle tidak dibuat: tidak ada narasi.', true); return; }

    var srtName = state.baseName + '.srt';
    var path = joinPath(dirName(state.jsonPath), srtName);
    var res = window.cep.fs.writeFile(path, srt.text);
    if (res.err !== window.cep.fs.NO_ERROR) {
      addReport(srtName + ' gagal ditulis (kode ' + res.err + ').', true);
      return;
    }
    addReport(srtName + ' dibuat: ' + srt.count + ' baris, di ' + path);

    evalHost('smeImportFile(' + toLiteral(path) + ')').then(function (r) {
      if (r.ok && r.data.imported) {
        addReport('Subtitle diimpor ke panel Project. Seret ke timeline tepat di 00:00.');
      } else {
        addReport('Subtitle tidak bisa diimpor otomatis' + (r.ok ? '' : ' (' + r.error + ')') +
          '. Impor manual lewat File > Import.', true);
      }
    });
  }

  // ---------- Ingat nilai terakhir ----------

  function remember(id, isCheck) {
    var el = $(id);
    var key = 'sme.' + id;
    try {
      var saved = window.localStorage.getItem(key);
      if (saved !== null) {
        if (isCheck) { el.checked = saved === '1'; } else { el.value = saved; }
      }
      el.addEventListener('change', function () {
        window.localStorage.setItem(key, isCheck ? (el.checked ? '1' : '0') : el.value);
      });
    } catch (e) { /* tanpa penyimpanan, pakai nilai bawaan */ }
  }
  remember('pad', false);
  remember('gap', false);
  remember('make-srt', true);

  $('btn-build').addEventListener('click', buildSequence);

  $('btn-pick').addEventListener('click', function () {
    var path = pickFile('Pilih highlight.json', '', ['json']);
    if (path) { loadHighlight(path); }
  });

  $('btn-pick-video').addEventListener('click', function () {
    var start = state.jsonPath ? dirName(state.jsonPath) : '';
    var path = pickFile('Pilih video asli', start, ['mp4', 'mov', 'mkv']);
    if (path) { state.videoPath = path; render(); }
  });

  checkHost();
  render();
})();
