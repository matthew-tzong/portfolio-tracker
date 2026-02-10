# Portfolio Tracker

**Single-user personal app** — net worth and expense tracking for one owner. Aggregates your accounts via Plaid and Snaptrade. No multi-user or multi-tenant support; designed only for you.

## Stack

- **Backend**: Go (HTTP server with JWT auth)
- **Frontend**: React + TypeScript + Vite
- **Database & Auth**: Supabase (Postgres + Auth)
- **Deploy**: Single deploy (Go serves React build)

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 18+
- Supabase account (free tier)

### Setup

1. **Clone and install dependencies**:
   ```bash
   # Backend
   cd backend
   go mod tidy

   # Frontend
   cd ../frontend
   npm install
   ```

2. **Set up Supabase (single user)**:
   - Create a new project in Supabase and note:
     - Project URL
     - anon/public key
     - JWT secret (Settings → API → JWT Settings)
   - Enable Email provider under **Auth → Providers**
   - Disable public sign-ups in the Email provider settings (sign-in only)
   - Create **one** user in **Auth → Users** (this is you)

3. **Configure environment variables**:
   - Copy `.env.example` to `backend/.env` and `frontend/.env`
   - In `backend/.env`, set:
     - `SUPABASE_URL`
     - `SUPABASE_JWT_SECRET`
     - `ALLOWED_USER_EMAIL` (the email of the single Supabase user you created)
   - In `frontend/.env`, set:
     - `VITE_SUPABASE_URL`
     - `VITE_SUPABASE_ANON_KEY`
     - `VITE_API_URL` (usually `http://localhost:8080` for local dev)

4. **Run locally**:
   ```bash
   # Terminal 1: Backend
   cd backend
   go run cmd/server/main.go

   # Terminal 2: Frontend
   cd frontend
   npm run dev
   ```

5. **Open**: `http://localhost:5173`

## Project Structure

```
.
├── backend/           # Go backend
│   ├── cmd/server/   # Main entry point
│   └── internal/     # Internal packages (server, auth)
├── frontend/         # React frontend
│   └── src/
│       ├── components/
│       └── lib/      # Supabase client, API helpers
```

## Development Status

- ✅ **Slice 1**: Repo, app shell, and auth (DONE)
- ⏳ **Slice 2**: Link management (Plaid + Snaptrade) (TODO)
- ⏳ **Slice 3**: Accounts and net worth (TODO)
- ... (see [TODO.md](./TODO.md) for full list)

## Environment Variables

See `.env.example` for all required variables. Never commit `.env` files.

## License

Private project.
