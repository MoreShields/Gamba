from google.protobuf import timestamp_pb2 as _timestamp_pb2
from google.protobuf.internal import enum_type_wrapper as _enum_type_wrapper
from google.protobuf import descriptor as _descriptor
from google.protobuf import message as _message
from collections.abc import Mapping as _Mapping
from typing import ClassVar as _ClassVar, Optional as _Optional, Union as _Union

DESCRIPTOR: _descriptor.FileDescriptor

class ValidationError(int, metaclass=_enum_type_wrapper.EnumTypeWrapper):
    __slots__ = ()
    VALIDATION_ERROR_UNKNOWN: _ClassVar[ValidationError]
    VALIDATION_ERROR_SUMMONER_NOT_FOUND: _ClassVar[ValidationError]
    VALIDATION_ERROR_INVALID_REGION: _ClassVar[ValidationError]
    VALIDATION_ERROR_API_ERROR: _ClassVar[ValidationError]
    VALIDATION_ERROR_RATE_LIMITED: _ClassVar[ValidationError]
    VALIDATION_ERROR_ALREADY_TRACKED: _ClassVar[ValidationError]
    VALIDATION_ERROR_NOT_TRACKED: _ClassVar[ValidationError]
    VALIDATION_ERROR_INTERNAL_ERROR: _ClassVar[ValidationError]
VALIDATION_ERROR_UNKNOWN: ValidationError
VALIDATION_ERROR_SUMMONER_NOT_FOUND: ValidationError
VALIDATION_ERROR_INVALID_REGION: ValidationError
VALIDATION_ERROR_API_ERROR: ValidationError
VALIDATION_ERROR_RATE_LIMITED: ValidationError
VALIDATION_ERROR_ALREADY_TRACKED: ValidationError
VALIDATION_ERROR_NOT_TRACKED: ValidationError
VALIDATION_ERROR_INTERNAL_ERROR: ValidationError

class StartTrackingSummonerRequest(_message.Message):
    __slots__ = ("summoner_name", "region", "requested_at")
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    summoner_name: str
    region: str
    requested_at: _timestamp_pb2.Timestamp
    def __init__(self, summoner_name: _Optional[str] = ..., region: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StartTrackingSummonerResponse(_message.Message):
    __slots__ = ("success", "summoner_details", "error_message", "error_code")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    SUMMONER_DETAILS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    summoner_details: SummonerDetails
    error_message: str
    error_code: ValidationError
    def __init__(self, success: bool = ..., summoner_details: _Optional[_Union[SummonerDetails, _Mapping]] = ..., error_message: _Optional[str] = ..., error_code: _Optional[_Union[ValidationError, str]] = ...) -> None: ...

class StopTrackingSummonerRequest(_message.Message):
    __slots__ = ("summoner_name", "region", "requested_at")
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    REQUESTED_AT_FIELD_NUMBER: _ClassVar[int]
    summoner_name: str
    region: str
    requested_at: _timestamp_pb2.Timestamp
    def __init__(self, summoner_name: _Optional[str] = ..., region: _Optional[str] = ..., requested_at: _Optional[_Union[datetime.datetime, _timestamp_pb2.Timestamp, _Mapping]] = ...) -> None: ...

class StopTrackingSummonerResponse(_message.Message):
    __slots__ = ("success", "error_message", "error_code")
    SUCCESS_FIELD_NUMBER: _ClassVar[int]
    ERROR_MESSAGE_FIELD_NUMBER: _ClassVar[int]
    ERROR_CODE_FIELD_NUMBER: _ClassVar[int]
    success: bool
    error_message: str
    error_code: ValidationError
    def __init__(self, success: bool = ..., error_message: _Optional[str] = ..., error_code: _Optional[_Union[ValidationError, str]] = ...) -> None: ...

class SummonerDetails(_message.Message):
    __slots__ = ("puuid", "account_id", "summoner_id", "summoner_name", "summoner_level", "region", "last_updated")
    PUUID_FIELD_NUMBER: _ClassVar[int]
    ACCOUNT_ID_FIELD_NUMBER: _ClassVar[int]
    SUMMONER_ID_FIELD_NUMBER: _ClassVar[int]
    SUMMONER_NAME_FIELD_NUMBER: _ClassVar[int]
    SUMMONER_LEVEL_FIELD_NUMBER: _ClassVar[int]
    REGION_FIELD_NUMBER: _ClassVar[int]
    LAST_UPDATED_FIELD_NUMBER: _ClassVar[int]
    puuid: str
    account_id: str
    summoner_id: str
    summoner_name: str
    summoner_level: int
    region: str
    last_updated: int
    def __init__(self, puuid: _Optional[str] = ..., account_id: _Optional[str] = ..., summoner_id: _Optional[str] = ..., summoner_name: _Optional[str] = ..., summoner_level: _Optional[int] = ..., region: _Optional[str] = ..., last_updated: _Optional[int] = ...) -> None: ...
