from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class PlayerTrackingCommand(_message.Message):
    __slots__ = ("start_tracking", "stop_tracking")
    START_TRACKING_FIELD_NUMBER: _ClassVar[int]
    STOP_TRACKING_FIELD_NUMBER: _ClassVar[int]
    start_tracking: StartTrackingPlayer
    stop_tracking: StopTrackingPlayer
    def __init__(self, start_tracking: _Optional[_Union[StartTrackingPlayer, _Mapping]] = ..., stop_tracking: _Optional[_Union[StopTrackingPlayer, _Mapping]] = ...) -> None: ...

class StartTrackingPlayer(_message.Message):
    __slots__ = ("summoner_name", "region", "requested_at")
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    summoner_name: str
    region: str
    requested_at: _timestamp_pb2.Timestamp
    def __init__(self, summoner_name: _Optional[str] = ..., region: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StopTrackingPlayer(_message.Message):
    __slots__ = ("summoner_name", "region", "requested_at")
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    summoner_name: str
    region: str
    requested_at: _timestamp_pb2.Timestamp
    def __init__(self, summoner_name: _Optional[str] = ..., region: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...
