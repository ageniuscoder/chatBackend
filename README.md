# MmChat Backend

A robust and scalable chat backend built with **Go**, featuring **real-time messaging** using WebSockets and a **PostgreSQL/SQLite** database for persistence.

---

## Table of Contents

1.  [Technologies Used](#technologies-used)
2.  [Getting Started](#getting-started)
3.  [Configuration](#configuration)
4.  [Running the Application](#running-the-application)
5.  [Authentication](#authentication)
6.  [API Endpoints](#api-endpoints)
    -   Public Endpoints
    -   Protected Endpoints
7.  [WebSocket API](#websocket-api)
8.  [License](#license)

---

## Technologies Used

-   **Go** (≥1.25)
-   **Gin** (HTTP framework)
-   **Gorilla/WebSocket**
-   **lib/pq** (PostgreSQL driver)
-   **golang-jwt/jwt**
-   **joho/godotenv**
-   **go-playground/validator/v10**

---

## Getting Started

### Prerequisites

-   Go installed (≥1.25)
-   PostgreSQL or SQLite database instance

### Installation

```bash
git clone [https://github.com/ageniuscoder/chatBackend.git](https://github.com/ageniuscoder/chatBackend.git)
cd chatBackend
go mod tidy  ```bash

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
