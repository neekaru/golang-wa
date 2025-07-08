# WhatsApp API Documentation

This document provides documentation for all available API endpoints and their corresponding curl commands for testing.

## Session Management

### 1. Create New Session
Creates a new WhatsApp session for a user.

```bash
curl -X POST http://localhost:8080/wa/add \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

### 2. Get QR Code
Get QR code for WhatsApp Web authentication. This endpoint will only generate a QR code if the user is not already logged in and connected.

```bash
curl -X GET "http://localhost:8080/wa/qr-image?user=test_user"
```

**Success Response:**
```json
{
  "qrcode": "data:image/png;base64,..."
}
```

**Error Response (when already logged in):**
```json
{
  "error": "Session is already logged in and connected. No QR code needed.",
  "status": {
    "logged_in": true,
    "connected": true,
    "user": "test_user"
  }
}
```

### 3. Check Session Status
Check if a session is connected and authenticated. Returns detailed status information.

```bash
curl -X GET "http://localhost:8080/wa/status?user=test_user"
```

**Response:**
```json
{
  "logged_in": true,
  "connected": true,
  "user": "test_user",
  "needs_qr": false,
  "timestamp": "2023-09-15T12:34:56Z"
}
```

The `needs_qr` field indicates whether the client should request a new QR code (true if either logged_in is false or connected is false).

### 4. Restart Session
Restart an existing session. Returns detailed status after restart.

```bash
curl -X POST "http://localhost:8080/wa/restart?user=test_user"
```

**Success Response:**
```json
{
  "msg": "Session restored and connected successfully",
  "status": {
    "logged_in": true,
    "connected": true,
    "user": "test_user",
    "needs_qr": false
  }
}
```

**Error Response:**
```json
{
  "error": "Failed to connect restored session: connection error",
  "status": {
    "logged_in": false,
    "connected": false,
    "user": "test_user"
  }
}
```

### 5. Logout Session
Logout and remove a session.

```bash
curl -X POST http://localhost:8080/wa/logout \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

## Messaging

### 1. Send Text Message
Send a text message to a WhatsApp number.

```bash
curl -X POST http://localhost:8080/send \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "message": "Hello, World!"
  }'
```

### 2. Send Image
Send an image with optional caption. The image can be provided as base64 encoded data or a URL.

```bash
# Using base64 encoded image data
curl -X POST http://localhost:8080/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "media": "BASE64_ENCODED_IMAGE_DATA",
    "caption": "Check out this image!"
  }'

# Using a URL
curl -X POST http://localhost:8080/send/image \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "url": "https://example.com/image.jpg",
    "caption": "Check out this image!"
  }'
```

### 3. Send Video
Send a video with optional caption. The video can be provided as base64 encoded data or a URL.

```bash
# Using base64 encoded video data
curl -X POST http://localhost:8080/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "media": "BASE64_ENCODED_VIDEO_DATA",
    "caption": "Check out this video!"
  }'

# Using a URL
curl -X POST http://localhost:8080/send/video \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "url": "https://example.com/video.mp4",
    "caption": "Check out this video!"
  }'
```

### 4. Send File
Send any type of file with optional caption. The file can be provided as base64 encoded data or a URL.

```bash
# Using base64 encoded file data
curl -X POST http://localhost:8080/send/file \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "media": "BASE64_ENCODED_FILE_DATA",
    "caption": "Here's the document!"
  }'

# Using a URL
curl -X POST http://localhost:8080/send/file \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "phone_number": "1234567890",
    "url": "https://example.com/document.pdf",
    "caption": "Here's the document!"
  }'
```

### 5. Mark Messages as Read
Mark one or more messages as read.

```bash
curl -X POST http://localhost:8080/msg/read \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user",
    "message_ids": ["MESSAGE_ID_1", "MESSAGE_ID_2"],
    "from_jid": "SENDER_PHONE_NUMBER",
    "to_jid": "RECIPIENT_PHONE_NUMBER"
  }'
```

## Health Check Endpoints

### 1. Root Health Check
Simple health check endpoint with basic information.

```bash
curl -X GET http://localhost:8080/
```

Response:
```json
{
  "status": "ok",
  "uptime": "3h5m10s",
  "session_count": 2,
  "version": "1.0.2"
}
```

### 2. Detailed Health Check
Detailed health check with session information.

```bash
curl -X GET http://localhost:8080/health
```

Response:
```json
{
  "status": "ok",
  "uptime": "3h5m10s",
  "total_sessions": 2,
  "active_sessions": 1,
  "timestamp": "2023-09-15T12:34:56Z"
}
```

## Connection Handling Details

The WhatsApp API implements robust connection handling with the following features:

1. **Automatic Reconnection:** The system attempts to reconnect automatically when disconnections occur.

2. **QR Code Generation Logic:**
   - QR codes are only generated when needed (when a user is not logged in or not connected)
   - The API prevents generating QR codes for already connected sessions
   - The `needs_qr` field in status responses indicates when to request a new QR code

3. **Connection Error Recovery:**
   - The system includes retry logic for connection errors
   - Special handling for "websocket is already connected" errors
   - Ensures proper disconnection before attempting to reconnect

4. **Session State Management:**
   - Proper tracking of both login state and connection state
   - Detailed logging of connection state changes
   - Clear distinction between logged_in and connected states

## Response Format

All endpoints return JSON responses with consistent formats:

### Success Response
```json
{
    "msg": "Success message"
}
```

### Error Response
```json
{
    "error": "Error message"
}
```

### Status Response
Many endpoints now include a detailed status object:
```json
{
    "status": {
        "logged_in": true,
        "connected": true,
        "user": "test_user",
        "needs_qr": false
    }
}
```

## Contact Management

### 1. Get All Contacts
Retrieve all contacts for a user (both saved and unsaved).

```bash
curl -X POST http://localhost:8080/contact \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

**Success Response:**
```json
{
  "contacts": [
    {
      "jid": "1234567890@s.whatsapp.net",
      "phone_number": "1234567890",
      "name": "John Doe",
      "push_name": "John",
      "business_name": "",
      "is_saved": true,
      "is_business": false
    },
    {
      "jid": "0987654321@s.whatsapp.net",
      "phone_number": "0987654321",
      "name": "",
      "push_name": "Unknown User",
      "business_name": "",
      "is_saved": false,
      "is_business": false
    }
  ],
  "total": 2,
  "user": "test_user"
}
```

### 2. Get Saved Contacts
Retrieve only contacts that have been saved (have names).

```bash
curl -X POST http://localhost:8080/contact/saved \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

### 3. Get Unsaved Contacts
Retrieve only contacts that haven't been saved (no names).

```bash
curl -X POST http://localhost:8080/contact/unsaved \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

### 4. Refresh Contacts
Force refresh the contact list from WhatsApp servers.

```bash
curl -X POST http://localhost:8080/contact/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "user": "test_user"
  }'
```

**Success Response:**
```json
{
  "msg": "Contacts refreshed successfully",
  "user": "test_user"
}
```

## Important Notes

1. Replace `test_user` with your actual user identifier
2. Phone numbers should be in international format without any special characters (e.g., "1234567890")
3. For media uploads, you can use either:
   - Base64 encoded data with the `media` parameter
   - Direct URL to the media with the `url` parameter
4. Session must be created and authenticated before sending messages
5. All endpoints run on `localhost:8080` by default
6. The server is designed to handle multiple WhatsApp sessions simultaneously
7. Sessions are persisted to the filesystem in the "data" directory
8. The server supports graceful shutdown when receiving SIGINT or SIGTERM signals
9. Connection issues are automatically handled with retry mechanisms
10. Check the `needs_qr` field in status responses to determine if a QR code is needed