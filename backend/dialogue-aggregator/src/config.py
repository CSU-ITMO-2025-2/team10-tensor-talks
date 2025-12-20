"""Configuration management using Pydantic Settings"""

from pydantic_settings import BaseSettings
from pydantic import Field
from typing import Optional


class Settings(BaseSettings):
    """Application settings loaded from environment variables"""

    # Service
    service_name: str = Field(default="dialogue-aggregator", alias="SERVICE_NAME")
    service_version: str = Field(default="1.0.0", alias="SERVICE_VERSION")
    log_level: str = Field(default="INFO", alias="LOG_LEVEL")
    log_format: str = Field(default="json", alias="LOG_FORMAT")

    # Kafka
    kafka_bootstrap_servers: str = Field(
        default="localhost:9092", alias="KAFKA_BOOTSTRAP_SERVERS"
    )
    kafka_consumer_group_publication: str = Field(
        default="dialogue-aggregator-publication-group",
        alias="KAFKA_CONSUMER_GROUP_PUBLICATION",
    )
    kafka_consumer_group_messages: str = Field(
        default="dialogue-aggregator-messages-group",
        alias="KAFKA_CONSUMER_GROUP_MESSAGES",
    )
    kafka_consumer_group_generated: str = Field(
        default="dialogue-aggregator-generated-group",
        alias="KAFKA_CONSUMER_GROUP_GENERATED",
    )
    kafka_topic_publication: str = Field(
        default="publication.events", alias="KAFKA_TOPIC_PUBLICATION"
    )
    kafka_topic_messages: str = Field(
        default="messages.events", alias="KAFKA_TOPIC_MESSAGES"
    )
    kafka_topic_generated: str = Field(
        default="generated.phrases", alias="KAFKA_TOPIC_GENERATED"
    )
    kafka_topic_full_data: str = Field(
        default="messages.full.data", alias="KAFKA_TOPIC_FULL_DATA"
    )
    kafka_topic_history: str = Field(
        default="history.full.events", alias="KAFKA_TOPIC_HISTORY"
    )
    kafka_topic_dlq: str = Field(
        default="dialogue-aggregator.dlq", alias="KAFKA_TOPIC_DLQ"
    )
    kafka_consumer_auto_offset_reset: str = Field(
        default="earliest", alias="KAFKA_CONSUMER_AUTO_OFFSET_RESET"
    )
    kafka_consumer_enable_auto_commit: bool = Field(
        default=False, alias="KAFKA_CONSUMER_ENABLE_AUTO_COMMIT"
    )
    kafka_consumer_max_poll_records: int = Field(
        default=100, alias="KAFKA_CONSUMER_MAX_POLL_RECORDS"
    )
    kafka_consumer_session_timeout_ms: int = Field(
        default=30000, alias="KAFKA_CONSUMER_SESSION_TIMEOUT_MS"
    )
    kafka_consumer_heartbeat_interval_ms: int = Field(
        default=10000, alias="KAFKA_CONSUMER_HEARTBEAT_INTERVAL_MS"
    )
    kafka_producer_acks: str = Field(default="all", alias="KAFKA_PRODUCER_ACKS")
    kafka_producer_retries: int = Field(default=3, alias="KAFKA_PRODUCER_RETRIES")
    kafka_producer_compression_type: str = Field(
        default="snappy", alias="KAFKA_PRODUCER_COMPRESSION_TYPE"
    )

    # Redis
    redis_host: str = Field(default="localhost", alias="REDIS_HOST")
    redis_port: int = Field(default=6379, alias="REDIS_PORT")
    redis_db: int = Field(default=0, alias="REDIS_DB")
    redis_password: Optional[str] = Field(default=None, alias="REDIS_PASSWORD")
    redis_max_connections: int = Field(
        default=50, alias="REDIS_MAX_CONNECTIONS"
    )
    redis_dialogue_ttl_seconds: int = Field(
        default=86400, alias="REDIS_DIALOGUE_TTL_SECONDS"
    )
    redis_messages_cache_size: int = Field(
        default=50, alias="REDIS_MESSAGES_CACHE_SIZE"
    )
    redis_connection_timeout: int = Field(
        default=5, alias="REDIS_CONNECTION_TIMEOUT"
    )
    redis_socket_timeout: int = Field(default=5, alias="REDIS_SOCKET_TIMEOUT")
    redis_retry_on_timeout: bool = Field(
        default=True, alias="REDIS_RETRY_ON_TIMEOUT"
    )

    # Processing
    max_messages_cache_size: int = Field(
        default=50, alias="MAX_MESSAGES_CACHE_SIZE"
    )
    state_update_batch_size: int = Field(
        default=10, alias="STATE_UPDATE_BATCH_SIZE"
    )
    event_processing_timeout_seconds: int = Field(
        default=30, alias="EVENT_PROCESSING_TIMEOUT_SECONDS"
    )

    # Metrics
    metrics_port: int = Field(default=9091, alias="METRICS_PORT")
    enable_prometheus: bool = Field(
        default=True, alias="ENABLE_PROMETHEUS"
    )

    class Config:
        env_file = ".env"
        case_sensitive = False


# Global settings instance
settings = Settings()

