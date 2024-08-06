A distributed file system implemented in go.
-> tcp p2p connection to share files

Functional Requirements :-
-- Implemented --
1. Save file and distribute to all its peers. (✔)
2. Get file (✔)
    -> if present, serve from local storage
    -> if not found, broadcast a message to fetch it from peers, fetch it
3. Added usedId to file path, so user can fetch all its files even in case he lost it (don't have   key) (✔)
--- To implement ---
1. Delete file and delete the same in all the peers (✗)
2. Currently, we are hard coding the connectivity, should implement auto connectivity with all its registered peers. (✗)
