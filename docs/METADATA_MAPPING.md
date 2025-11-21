# Metadata Mapping and Conflict Resolution

Authoritative map between database fields and tag frames/atoms, plus rewrite/conflict rules.

## Field Map
- Track
  - DB: `songs.name` → ID3 `TIT2`, Vorbis `TITLE`, MP4 `\xa9nam`
  - DB: `songs.disc_number`/`disc_total` → ID3 `TPOS` (`<disc>/<total>`), Vorbis `DISCNUMBER`/`DISCTOTAL`, MP4 `disk`
  - DB: `songs.track_number`/`track_total` → ID3 `TRCK` (`<track>/<total>`), Vorbis `TRACKNUMBER`/`TRACKTOTAL`, MP4 `trkn`
  - DB: `songs.duration` → set only in DB; do not overwrite file duration
  - DB: `songs.genre` → ID3 `TCON`, Vorbis `GENRE`, MP4 `\xa9gen`
  - DB: `songs.replaygain_track_gain`/`peak` → ID3 `TXXX:REPLAYGAIN_TRACK_GAIN`/`TXXX:REPLAYGAIN_TRACK_PEAK`, Vorbis `REPLAYGAIN_TRACK_GAIN`/`REPLAYGAIN_TRACK_PEAK`
- Album
  - DB: `albums.name` → ID3 `TALB`, Vorbis `ALBUM`, MP4 `\xa9alb`
  - DB: `albums.album_artist` → ID3 `TPE2`, Vorbis `ALBUMARTIST`, MP4 `aART`
  - DB: `albums.release_date` → ID3 `TDRL`, Vorbis `DATE`, MP4 `\xa9day`
  - DB: `albums.is_compilation` → ID3 `TCMP` (1), Vorbis `COMPILATION=1`, MP4 `cpil`
  - DB: `albums.comment` → ID3 `COMM`, Vorbis `COMMENT`, MP4 `\xa9cmt`
- Artist
  - DB: `artists.name` → ID3 `TPE1`, Vorbis `ARTIST`, MP4 `\xa9ART`
  - DB: `artists.musicbrainz_id` → ID3 `UFID:http://musicbrainz.org`, Vorbis `MUSICBRAINZ_ARTISTID`, MP4 custom `----:com.apple.iTunes:MusicBrainz Artist Id`

## Artwork Rules
- Write front cover as 600x600 JPEG quality 85 to: ID3 APIC (type 3), Vorbis PICTURE, MP4 `covr` (JPEG). If image >1MB, store as `cover.jpg` next to media and point DB `album_image` to filesystem.

## Conflict Resolution
- DB is source of truth once an item enters staging. On ingest, if on-disk tags differ from DB after user edits, rewrite file tags from DB and log `metadata.drift` metric.
- Preserve unknown/extra tags; do not strip.
- For multi-artist tracks, prefer DB `contributor` table; write primary artist only into main artist frames, others into ID3 `TXXX:CONTRIBUTOR:*` / Vorbis `CONTRIBUTOR:*`.

## Quarantine Reasons (enumeration)
- `checksum_mismatch`
- `tag_parse_error`
- `unsupported_container`
- `ffmpeg_failure`
- `path_safety`
- `validation_bounds` (duration/bitrate)
- `metadata_conflict`
- `disk_full`

## Gapless/Cues/Chapters
- If `.cue` present: store in DB as `cue_path`; do not split tracks automatically. Block promotion if referenced audio missing; quarantine `cue_missing_audio`.
- Gapless: preserve encoder delay/padding fields; do not transcode to lossy unless requested.
- Chapters (MP4/OGG): keep chapter atoms/blocks; do not drop on rewrite.
