# 💬 MmChat Backend

A robust and scalable chat backend built with **Go**, featuring **real-time messaging** using WebSockets and a **PostgreSQL/SQLite** database for persistence.

---

## 📑 Table of Contents
1. [Technologies Used](#-technologies-used)
2. [Getting Started](#-getting-started)
3. [Configuration](#-configuration)
4. [Running the Application](#-running-the-application)
5. [Authentication](#-authentication)
6. [API Overview](#-api-overview)
7. [REST Endpoints](#-rest-endpoints)
   - [Public Endpoints](#public-endpoints)
   - [Protected Endpoints](#protected-endpoints)
8. [WebSocket API](#-websocket-api)
9. [License](#-license)

---

## ⚙️ Technologies Used
- **Go** (≥1.25)
- **Gin** (HTTP framework)
- **Gorilla/WebSocket**
- **PostgreSQL / SQLite**
- **JWT (golang-jwt/jwt)**
- **SendGrid API**
- **dotenv / validator**

---

## 🚀 Getting Started

### Prerequisites
- Go installed (≥1.25)
- PostgreSQL or SQLite database

### Installation
```bash
git clone https://github.com/ageniuscoder/chatBackend.git
cd chatBackend
go mod tidy

## Configuration
Create a .env file in the root directory:

| Variable           | Default | Description                        |
| ------------------ | ------- | ---------------------------------- |
| HTTP\_ADDR         | :8080   | HTTP server listen address         |
| JWT\_SECRET        | mangal  | Secret key for JWT signing         |
| JWT\_TTL\_MIN      | 1440    | JWT expiration (minutes)           |
| DATABASE\_URL      | ""      | PostgreSQL connection string       |
| OTP\_DIGITS        | 6       | OTP length                         |
| OTP\_TTL\_SEC      | 300     | OTP expiration (seconds)           |
| SENDGRID\_API\_KEY | ""      | SendGrid API key                   |
| SENDGRID\_FROM     | ""      | Verified sender email for SendGrid |

## Running The Applicaion
# Run migrations
go run ./cmd/mmchat -migrate

# Start server
go run ./cmd/mmchat

## Authentications
JWT is stored in an HTTP-only cookie named token.
Requests must include cookies:

axios.get('/api/me', { withCredentials: true });

Logout clears cookie via /api/logout.
Enable CORS with AllowCredentials: true if cross-origin.

## API Overview
Rest EndPoints

| Method | Endpoint                                      | Auth | Description                     |
| ------ | --------------------------------------------- | ---- | ------------------------------- |
| POST   | `/api/signup/initiate`                        | ❌    | Start user signup & send OTP    |
| POST   | `/api/signup/verify`                          | ❌    | Verify OTP & finalize signup    |
| POST   | `/api/login`                                  | ❌    | Authenticate user               |
| POST   | `/api/logout`                                 | ✅    | Logout & clear JWT cookie       |
| POST   | `/api/forgot/initiate`                        | ❌    | Start password reset (OTP)      |
| POST   | `/api/forgot/reset`                           | ❌    | Reset password with OTP         |
| GET    | `/api/me`                                     | ✅    | Get user profile                |
| PUT    | `/api/me`                                     | ✅    | Update profile                  |
| GET    | `/api/users/search?q=<string>`                | ✅    | Search users by username        |
| GET    | `/api/users/:id/last-seen`                    | ✅    | Get last seen status            |
| GET    | `/api/conversations`                          | ✅    | List user conversations         |
| POST   | `/api/conversations/private`                  | ✅    | Create/get private conversation |
| POST   | `/api/conversations/group`                    | ✅    | Create group chat               |
| POST   | `/api/conversations/:id/participants`         | ✅    | Add participant (admin only)    |
| DELETE | `/api/conversations/:id/participants/:userId` | ✅    | Remove participant (admin only) |
| GET    | `/api/conversations/:id/participants`         | ✅    | List conversation participants  |
| GET    | `/api/conversations/:id/messages`             | ✅    | Get messages (paginated)        |
| POST   | `/api/messages`                               | ✅    | Send a message                  |
| POST   | `/api/messages/read`                          | ✅    | Mark messages as read           |
| PATCH  | `/api/messages/:id`                           | ✅    | Edit a message                  |

## ApiEndpoins

# Chat App API Documentation

This document provides a complete reference for the REST and WebSocket APIs for our chat application.

---

### 🔑 Authentication Endpoints

These endpoints handle user registration, login, and password management.

#### **Sign Up**

* **POST /api/signup/initiate**
    * **Description:** Starts the signup process by sending a one-time password (OTP) to the user's email.
    * **Auth Required:** ❌
    * **Request Body:**
        ```json
        {
          "username": "alice",
          "email": "alice@example.com",
          "password": "StrongPassword123"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message": "OTP sent"
        }
        ```
    * **Error Response (400):**
        ```json
        {
          "error": "email already registered"
        }
        ```

* **POST /api/signup/verify**
    * **Description:** Completes the signup process by verifying the OTP.
    * **Auth Required:** ❌
    * **Request Body:**
        ```json
        {
          "username": "alice",
          "email": "alice@example.com",
          "password": "StrongPassword123",
          "otp": "123456"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "user_id": 42
        }
        ```
    * **Error Response (400):**
        ```json
        {
          "error": "invalid or expired OTP"
        }
        ```

#### **Login & Logout**

* **POST /api/login**
    * **Description:** Authenticates a user and issues a session token.
    * **Auth Required:** ❌
    * **Request Body:**
        ```json
        {
          "username": "alice",
          "password": "StrongPassword123"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "user_id": 42
        }
        ```
    * **Error Response (400):**
        ```json
        {
          "error": "invalid credentials"
        }
        ```

* **POST /api/logout**
    * **Description:** Invalidates the current session token.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message": "Logged out successfully"
        }
        ```

#### **Forgot Password**

* **POST /api/forgot/initiate**
    * **Description:** Initiates the password reset process by sending an OTP.
    * **Auth Required:** ❌
    * **Request Body:**
        ```json
        {
          "email": "alice@example.com"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message": "OTP sent"
        }
        ```

* **POST /api/forgot/reset**
    * **Description:** Resets the password using the provided OTP.
    * **Auth Required:** ❌
    * **Request Body:**
        ```json
        {
          "email": "alice@example.com",
          "otp": "123456",
          "new_password": "NewStrongPassword456"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message": "Password updated"
        }
        ```

---

### 👤 User Profile Endpoints

These endpoints manage user profile information.

* **GET /api/me**
    * **Description:** Retrieves the authenticated user's profile.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "id": 42,
          "username": "alice",
          "email": "alice@example.com",
          "profile_picture": "[https://cdn.example.com/avatars/alice.jpg](https://cdn.example.com/avatars/alice.jpg)",
          "created_at": "2025-09-20T12:34:56Z"
        }
        ```

* **PUT /api/me**
    * **Description:** Updates the authenticated user's profile.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "username": "alice_new",
          "profile_picture": "[https://cdn.example.com/avatars/alice_new.jpg](https://cdn.example.com/avatars/alice_new.jpg)"
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "id": 42,
          "username": "alice_new",
          "email": "alice@example.com",
          "profile_picture": "[https://cdn.example.com/avatars/alice_new.jpg](https://cdn.example.com/avatars/alice_new.jpg)",
          "created_at": "2025-09-20T12:34:56Z"
        }
        ```

* **GET /api/users/search?q=<string>**
    * **Description:** Searches for users by their username.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "users": [
            {
              "id": 43,
              "username": "alice_wonder",
              "profile_picture": "[https://cdn.example.com/avatars/alice_wonder.jpg](https://cdn.example.com/avatars/alice_wonder.jpg)"
            }
          ]
        }
        ```

* **GET /api/users/:id/last-seen**
    * **Description:** Gets the last seen timestamp for a specific user.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "last_seen": "2025-09-20T15:00:00Z"
        }
        ```

---

### 💬 Conversation Endpoints

These endpoints are for managing conversations, including private and group chats.

* **GET /api/conversations**
    * **Description:** Lists all conversations for the authenticated user.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "conversations": [
            {
              "conversation_id": 100,
              "is_group": false,
              "name": "alice & bob",
              "profile_picture": "[https://cdn.example.com/group_avatars/100.jpg](https://cdn.example.com/group_avatars/100.jpg)",
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
        ```

* **POST /api/conversations/private**
    * **Description:** Creates or retrieves a private conversation with a specified user.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "other_user_id": 43
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "conversation_id": 100,
          "is_group": false
        }
        ```

* **POST /api/conversations/group**
    * **Description:** Creates a new group chat.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "name": "Study Group",
          "member_ids": [42, 43, 44]
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "conversation_id": 101,
          "is_group": true
        }
        ```

#### **Participants**

* **POST /api/conversations/:id/participants**
    * **Description:** Adds a new user to a group conversation.
    * **Auth Required:** ✅ (Admin only)
    * **Request Body:**
        ```json
        {
          "user_id": 45
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true
        }
        ```

* **DELETE /api/conversations/:id/participants/:userId**
    * **Description:** Removes a user from a group conversation.
    * **Auth Required:** ✅ (Admin only)
    * **Success Response (200):**
        ```json
        {
          "success": true
        }
        ```

* **GET /api/conversations/:id/participants**
    * **Description:** Lists all participants in a conversation.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "participants": [
            {
              "id": 42,
              "username": "alice",
              "profile_picture": "[https://cdn.example.com/avatars/alice.jpg](https://cdn.example.com/avatars/alice.jpg)",
              "is_admin": true
            }
          ]
        }
        ```

---

### 💌 Messaging Endpoints

These endpoints manage sending, retrieving, and editing messages.

* **GET /api/conversations/:id/messages?limit=<int>&offset=<int>**
    * **Description:** Fetches a paginated list of messages for a conversation.
    * **Auth Required:** ✅
    * **Success Response (200):**
        ```json
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
        ```

* **POST /api/messages**
    * **Description:** Sends a new message to a conversation.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "conversation_id": 100,
          "content": "Let’s meet up at 5 PM."
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "message_id": 5003,
          "sender_id": 42,
          "conversation_id": 100,
          "content": "Let’s meet up at 5 PM.",
          "created_at": "2025-09-20T14:02:00Z"
        }
        ```

* **POST /api/messages/read**
    * **Description:** Marks one or more messages as read.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "message_ids": [5001, 5002]
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message": "Marked as read"
        }
        ```

* **PATCH /api/messages/:id**
    * **Description:** Edits the content of an existing message.
    * **Auth Required:** ✅
    * **Request Body:**
        ```json
        {
          "content": "Updated message content."
        }
        ```
    * **Success Response (200):**
        ```json
        {
          "success": true,
          "message_id": 5001,
          "content": "Updated message content.",
          "edited_at": "2025-09-20T14:10:00Z"
        }
        ```

---

### 🔌 WebSocket API

The WebSocket API provides real-time updates for messages, presence, and other chat events.

* **Endpoint:** `/api/ws?token=<JWT>`
* **Message Types:**
    * `message`: New chat message
    * `read_receipt`: Message read notification
    * `typing_start`: Typing indicator start
    * `typing_stop`: Typing indicator stop
    * `presence`: Online/offline status
    * `edited_message`: Edited message event
    * `deleted_message`: Soft-deleted message
    * `conversation_update`: Conversation metadata updated
    * `system_message`: System notifications (e.g., join/leave)
* **Example Payload:**
    ```json
    {
      "type": "message",
      "conversation_id": 100,
      "sender_id": 42,
      "content": "Hello!",
      "created_at": "2025-09-20T14:55:00Z"
    }
    ```

---
