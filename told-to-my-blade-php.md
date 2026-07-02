# Passkey Pairing Flow — Panduan Frontend (PHP/Blade)

## Ringkasan Alur

Setelah user scan QR dan WhatsApp minta passkey, frontend harus:

1. **Polling** `GET /wa/passkey/status?user=X` tiap 2-3 detik
2. Begitu `pending: true` → tampilkan modal dengan tutorial + snippet
3. Admin paste snippet di **console DevTools web.whatsapp.com** → copy JSON hasilnya
4. Admin paste JSON ke textarea di modal → klik Submit
5. JS modal **POST langsung** ke `POST /wa/passkey/response?user=X`
6. Lanjut poll sampai `done: true` atau `logged_in: true`

## Kenapa Harus Buka web.whatsapp.com?

`navigator.credentials.get()` cuma bisa jalan di origin `web.whatsapp.com`. Nggak bisa dari domain admin travel kamu. Makanya admin harus buka `https://web.whatsapp.com` → F12 Console → paste snippet.

## Response `GET /wa/passkey/status`

```json
{
  "pending": true,
  "webauthn": {
    "url": "https://web.whatsapp.com",
    "public_key": { "challenge": "...", "rpId": "web.whatsapp.com", ... },
    "identity": null,
    "otp": null,
    "password": false
  },
  "snippet": "console.log(...)",
  "code": "",
  "skip_ux": false,
  "error": "",
  "done": false,
  "logged_in": false
}
```

- `webauthn` → `null` kalau nggak ada passkey pending. Field `url` + `public_key` selalu ada kalau pending.
- `identity`, `otp`, `password` → reserved buat future, WhatsApp saat ini public-key only.

## Kapan Trigger Flow Ini?

Setelah berhasil dapet QR (`GET /wa/qr-image`) dan user udah scan. Mulai polling `/wa/passkey/status` setiap 2-3 detik. Kalau `pending: true`, tampilkan modal.

## Contoh Implementasi JS

```javascript
// Konfigurasi
const USER = 'user_dari_session';
const API_BASE = 'http://localhost:8080';
let pollInterval = null;

// Fungsi polling status
async function checkPasskeyStatus() {
  try {
    const res = await fetch(`${API_BASE}/wa/passkey/status?user=${USER}`);
    const data = await res.json();

    if (data.pending && data.webauthn) {
      // Stop polling, tampilkan modal
      clearInterval(pollInterval);
      showPasskeyModal(data.webauthn);
    } else if (data.done || data.logged_in) {
      // Pairing selesai!
      clearInterval(pollInterval);
      hidePasskeyModal();
      alert('Session berhasil login!');
    } else if (data.error) {
      clearInterval(pollInterval);
      alert('Passkey error: ' + data.error);
    }
    // else: masih nunggu, polling lanjut
  } catch (err) {
    console.error('Gagal cek status passkey:', err);
  }
}

// Mulai polling setelah QR di-scan
function startPasskeyPolling() {
  pollInterval = setInterval(checkPasskeyStatus, 2500);
}

// Tampilkan modal passkey
function showPasskeyModal(webauthn) {
  const modal = document.getElementById('passkey-modal');
  const snippetEl = document.getElementById('passkey-snippet');
  const responseEl = document.getElementById('passkey-response');

  // Build snippet dari public_key
  const snippet = buildSnippet(webauthn.public_key);
  snippetEl.textContent = snippet;

  // Reset textarea
  responseEl.value = '';

  modal.style.display = 'block';
}

// Submit WebAuthn response
async function submitPasskeyResponse() {
  const responseEl = document.getElementById('passkey-response');
  const jsonText = responseEl.value.trim();

  if (!jsonText) {
    alert('Paste JSON hasil dari console dulu');
    return;
  }

  let parsed;
  try {
    parsed = JSON.parse(jsonText);
  } catch (e) {
    alert('JSON tidak valid, pastikan formatnya benar');
    return;
  }

  try {
    const res = await fetch(`${API_BASE}/wa/passkey/response?user=${USER}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(parsed),
    });
    const result = await res.json();

    if (res.ok) {
      // Response terkirim, lanjut polling
      hidePasskeyModal();
      pollInterval = setInterval(checkPasskeyStatus, 2500);
    } else {
      alert('Gagal: ' + (result.error || 'Unknown error'));
    }
  } catch (err) {
    console.error('Gagal submit passkey:', err);
    alert('Gagal mengirim response');
  }
}

function hidePasskeyModal() {
  document.getElementById('passkey-modal').style.display = 'none';
}

// Build snippet dari public_key (server juga kasih snippet di response, ini fallback)
function buildSnippet(publicKey) {
  return `console.log((await navigator.credentials.get({\n  publicKey: PublicKeyCredential.parseRequestOptionsFromJSON(${JSON.stringify(publicKey)})\n})).toJSON())`;
}

// Konfirmasi pairing code (fallback, jarang dipakai)
async function confirmPasskeyCode() {
  try {
    const res = await fetch(`${API_BASE}/wa/passkey/confirm?user=${USER}`, {
      method: 'POST',
    });
    if (res.ok) {
      pollInterval = setInterval(checkPasskeyStatus, 2500);
    } else {
      const result = await res.json();
      alert('Gagal konfirmasi: ' + (result.error || 'Unknown error'));
    }
  } catch (err) {
    console.error('Gagal konfirmasi:', err);
  }
}
```

## HTML Modal (Bootstrap)

```html
<div class="modal fade" id="passkey-modal" tabindex="-1" data-bs-backdrop="static">
  <div class="modal-dialog modal-lg">
    <div class="modal-content">
      <div class="modal-header">
        <h5 class="modal-title">Verifikasi Passkey WhatsApp</h5>
      </div>
      <div class="modal-body">

        <div class="alert alert-info">
          <strong>Langkah 1:</strong> Buka <a href="https://web.whatsapp.com" target="_blank">web.whatsapp.com</a><br>
          <strong>Langkah 2:</strong> Buka Console (tekan <kbd>F12</kbd> lalu tab <em>Console</em>)<br>
          <strong>Langkah 3:</strong> Copy-paste kode di bawah ini ke Console, lalu tekan Enter
        </div>

        <div class="bg-dark text-light p-3 rounded mb-3" style="font-family: monospace; font-size: 13px;">
          <pre id="passkey-snippet" style="white-space: pre-wrap; word-break: break-all; margin: 0;"></pre>
        </div>
        <button class="btn btn-sm btn-outline-secondary mb-3" onclick="navigator.clipboard.writeText(document.getElementById('passkey-snippet').textContent)">
          Copy Snippet
        </button>

        <div class="alert alert-warning">
          <strong>Langkah 4:</strong> Console akan mengeluarkan hasil JSON. Copy seluruh JSON-nya (dari <code>{</code> sampai <code>}</code>).
        </div>

        <div class="mb-3">
          <label class="form-label"><strong>Langkah 5:</strong> Paste hasil JSON di sini:</label>
          <textarea id="passkey-response" class="form-control" rows="8" placeholder='Paste JSON hasil console di sini...'></textarea>
        </div>

      </div>
      <div class="modal-footer">
        <button class="btn btn-primary" onclick="submitPasskeyResponse()">Submit</button>
      </div>
    </div>
  </div>
</div>
```

## Catatan Penting

1. **CORS:** Backend sudah nyalain CORS (`AllowCredentials: true`), jadi JS frontend bisa langsung POST cross-origin.
2. **Polling interval:** 2-3 detik cukup. Jangan terlalu cepat (nge-spam server).
3. **Error handling:** Kalau `error` di status terisi, tampilkan ke admin dan stop polling.
4. **`skip_ux`:** Biasanya `true` (backend auto-konfirmasi). Kalau `false` dan `code` terisi, tampilkan kode pairing ke admin, lalu admin konfirmasi di HP-nya, baru panggil `POST /wa/passkey/confirm`.
5. **Response `webauthn`:** Object `{url, public_key, identity, otp, password}` — mirror mautrix `LoginWebAuthnParams`. `identity`/`otp`/`password` reserved buat future, saat ini WhatsApp cuma pakai `public_key`.