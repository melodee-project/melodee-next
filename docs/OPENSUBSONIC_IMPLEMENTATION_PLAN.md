# OpenSubsonic API Implementation Plan

This document outlines the missing OpenSubsonic endpoints required to achieve full API compliance. The implementation is broken down into phases based on priority and feature grouping.

## Phase 0: System & Authentication
These endpoints are fundamental for server connectivity, user authentication, and system status.

- [x] `ping` - Used to test connectivity with the server.
- [x] `getLicense` - Get details about the software license.
- [x] `getOpenSubsonicExtensions` - Get information about supported OpenSubsonic extensions.
- [x] `getUser` - Get details about a specific user.
- [x] `getUsers` - Get details about all users.
- [x] `createUser` - Create a new user.
- [x] `updateUser` - Update a user.
- [x] `deleteUser` - Delete a user.
- [x] `changePassword` - Changes the password of an existing user.
- [x] `tokenInfo` - Returns information about an API key.

## Phase 1: Core Discovery & Lists
These endpoints are critical for clients to browse the library effectively beyond simple directory structures. They allow for "Newest", "Random", and "Genre" based browsing.

- [x] `getMusicFolders` - Returns all configured music folders.
- [x] `getIndexes` - Returns an indexed structure of the music library.
- [x] `getMusicDirectory` - Returns a listing of the music directory.
- [x] `getGenres` - Returns all genres.
- [x] `getArtists` - Returns all artists.
- [x] `getArtist` - Returns details for an artist.
- [x] `getAlbum` - Returns details for an album.
- [x] `getSong` - Returns details for a song.
- [x] `getAlbumList` - Returns a list of random, newest, highest rated etc. albums.
- [x] `getAlbumList2` - Returns a list of random, newest, highest rated etc. albums (Version 2).
- [x] `getRandomSongs` - Returns random songs matching the given criteria.
- [x] `getSongsByGenre` - Returns songs in a given genre.
- [x] `getNowPlaying` - Returns what is currently being played by all users.
- [x] `getTopSongs` - Returns top songs for the given artist.
- [x] `getSimilarSongs` - Returns a random collection of songs from the given artist and similar artists.
- [x] `getSimilarSongs2` - Returns a random collection of songs from the given artist and similar artists (Version 2).
- [x] `search` - Returns a listing of files matching the given search criteria.
- [x] `search2` - Returns a listing of files matching the given search criteria (Version 2).
- [x] `search3` - Returns a listing of files matching the given search criteria (Version 3).

## Phase 2: User Interaction & Metadata
These endpoints enable user personalization (starring, rating) and richer metadata display (biographies, lyrics).

- [x] `star` - Attaches a star to a song, album or artist.
- [x] `unstar` - Removes a star from a song, album or artist.
- [x] `getStarred` - Returns starred songs, albums and artists.
- [x] `getStarred2` - Returns starred songs, albums and artists (Version 2).
- [x] `setRating` - Sets the rating for a music file.
- [x] `scrobble` - Registers the local playback of one or more media files.
- [x] `getAlbumInfo` - Returns album info (notes, etc).
- [x] `getAlbumInfo2` - Returns album info (notes, etc) (Version 2).
- [x] `getArtistInfo` - Returns artist info (bio, image).
- [x] `getArtistInfo2` - Returns artist info (bio, image) (Version 2).
- [x] `getLyrics` - Searches for and returns lyrics for a given song.
- [x] `getLyricsBySongId` - Retrieval of lyrics by song ID.

## Phase 3: Playlists & Bookmarks
These endpoints handle user state persistence, bookmarks (resume points), and play queues.

- [x] `getPlaylists` - Returns all playlists a user is allowed to play.
- [x] `getPlaylist` - Returns a listing of files in a playlist.
- [x] `createPlaylist` - Creates (or updates) a playlist.
- [x] `updatePlaylist` - Updates a playlist.
- [x] `deletePlaylist` - Deletes a playlist.
- [x] `createBookmark` - Creates or updates a bookmark (resume point).
- [x] `deleteBookmark` - Deletes a bookmark.
- [x] `getBookmarks` - Returns all bookmarks for this user.
- [x] `getPlayQueue` - Returns the state of the play queue for this user.
- [x] `savePlayQueue` - Saves the state of the play queue for this user.
- [x] `getPlayQueueByIndex` - Returns the state of the play queue for this user (by index).
- [x] `savePlayQueueByIndex` - Saves the state of the play queue for this user (by index).

## Phase 4: Podcasts & Internet Radio
Features for managing external media sources.

- [ ] `getPodcasts` - Returns all Podcast channels.
- [ ] `getNewestPodcasts` - Returns the most recently published Podcast episodes.
- [ ] `refreshPodcasts` - Requests the server to check for new Podcast episodes.
- [ ] `createPodcastChannel` - Adds a new Podcast channel.
- [ ] `deletePodcastChannel` - Deletes a Podcast channel.
- [ ] `getPodcastEpisode` - Returns details for a podcast episode.
- [ ] `downloadPodcastEpisode` - Request the server to start downloading a given Podcast episode.
- [ ] `deletePodcastEpisode` - Deletes a Podcast episode.
- [ ] `getInternetRadioStations` - Returns all internet radio stations.
- [ ] `createInternetRadioStation` - Adds a new internet radio station.
- [ ] `updateInternetRadioStation` - Updates an existing internet radio station.
- [ ] `deleteInternetRadioStation` - Deletes an existing internet radio station.

## Phase 5: Media Retrieval & Advanced
Sharing, Video, Chat, System control, and specialized streaming.

- [x] `stream` - Streams a media file.
- [x] `download` - Downloads a media file.
- [x] `getCoverArt` - Returns a cover art image.
- [x] `getAvatar` - Returns the avatar for a user.
- [ ] `hls.m3u8` - HLS Streaming endpoint (if not covered by stream.view).
- [ ] `getShares` - Returns information about shared media.
- [ ] `createShare` - Creates a public URL for sharing.
- [ ] `updateShare` - Updates a share.
- [ ] `deleteShare` - Deletes a share.
- [ ] `getVideoInfo` - Returns details for a video.
- [ ] `getVideos` - Returns all video files.
- [ ] `getCaptions` - Returns captions for a video.
- [ ] `getChatMessages` - Returns chat messages.
- [ ] `addChatMessage` - Adds a message to the chat log.
- [ ] `getScanStatus` - Returns the current status for media library scanning.
- [ ] `startScan` - Initiates a rescan of the media libraries.
- [ ] `jukeboxControl` - Controls the jukebox (server-side playback).

## Definition of Done

For an endpoint to be considered "Done", it must meet the following criteria:

1.  **Implementation**: The endpoint is fully implemented in Go and registered in the router.
2.  **Compliance**: The response format matches the OpenSubsonic specification (XML and JSON).
3.  **Unit Tests**: Unit tests are created for the endpoint, covering success and error scenarios.
4.  **Documentation**: The endpoint is marked as completed in this plan.

**Agent Instruction**: When you complete a task or set of tasks, you MUST update this document by checking the corresponding box (change `[ ]` to `[x]`).

**Verification Protocol for Agents**:
Before marking an item as complete, ensure:
1.  **Code Implementation**: The handler function exists in the codebase (e.g., `src/open_subsonic/handlers/`).
2.  **Route Registration**: The endpoint is registered in the router (usually in `src/open_subsonic/main.go` or similar).
3.  **Test Coverage**: A corresponding test exists in the `handlers` package (e.g., `phase2_test.go`, `browsing_test.go`) and passes successfully.
4.  **Schema Compatibility**: Ensure any new tests use the standard "Manual SQLite Schema" pattern to avoid Postgres/SQLite type conflicts.

Only when all these conditions are met should the item be marked as `[x]`.
