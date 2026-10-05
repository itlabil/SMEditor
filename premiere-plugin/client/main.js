// Panel SMEditor Highlight - tahap 1 (SM-11): sambungan ke Premiere dan baca highlight.json.
// Panel memakai API bawaan CEP (window.cep.fs dan window.__adobe_cep__), tanpa pustaka tambahan.
(function () {
  'use strict';

  var state = { jsonPath: '', videoPath: '', data: null, valid: [], problems: [] };

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
    var folder = dirName(path).replace(/^.*[\\\/]/, '');
    $('seq-name').value = 'Highlight - ' + folder;
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
    });
  }

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
