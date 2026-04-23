# Traveller Frontend

React frontend scaffolded with Vite in the `frontend` folder.

## Prerequisites

- Node.js 18+ (or 20+ recommended)
- npm

## Run locally

```bash
npm install
npm run dev
```

The app starts at `http://localhost:5173`.

## API base URL

The frontend reads backend URL from `VITE_API_BASE_URL`.

- Default local setup is already configured in `.env`.
- To override for another environment, update `.env` or create environment-specific Vite env files.

## Google Sign-In

Set `VITE_GOOGLE_CLIENT_ID` to the Google OAuth web client ID that should render the sign-in button:

```bash
VITE_GOOGLE_CLIENT_ID=your-google-web-client-id.apps.googleusercontent.com
```

The backend must use the same client ID through `GOOGLE_CLIENT_ID`.
For this repo, a local `frontend/.env.local` file works out of the box with Vite.

## Delhi location fallback

The Tickets page can fall back to hardcoded Delhi coordinates for QR check-in/check-out when browser geolocation is unavailable or outside Delhi.

- `VITE_FORCE_DELHI_LOCATION=true` always uses the hardcoded Delhi location.
- `VITE_DELHI_FALLBACK_LAT=28.632896` sets the fallback latitude.
- `VITE_DELHI_FALLBACK_LON=77.219574` sets the fallback longitude.

Example:

```bash
VITE_FORCE_DELHI_LOCATION=true
VITE_DELHI_FALLBACK_LAT=28.632896
VITE_DELHI_FALLBACK_LON=77.219574
```
