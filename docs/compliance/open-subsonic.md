# OpenSubsonic Compatibility Notes (Home‑lab)

This document tracks endpoint compatibility and deviations.

## Scope
- Authentication
- Browsing
- Search
- Playlists
- Starred/Favorites
- Ratings
- Now-playing
- Scrobbling
- Transcoding parameters
- Paging

## Status (initial)
- Target: Interoperable with common OpenSubsonic clients for the listed endpoints.
- Extras (namespaced): replaygain info, fingerprint match status, proposals queue, device capabilities (for first‑party clients).
	- User engagement extras: like/dislike (thumbs up/down) for tracks/albums/artists.

## Notes
- Progressive stream remains for third-party clients.
- First-party apps prefer HLS via extras endpoints.
- This is a home‑lab server; behavior aims for practicality and stability over exhaustive spec coverage.

## Mappings
- Favorites ↔ OpenSubsonic star/unstar
	- Supported for songs, albums, and artists where the client passes the appropriate IDs
- Ratings ↔ OpenSubsonic setRating (0–5)
	- 0 clears the rating; 1–5 sets the rating
- Like/Dislike
	- Not defined in OpenSubsonic; exposed via first‑party extras endpoints only
