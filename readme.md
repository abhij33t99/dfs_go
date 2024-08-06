Functional Requirements :-
1. Save file and distribute to all its  (✔)
2. Get file (✔)
    -> if present, serve from local storage
    -> if not found, broadcast a message to fetch it from peers, fetch it
--- To implement ---
3. Delete file and delete the same in all the peers (✗)
4. Currently, we are hard coding the  connectivity, should implement auto connectivity with all its registered peers (✗)