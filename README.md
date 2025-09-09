# MmChat Backend

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![SQLite](https://img.shields.io/badge/sqlite-%2307405e.svg?style=for-the-badge&logo=sqlite&logoColor=white)
![Gin](https://img.shields.io/badge/gin-v1.10.1-brightgreen.svg?style=for-the-badge&logo=go&logoColor=white)
![JWT](https://img.shields.io/badge/JWT-Authentication-blue.svg?style=for-the-badge)
![Websocket](https://img.shields.io/badge/WebSocket-Realtime-red.svg?style=for-the-badge)

A robust and scalable chat backend built with Go, featuring real-time messaging using WebSockets and a SQLite database for persistence.

## Features

-   **User Management**: Secure user signup, login, and password reset with OTP verification.
-   **Authentication**: JWT-based authentication for secure API access.
-   **Real-time Messaging**: Utilize WebSockets for instant message delivery and real-time user presence updates.
-   **Messaging**: Send and receive private and group messages.
-   **Message Status**: Track message delivery and read receipts.
-   **User Presence**: Get real-time updates on a user's online/offline status.
-   **Profile Management**: Update user profile details like username and profile picture.
-   **Group Chats**: Create and manage group conversations with admin and member roles.
-   **Database**: SQLite database with built-in schema migrations.

## Technologies Used

This project leverages the following core technologies:

-   **Go**: The primary programming language.
-   **Gin**: A high-performance HTTP web framework.
-   **Gorilla/WebSocket**: A powerful WebSocket library for Go.
-   **modernc.org/sqlite**: A CGo-free SQLite driver for Go.
-   **golang-jwt/jwt**: For handling JSON Web Tokens.
-   **joho/godotenv**: To load environment variables from a `.env` file.

## Getting Started

### Prerequisites

-   Go 1.25.0 or later installed.
-   An internet connection to fetch dependencies.

### Installation

1.  Clone the repository:
    ```bash
    git clone [https://github.com/ageniuscoder/mmchat.git](https://github.com/ageniuscoder/mmchat.git)
    cd mmchat
    ```
2.  Install the Go modules:
    ```bash
    go mod tidy
    ```

### Configuration

Create a `.env` file in the project root to configure the application. You can refer to the `.gitignore` file for a list of files that should not be committed, including `.env` and the database files.

The following environment variables can be set:

| Variable | Default Value | Description |
| :--- | :--- | :--- |
| `HTTP_ADDR` | `:8080` | The address to listen on. |
| `JWT_SECRET` | `mangal` | The secret key for signing JWTs. |
| `JWT_TTL_MIN` | `1440` | JWT token expiration time in minutes. |
| `SQLITE_DSN` | `file:chat.db...` | Database connection string for SQLite. |
| `OTP_DIGITS` | `6` | The number of digits for OTP codes. |
| `OTP_TTL_SEC` | `300` | OTP expiration time in seconds. |

### Running the application

1.  Run the database migrations:
    ```bash
    go run ./cmd/mmchat -migrate
    ```
    This will create the necessary tables in your SQLite database (`chat.db`).
2.  Start the server:
    ```bash
    go run ./cmd/mmchat
    ```
    The server will start and listen on the address specified in the configuration.

## API Endpoints

The API is grouped under `/api`. All protected endpoints require a JWT in the `Authorization: Bearer <token>` header.

### Public Endpoints

-   `POST /api/signup/initiate`: Initiate a new user signup with OTP verification.
-   `POST /api/signup/verify`: Verify the OTP and finalize user signup.
-   `POST /api/login`: Authenticate a user and receive a JWT.
-   `POST /api/forgot/initiate`: Initiate a password reset with OTP.
-   `POST /api/forgot/reset`: Complete the password reset with OTP and new password.
-   `GET /api/ws`: WebSocket endpoint for real-time communication.

### Protected Endpoints

-   `GET /api/me`: Get the authenticated user's profile information.
-   `PUT /api/me`: Update the authenticated user's profile.
-   `GET /api/users/:id/last-seen`: Get the last active timestamp of a user.
-   `POST /api/conversations/private`: Create or get a private conversation with another user.
-   `POST /api/conversations/group`: Create a new group conversation.
-   `POST /api/conversations/:id/participants`: Add a participant to a group chat (admin only).
-   `DELETE /api/conversations/:id/participants/:userId`: Remove a participant from a group chat (admin only).
-   `GET /api/conversations`: List all conversations for the authenticated user.
-   `POST /api/messages`: Send a new message to a conversation.
-   `GET /api/conversations/:id/messages`: List messages for a specific conversation with pagination.
-   `POST /api/messages/read`: Mark messages as read.

## WebSocket API

The WebSocket endpoint is located at `GET /api/ws?token=<JWT>`. After establishing a connection, the `hub` manages real-time events.

### Message Types

The server sends the following JSON message types to connected clients:

-   `"message"`: A new chat message.
-   `"read_receipt"`: A notification that a message has been read by another user.
-   `"typing_start"`: A user has started typing.
-   `"typing_stop"`: A user has stopped typing.
-   `"presence"`: A user's online/offline status has changed.
