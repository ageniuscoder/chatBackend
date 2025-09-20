# MmChat Backend

A robust and scalable chat backend built with **Go**, featuring **real-time messaging** using WebSockets and a **PostgreSQL/SQLite** database for persistence.

---

## Table of Contents

1. [Technologies Used](#technologies-used)  
2. [Getting Started](#getting-started)  
3. [Configuration](#configuration)  
4. [Running the Application](#running-the-application)  
5. [Authentication](#authentication)  
6. [API Endpoints](#api-endpoints)  
   - Public Endpoints  
   - Protected Endpoints  
7. [WebSocket API](#websocket-api)  
8. [License](#license)  

---

## Technologies Used

- **Go** (≥1.25)  
- **Gin** (HTTP framework)  
- **Gorilla/WebSocket**  
- **lib/pq** (PostgreSQL driver)  
- **golang-jwt/jwt**  
- **joho/godotenv**  
- **go-playground/validator/v10**  

---

## Getting Started

### Prerequisites

- Go installed (≥1.25)  
- PostgreSQL or SQLite database instance  

### Installation

```bash
git clone https://github.com/ageniuscoder/chatBackend.git
cd chatBackend
go mod tidy
Configuration
Create a .env file in the project root:

Variable	Default	Description
HTTP_ADDR	:8080	HTTP server listen address
JWT_SECRET	mangal	Secret for signing JWTs
JWT_TTL_MIN	1440	JWT token expiration time in minutes
DATABASE_URL	""	PostgreSQL connection string
OTP_DIGITS	6	Number of digits in OTP
OTP_TTL_SEC	300	OTP expiration time in seconds
SENDGRID_API_KEY	""	SendGrid API key
SENDGRID_FROM	""	Verified sender email for SendGrid

Running the Application
bash
Copy code
# Run database migrations
go run ./cmd/mmchat -migrate

# Start server
go run ./cmd/mmchat
Server listens on the address specified in HTTP_ADDR (default :8080).

Authentication
JWT token is stored in an HTTP-only cookie named token.

Frontend does not need to set an Authorization header.

Ensure cookies are sent with requests:

javascript
Copy code
// Using fetch
fetch('/api/me', { credentials: 'include' });

// Using axios
axios.get('/api/me', { withCredentials: true });
Logging out clears the cookie via /api/logout.

Note: If using cross-origin requests, configure CORS with AllowCredentials: true on the backend.

API Endpoints
All endpoints are under /api.

Public endpoints do not require JWT cookie.

Protected endpoints require JWT cookie (token).

Public Endpoints
POST /api/signup/initiate
Start user signup and send OTP.

Request Body:

json
Copy code
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "StrongPassword123"
}
Response:

json
Copy code
{
  "success": true,
  "message": "OTP sent"
}
POST /api/signup/verify
Verify OTP and finalize signup.

Request Body:

json
Copy code
{
  "username": "alice",
  "email": "alice@example.com",
  "password": "StrongPassword123",
  "otp": "123456"
}
Response:

json
Copy code
{
  "success": true,
  "user_id": 42
}
POST /api/login
Authenticate user and set JWT cookie.

Request Body:

json
Copy code
{
  "username": "alice",
  "password": "StrongPassword123"
}
Response:

json
Copy code
{
  "success": true,
  "user_id": 42
}
POST /api/logout
Clear JWT cookie to log out user.

Response:

json
Copy code
{
  "success": true,
  "message": "Logged out successfully"
}
POST /api/forgot/initiate
Initiate password reset with OTP.

Request Body:

json
Copy code
{
  "email": "alice@example.com"
}
Response:

json
Copy code
{
  "success": true,
  "message": "OTP sent"
}
POST /api/forgot/reset
Complete password reset using OTP.

Request Body:

json
Copy code
{
  "email": "alice@example.com",
  "otp": "123456",
  "new_password": "NewStrongPassword456"
}
Response:

json
Copy code
{
  "success": true,
  "message": "Password updated"
}
Protected Endpoints (require JWT cookie)
GET /api/me
Get the logged-in user's profile.

Response:

json
Copy code
{
  "success": true,
  "id": 42,
  "username": "alice",
  "email": "alice@example.com",
  "profile_picture": "https://cdn.example.com/avatars/alice.jpg",
  "created_at": "2025-09-20T12:34:56Z"
}
PUT /api/me
Update user profile.

Request Body:

json
Copy code
{
  "username": "alice_new",
  "profile_picture": "https://cdn.example.com/avatars/alice_new.jpg"
}
Response:

json
Copy code
{
  "success": true,
  "id": 42,
  "username": "alice_new",
  "email": "alice@example.com",
  "profile_picture": "https://cdn.example.com/avatars/alice_new.jpg",
  "created_at": "2025-09-20T12:34:56Z"
}
GET /api/users/search?q=<string>
Search users by username substring.

Response:

json
Copy code
{
  "success": true,
  "users": [
    {
      "id": 43,
      "username": "alice_wonder",
      "profile_picture": "https://cdn.example.com/avatars/alice_wonder.jpg"
    }
  ]
}
GET /api/users/:id/last-seen
Get last active timestamp of a user.

Response:

json
Copy code
{
  "success": true,
  "last_seen": "2025-09-20T15:00:00Z"
}
GET /api/conversations
List conversations for the authenticated user.

Response Example:

json
Copy code
{
  "success": true,
  "conversations": [
    {
      "conversation_id": 100,
      "is_group": false,
      "name": "alice & bob",
      "profile_picture": "https://cdn.example.com/group_avatars/100.jpg",
      "last_message": {
        "sender_id": 43,
        "content": "Hello!",
        "created_at": "2025-09-20T14:55:00Z"
      },
      "unread_count": 2,
      "other_user_online": true
    }
  ]
}
POST /api/conversations/private
Create or get private conversation.

Request Body:

json
Copy code
{
  "other_user_id": 43
}
Response:

json
Copy code
{
  "conversation_id": 100,
  "is_group": false
}
POST /api/conversations/group
Create a new group chat.

Request Body:

json
Copy code
{
  "name": "Study Group",
  "member_ids": [42, 43, 44]
}
Response:

json
Copy code
{
  "success": true,
  "conversation_id": 101,
  "is_group": true
}
POST /api/conversations/:id/participants
Add participant (admin only).

Request Body:

json
Copy code
{
  "user_id": 45
}
Response:

json
Copy code
{
  "success": true
}
DELETE /api/conversations/:id/participants/:userId
Remove participant (admin only).

Response:

json
Copy code
{
  "success": true
}
GET /api/conversations/:id/participants
List conversation participants.

Response:

json
Copy code
{
  "success": true,
  "participants": [
    {
      "id": 42,
      "username": "alice",
      "profile_picture": "https://cdn.example.com/avatars/alice.jpg",
      "is_admin": true
    }
  ]
}
GET /api/conversations/:id/messages?limit=<int>&offset=<int>
List messages in a conversation (paginated).

Response:

json
Copy code
{
  "success": true,
  "messages": [
    {
      "id": 5001,
      "sender_id": 42,
      "content": "Hey, how’s it going?",
      "created_at": "2025-09-20T14:00:00Z",
      "status": "delivered",
      "edited": false
    }
  ]
}
POST /api/messages
Send a message.

Request Body:

json
Copy code
{
  "conversation_id": 100,
  "content": "Let’s meet up at 5 PM."
}
Response:

json
Copy code
{
  "message_id": 5003,
  "sender_id": 42,
  "conversation_id": 100,
  "content": "Let’s meet up at 5 PM.",
  "created_at": "2025-09-20T14:02:00Z"
}
POST /api/messages/read
Mark messages as read.

Request Body:

json
Copy code
{
  "message_ids": [5001, 5002]
}
Response:

json
Copy code
{
  "success": true,
  "message": "Marked as read"
}
PATCH /api/messages/:id
Edit a message.

Request Body:

json
Copy code
{
  "content": "Updated message content."
}
Response:

json
Copy code
{
  "success": true,
  "message_id": 5001,
  "content": "Updated message content.",
  "edited_at": "2025-09-20T14:10:00Z"
}
WebSocket API
Endpoint:

bash
Copy code
GET /api/ws?token=<JWT>
Message Types
Type	Description
message	New chat message
read_receipt	Message read notification
typing_start	Typing indicator start
typing_stop	Typing indicator stop
presence	User online/offline status
edited_message	Message edited
deleted_message	Message soft-deleted
conversation_update	Conversation updated
system_message	System notifications (user added/removed)

Example Payload:

json
Copy code
{
  "type": "message",
  "conversation_id": 100,
  "sender_id": 42,
  "content": "Hello!",
  "created_at": "2025-09-20T14:55:00Z"
}
