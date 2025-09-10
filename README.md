# Burn After Read

Just another implementation of a **“burn after reading”** style message sharing tool.  
Messages are stored temporarily, can be accessed once, and are then deleted.

All encryption and decryption happens **client-side** in the browser.  
The server never sees the plaintext message — it only stores the encrypted string until it is read once and then deleted.

## Features
- Temporary storage of text messages
- One-time access: messages are removed after being read
- Background cleanup of expired messages
- Simple web interface