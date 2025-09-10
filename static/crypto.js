async function encryptMessage({ message, files }) {
  const enc = new TextEncoder();

  const payloadObj = {
    message: message || "",
    files: Array.isArray(files)
      ? files.map(f => ({
        filename: f.filename,
        fileData: f.fileData ? arrayBufferToBase64(f.fileData) : null
      }))
      : []
  };

  const payloadStr = JSON.stringify(payloadObj);
  const payloadBytes = enc.encode(payloadStr);

  // Generate AES-GCM key
  const key = await crypto.subtle.generateKey(
    { name: "AES-GCM", length: 256 },
    true,
    ["encrypt", "decrypt"]
  );
  const rawKey = await crypto.subtle.exportKey("raw", key);

  // Generate nonce
  const nonce = crypto.getRandomValues(new Uint8Array(12));

  // Encrypt
  const ciphertextBuffer = await crypto.subtle.encrypt(
    { name: "AES-GCM", iv: nonce },
    key,
    payloadBytes
  );

  return {
    ciphertext: uint8ToBase64(new Uint8Array(ciphertextBuffer)),
    nonce: uint8ToBase64(nonce),
    key: uint8ToBase64(new Uint8Array(rawKey))
  };
}


function uint8ToBase64(u8arr) {
  let binary = [];
  const chunkSize = 0x8000; // 32KB
  for (let i = 0; i < u8arr.length; i += chunkSize) {
    const chunk = u8arr.subarray(i, i + chunkSize);
    let chunkStr = '';
    for (let j = 0; j < chunk.length; j++) {
      chunkStr += String.fromCharCode(chunk[j]);
    }
    binary.push(chunkStr);
  }
  return btoa(binary.join(''));
}


function arrayBufferToBase64(buffer) {
  const bytes = new Uint8Array(buffer);
  let binary = [];
  const chunkSize = 0x8000; // 32KB

  for (let i = 0; i < bytes.length; i += chunkSize) {
    const chunk = bytes.subarray(i, i + chunkSize);
    let chunkStr = '';
    for (let j = 0; j < chunk.length; j++) {
      chunkStr += String.fromCharCode(chunk[j]);
    }
    binary.push(chunkStr);
  }

  return btoa(binary.join(''));
}

async function decryptMessage(ciphertextB64, nonceB64, keyB64) {
  const dec = new TextDecoder();

  // Import the AES key
  const rawKey = Uint8Array.from(atob(keyB64), c => c.charCodeAt(0));
  const key = await crypto.subtle.importKey(
    "raw",
    rawKey,
    "AES-GCM",
    true,
    ["decrypt"]
  );

  // Decode nonce and ciphertext
  const nonce = Uint8Array.from(atob(nonceB64), c => c.charCodeAt(0));
  const ciphertext = Uint8Array.from(atob(ciphertextB64), c => c.charCodeAt(0));

  // Decrypt
  const decryptedBuffer = await crypto.subtle.decrypt(
    { name: "AES-GCM", iv: nonce },
    key,
    ciphertext
  );

  // Decode UTF-8 to string
  const plaintextStr = dec.decode(decryptedBuffer);

  // Parse JSON to get message + files
  const payload = JSON.parse(plaintextStr);

  // Convert each fileData back to ArrayBuffer if it exists
  if (payload.files && Array.isArray(payload.files)) {
    payload.files = payload.files.map(f => {
      if (f.fileData) {
        f.fileData = base64ToArrayBuffer(f.fileData);
      }
      return f;
    });
  }

  return payload; 
}

function base64ToArrayBuffer(base64) {
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}


document.addEventListener('DOMContentLoaded', () => {
  const form = document.getElementById('form');
  const resultBlock = document.getElementById('result');
  const resultLink = document.getElementById('result-link');

  if (form) {
    form.addEventListener('submit', async (e) => {
      e.preventDefault();

      const message = document.getElementById('message').value.trim();
      const ttl = parseInt(document.getElementById('ttl').value, 10);

      // Get files from the input element
      const fileInput = document.getElementById('files');
      const filesArray = fileInput.files ? Array.from(fileInput.files) : [];

      // Map files to the format encryptMessage expects
      const files = await Promise.all(
        filesArray
          .filter(f => f.size > 0)
          .map(async f => ({
            filename: f.name,
            fileData: await f.arrayBuffer()
          }))
      );

      // Encrypt
      const { ciphertext, nonce, key } = await encryptMessage({ message, files });

      // Send to server
      const res = await fetch('/api/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ciphertext, nonce, ttl_seconds: ttl })
      });

      if (!res.ok) {
        alert("Error: " + await res.text());
        return;
      }

      const data = await res.json();
      const url = window.location.origin + data.view_url + '#' + key;

      // Hide the form
      form.style.display = "none";
      form.reset();
      // display link
      resultBlock.style.display = "block";

      resultLink.value = url;
      
    });
  }
});

document.addEventListener("DOMContentLoaded", () => {
  const resultBlock = document.getElementById("result");
  const copyBtn = document.getElementById("copy-btn");
  const resultLink = document.getElementById("result-link");

  if (copyBtn && resultLink) {
    copyBtn.addEventListener("click", async () => {
      try {
        await navigator.clipboard.writeText(resultLink.value);
        copyBtn.textContent = "Copied! 🔥";
        setTimeout(() => {
          copyBtn.textContent = "Copy Link";
        }, 2000);
      } catch (err) {
        console.error("Failed to copy:", err);
        alert("Copy failed. Please copy manually.");
      }
    });
  }
});
