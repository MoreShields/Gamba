# LoL Tracker service

lol-tracker is a Python service that polls the Riot Games API to retrieve information about game state for tracked players.
It emits events on game state changes along with other relevant game metadata.

### Onboarding tracked league players
lol-tracker expects to receive information about league of legends accounts we are interested in tracking. These will come in the form of a domain event over the message bus.
This event will indicate either that we should begin or stop tracking a player.
lol-tracker will then persist these accounts in a database to keep track of what accounts are being watched.

### Riot API polling
lol-tracker will poll the Riot API periodically for each account it is watching in it's database. When a change of game state is observed, in the case of the player starting or ending a game,
the database will be updated to record the new game state and an event will be emitted to the message bus. 

The event will contain previous and current state as well as any interesting metadata available about the game, especially the win or loss state.

