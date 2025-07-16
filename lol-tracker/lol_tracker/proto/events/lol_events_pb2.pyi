from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class GameStatus(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    GAME_STATUS_UNKNOWN: _ClassVar[GameStatus]
    GAME_STATUS_NOT_IN_GAME: _ClassVar[GameStatus]
    GAME_STATUS_IN_CHAMPION_SELECT: _ClassVar[GameStatus]
    GAME_STATUS_IN_GAME: _ClassVar[GameStatus]
GAME_STATUS_UNKNOWN: GameStatus
GAME_STATUS_NOT_IN_GAME: GameStatus
GAME_STATUS_IN_CHAMPION_SELECT: GameStatus
GAME_STATUS_IN_GAME: GameStatus

class LoLGameStateChanged(_message.Message):
    __slots__ = ("summoner_name", "region", "previous_status", "current_status", "game_result", "event_time", "game_id", "queue_type")
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    PREVIOUS_STATUS_FIELD_NUMBER: _ClassVar[int]
    CURRENT_STATUS_FIELD_NUMBER: _ClassVar[int]
    GAME_RESULT_FIELD_NUMBER: _ClassVar[int]
    EVENT_TIME_FIELD_NUMBER: _ClassVar[int]
    GAME_ID_FIELD_NUMBER: _ClassVar[int]
    QUEUE_TYPE_FIELD_NUMBER: _ClassVar[int]
    summoner_name: str
    region: str
    previous_status: GameStatus
    current_status: GameStatus
    game_result: GameResult
    event_time: _timestamp_pb2.Timestamp
    game_id: str
    queue_type: str
    def __init__(self, summoner_name: _Optional[str] = ..., region: _Optional[str] = ..., previous_status: _Optional[_Union[GameStatus, str]] = ..., current_status: _Optional[_Union[GameStatus, str]] = ..., game_result: _Optional[_Union[GameResult, _Mapping]] = ..., event_time: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ..., game_id: _Optional[str] = ..., queue_type: _Optional[str] = ...) -> None: ...

class GameResult(_message.Message):
    __slots__ = ("won", "duration_seconds", "queue_type", "champion_played")
    WON_FIELD_NUMBER: _ClassVar[int]
    DURATION_SECONDS_FIELD_NUMBER: _ClassVar[int]
    QUEUE_TYPE_FIELD_NUMBER: _ClassVar[int]
    CHAMPION_PLAYED_FIELD_NUMBER: _ClassVar[int]
    won: bool
    duration_seconds: int
    queue_type: str
    champion_played: str
    def __init__(self, won: bool = ..., duration_seconds: _Optional[int] = ..., queue_type: _Optional[str] = ..., champion_played: _Optional[str] = ...) -> None: ...
