# Go WebSocket Service

This service provides a WebSocket server that pushes notifications to connected clients. It consumes messages from RabbitMQ and forwards them to specific users or broadcasts them.

## Features

- **WebSocket Server**: Uses `gorilla/websocket` for handling connections.
- **RabbitMQ Consumer**: Listens for events from RabbitMQ and pushes them to clients.
- **JWT Authentication**: Validates RSA-signed JWT tokens using a public key.
- **Entity Differentiation**: Distinguishes between `customer` and `user` (admin) by prefixing IDs (e.g., `customer:1`, `user:1`).
- **Targeted Push**: Can send messages to a specific user based on their entity type and ID.

## Prerequisites

- Go 1.20+
- RabbitMQ running locally or remotely.
- `public.pem`: An RSA Public Key file (extracted from `ai-cart-api`) must be present in the root directory.

## Configuration

The service is configured using environment variables or `.env` file.

| Variable | Default | Description |
|---|---|---|
| `RABBITMQ_URL` | `amqp://guest:guest@localhost:5672/` | RabbitMQ connection URL |
| `RABBITMQ_QUEUE` | `notifications` | Name of the queue to consume from |
| `JWT_PUBLIC_KEY_PATH` | `public.pem` | Path to the RSA Public Key file |
| `PORT` | `8080` | Port to listen on |

## Running the Service

```bash
go run main.go
```

The server will start on port 8080 (or configured PORT).

## Connecting from Client

1.  **Get Token**: Call `GET https://api.aicart.store/customers/auth/ws-token` (authenticated with your user Bearer token).
    - Response: `{ "data": { "token": "eyJ..." } }`
2.  **Connect**:
    ```
    ws://localhost:8080/ws?token=<YOUR_WS_TOKEN>
    ```

The token is short-lived (60 seconds) and is only used for the initial handshake.

## Testing with Producer

A test producer script is provided in `cmd/producer`.

```bash
# Send a message to a CUSTOMER (ID: 123)
go run cmd/producer/main.go -user "123" -type "customer" -msg "Hello Customer 123"

# Send a message to an ADMIN USER (ID: 456)
go run cmd/producer/main.go -user "456" -type "user" -msg "Hello Admin 456"
```

The service uses the `groups` claim from the JWT token to determine the entity type (`customer` or `user`) and prefixes it to the ID.

## HTTP Gateway API (Optional RabbitMQ)

**Health Check**:
`GET /health` -> Returns `{"status":"ok"}` (200 OK)

You can push messages directly via HTTP POST without RabbitMQ.

**Endpoint**: `POST /api/send`
**Headers**:
- `X-API-Secret`: Your configured `API_SECRET` (in `.env`)
- `Content-Type`: `application/json`

**Body**:
```json
{
  "user_id": "customer:1",
  "data": "Your Hello Message"
}
```

**Example Curl**:
```bash
curl -X POST http://localhost:8080/api/send \
  -H "X-API-Secret: mysecuresecret" \
  -H "Content-Type: application/json" \
  -d '{"user_id": "customer:1", "data": "Hello via API"}'
```

This makes RabbitMQ **optional**. If `RABBITMQ_URL` is not set in `.env`, the consumer will simply not start, and the service will rely solely on this API.

## Chat Application Usage

Yes, this service is perfect for building a **Real-time Chat App**.

**Architecture:**
1.  **Frontend (Next.js)**: User sends a chat message to your backend API (`/api/chat/send`).
2.  **Backend (Java/Next.js)**:
    -   Validates the message.
    -   Saves it to the Database (Postgres/Mongo).
    -   **Pushes** the message to the recipient using this WebSocket Service (via RabbitMQ or `POST /api/send`).
3.  **WebSocket Service**: Delivers the message instantly to the recipient's active connection.

This "Pusher-style" architecture decouples your chat logic/persistence from the real-time delivery mechanism.

## Next.js Integration Guide

To integrate this WebSocket service with your Next.js application, follow these steps using Server Actions and Client Components.

### 1. Create a Server Action to Fetch Token

Create a server action (e.g., `src/actions/websocket.ts`) to fetch the short-lived WebSocket token from your Java Backend.

```typescript
// src/actions/websocket.ts
'use server'

import { cookies } from 'next/headers'

export async function getWebsocketToken() {
  const cookieStore = cookies()
  const accessToken = cookieStore.get('accessToken')?.value // Adjust based on your auth storage

  if (!accessToken) {
    return null
  }

  try {
    // Call the Java Backend API to get the WS token
    // For Customers: /customers/auth/ws-token
    // For Admins: /auth/ws-token
    const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/customers/auth/ws-token`, {
      headers: {
        'Authorization': `Bearer ${accessToken}`,
      },
      cache: 'no-store'
    })

    if (!res.ok) {
      console.error('Failed to fetch WS token', await res.text())
      return null
    }

    const data = await res.json()
    return data.data.token // The short-lived JWT
  } catch (error) {
    console.error('Error fetching WS token:', error)
    return null
  }
}
```

### 2. Create a WebSocket Provider (Client Component)

Create a client component (e.g., `src/components/providers/websocket-provider.tsx`) to manage the connection.

```tsx
// src/components/providers/websocket-provider.tsx
'use client'

import { useEffect, useState } from 'react'
import { getWebsocketToken } from '@/actions/websocket'

export function WebsocketProvider({ children }: { children: React.ReactNode }) {
  const [socket, setSocket] = useState<WebSocket | null>(null)

  useEffect(() => {
    let ws: WebSocket | null = null

    const connect = async () => {
      const token = await getWebsocketToken()
      if (!token) return

      // Connect to Go WebSocket Service
      const verifiedWsUrl = process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080'
      ws = new WebSocket(`${verifiedWsUrl}/ws?token=${token}`)

      ws.onopen = () => {
        console.log('Connected to Notification Service')
      }

      ws.onmessage = (event) => {
        const message = JSON.parse(event.data)
        console.log('New Notification:', message)
        // Handle notification (e.g., show toast, update store)
      }

      ws.onclose = () => {
        console.log('Disconnected from Notification Service')
      }
      setSocket(ws)
    }

    connect()

    return () => {
      if (ws) ws.close()
    }
  }, [])

  return <>{children}</>
}
```

### 3. Wrap Your Application

Wrap your application or specific layouts with the provider in `layout.tsx`.

```tsx
// src/app/layout.tsx
import { WebsocketProvider } from '@/components/providers/websocket-provider'

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html>
      <body>
        <WebsocketProvider>
          {children}
        </WebsocketProvider>
      </body>
    </html>
  )
}
```

