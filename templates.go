package main

const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <title>Burn After Reading</title>
  <style>{{.Style}}</style>
  <script>{{.Script}}</script>
</head>
<body>
  <div class="container">
    <a href="/"><h1>🔥 BurnAfterRead</h1></a>

    <form id="form" class="card">
      <label for="message">Message</label>
      <textarea id="message" placeholder="Enter secret" rows="5"></textarea>

      <label for="ttl">Time to live</label>
			<select id="ttl">
				<option value="3600">1 hour</option>
				<option value="86400" selected>24 hours</option>
				<option value="604800">1 week</option>
			</select>

      <label for="files">Attach files</label>
      <input type="file" id="files" multiple>

      <button type="submit">Create Secret</button>
    </form>

    <div id="result" class="card" style="display:none;">
			<label for="result-link">Message encrypted! Share this link:</label>
		  <input type="text" id="result-link" onclick="this.select()" readonly></input>
			<button id="copy-btn">Copy Link</button>
		</div>
  </div>
</body>
</html>`

const showHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>BurnAfterRead</title>
<style>{{.Style}}</style>
<script>{{.Script}}</script>
</head>
<body>
  <div class="container">
    <a href="/"><h1>🔥 BurnAfterRead</h1></a>

    <div class="card">
			
			<label id="secret-label" for="secret">Message</label>
			<textarea id="secret" placeholder="Loading" rows="5" readonly></textarea>

			<div id="dl-label" class="label-style">Files</div>
			<div id="downloads" aria-labelledby="dl-label">
				<ul id="file-list"></ul>
			</div>
    </div>

    
  </div>

<script>
const id = {{.ID}}; // injected by template

(async () => {
  const secretEl = document.getElementById('secret');
  const container = document.getElementById("downloads");
	const fileList = document.getElementById("file-list");
	const secretLabel = document.getElementById('secret-label');

  try {
    const res = await fetch('/api/message/' + id);
    if (!res.ok) {
      secretEl.textContent = 'Error: ' + await res.text();
      return;
    }

    const { ciphertext, nonce } = await res.json();
    const key = location.hash.slice(1);

    const decrypted = await decryptMessage(ciphertext, nonce, key);

		// Success badge
		const successBadge = document.createElement('span');
		successBadge.className = "badge success";
		successBadge.textContent = "Successfully decrypted";

		secretEl.parentNode.insertBefore(successBadge, secretLabel);

    // Show message
    secretEl.value = decrypted.message || "No message.";

    // Show files
		fileList.innerHTML = "";
    if (decrypted.files && decrypted.files.length) {
      decrypted.files.forEach(f => {
        if (!f.fileData) return;

        const blob = new Blob([f.fileData]);
        const url = URL.createObjectURL(blob);

				const listItem = document.createElement("li");

        const link = document.createElement("a");
        link.href = url;
        link.download = f.filename;
        link.textContent = "🔥 Download " + f.filename;
        link.className = "download-link"; // reuse CSS

        // Revoke object URL on click
        //link.addEventListener("click", () => URL.revokeObjectURL(url));

        listItem.appendChild(link);
    		fileList.appendChild(listItem);
      });
    } else {
      container.textContent = "No files attached.";
    }

  } catch (err) {
    secretEl.textContent = 'Error decrypting message';
    console.error(err);
  }
})();
</script>
</body>
</html>`
