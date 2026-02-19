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

## Configuration
You can configure the application using either command-line flags or environment variables. Flags take precedence over environment variables. 

| Flag             | Environment Variable | Description                            | Default Value        |
| ---------------- | -------------------- | -------------------------------------- | -------------------- |
| -db              | DB_PATH              | Path to the database file              | /dev/shm/messages.db |
| -default-ttl     | DEFAULT_TTL          | Default TTL for messages (**seconds**) | 86400 (24h)          |
| -max-upload-size | MAX_UPLOAD_SIZE      | Maximum upload size in **bytes**       | 10485760 (10MB)      |
| -port            | PORT                 | Port the server listens on             | 8080                 |


# Example using Docker
```yaml
docker run -d \
  -p 9000:9000 \
  -e PORT=9000 \
  -e DEFAULT_TTL=3600 \
  -e MAX_UPLOAD_SIZE=20971520 \
  --name burn-after-read \
  ghcr.io/bloodyzonk/burnafterread:latest
```