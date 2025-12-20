"""Redis client for dialogue state management"""

import json
import redis
from typing import Optional, List, Set
from datetime import datetime

from ..config import settings
from ..logger import get_logger
from ..models.dialogue import DialogueState
from ..models.messages import Message
from ..metrics import get_metrics_collector

logger = get_logger(__name__)


class RedisClient:
    """Redis client wrapper with dialogue-specific operations"""

    def __init__(self):
        self.client = redis.Redis(
            host=settings.redis_host,
            port=settings.redis_port,
            db=settings.redis_db,
            password=settings.redis_password,
            max_connections=settings.redis_max_connections,
            socket_connect_timeout=settings.redis_connection_timeout,
            socket_timeout=settings.redis_socket_timeout,
            retry_on_timeout=settings.redis_retry_on_timeout,
            decode_responses=True,
        )
        self.metrics = get_metrics_collector()
        logger.info(
            "Redis client initialized",
            host=settings.redis_host,
            port=settings.redis_port,
            db=settings.redis_db,
        )

    def _get_state_key(self, chat_id: str) -> str:
        """Get Redis key for dialogue state"""
        return f"dialogue:{chat_id}:state"

    def _get_messages_key(self, chat_id: str) -> str:
        """Get Redis key for messages list"""
        return f"dialogue:{chat_id}:messages"

    def _get_active_index_key(self) -> str:
        """Get Redis key for active dialogues index"""
        return "dialogue:active:index"

    def get_dialogue_state(self, chat_id: str) -> Optional[DialogueState]:
        """Get dialogue state from Redis"""
        with self.metrics.redis_operation_duration.labels(operation="get_state").time():
            try:
                key = self._get_state_key(chat_id)
                data = self.client.get(key)
                if data is None:
                    return None

                state_dict = json.loads(data)
                return DialogueState(**state_dict)
            except Exception as e:
                logger.error(
                    "Failed to get dialogue state",
                    chat_id=chat_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_get_state", service=settings.service_name
                ).inc()
                raise

    def save_dialogue_state(
        self, chat_id: str, state: DialogueState, ttl: Optional[int] = None
    ) -> None:
        """Save dialogue state to Redis"""
        with self.metrics.redis_operation_duration.labels(
            operation="save_state"
        ).time():
            try:
                key = self._get_state_key(chat_id)
                ttl = ttl or settings.redis_dialogue_ttl_seconds
                data = state.model_dump_json()
                self.client.setex(key, ttl, data)

                self.metrics.dialogue_state_updates_total.labels(
                    chat_id=chat_id
                ).inc()
            except Exception as e:
                logger.error(
                    "Failed to save dialogue state",
                    chat_id=chat_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_save_state", service=settings.service_name
                ).inc()
                raise

    def add_message(self, chat_id: str, message: Message) -> None:
        """Add message to dialogue messages list"""
        with self.metrics.redis_operation_duration.labels(
            operation="add_message"
        ).time():
            try:
                key = self._get_messages_key(chat_id)
                message_data = message.model_dump_json()

                # Add message to list (right push)
                self.client.rpush(key, message_data)

                # Trim to keep only last N messages
                max_size = settings.redis_messages_cache_size
                self.client.ltrim(key, -max_size, -1)

                # Set TTL
                self.client.expire(key, settings.redis_dialogue_ttl_seconds)

                self.metrics.dialogue_messages_total.labels(chat_id=chat_id).inc()
            except Exception as e:
                logger.error(
                    "Failed to add message",
                    chat_id=chat_id,
                    message_id=message.message_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_add_message", service=settings.service_name
                ).inc()
                raise

    def get_messages(
        self, chat_id: str, limit: Optional[int] = None
    ) -> List[Message]:
        """Get last N messages from dialogue"""
        with self.metrics.redis_operation_duration.labels(
            operation="get_messages"
        ).time():
            try:
                key = self._get_messages_key(chat_id)
                limit = limit or settings.redis_messages_cache_size

                # Get last N messages
                messages_data = self.client.lrange(key, -limit, -1)
                messages = []
                for msg_data in messages_data:
                    msg_dict = json.loads(msg_data)
                    messages.append(Message(**msg_dict))

                return messages
            except Exception as e:
                logger.error(
                    "Failed to get messages",
                    chat_id=chat_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_get_messages", service=settings.service_name
                ).inc()
                raise

    def add_to_active_index(self, chat_id: str) -> None:
        """Add dialogue to active index"""
        with self.metrics.redis_operation_duration.labels(
            operation="add_to_active_index"
        ).time():
            try:
                key = self._get_active_index_key()
                self.client.sadd(key, chat_id)
            except Exception as e:
                logger.error(
                    "Failed to add to active index",
                    chat_id=chat_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_add_active_index",
                    service=settings.service_name,
                ).inc()
                raise

    def remove_from_active_index(self, chat_id: str) -> None:
        """Remove dialogue from active index"""
        with self.metrics.redis_operation_duration.labels(
            operation="remove_from_active_index"
        ).time():
            try:
                key = self._get_active_index_key()
                self.client.srem(key, chat_id)
            except Exception as e:
                logger.error(
                    "Failed to remove from active index",
                    chat_id=chat_id,
                    error=str(e),
                )
                self.metrics.error_count.labels(
                    error_type="redis_remove_active_index",
                    service=settings.service_name,
                ).inc()
                raise

    def get_active_dialogues(self) -> Set[str]:
        """Get all active dialogue chat_ids"""
        with self.metrics.redis_operation_duration.labels(
            operation="get_active_dialogues"
        ).time():
            try:
                key = self._get_active_index_key()
                chat_ids = self.client.smembers(key)
                return set(chat_ids) if chat_ids else set()
            except Exception as e:
                logger.error("Failed to get active dialogues", error=str(e))
                self.metrics.error_count.labels(
                    error_type="redis_get_active_dialogues",
                    service=settings.service_name,
                ).inc()
                raise

    def ping(self) -> bool:
        """Ping Redis server"""
        try:
            return self.client.ping()
        except Exception as e:
            logger.error("Redis ping failed", error=str(e))
            return False

    def close(self) -> None:
        """Close Redis connection"""
        self.client.close()


def create_redis_client() -> RedisClient:
    """Create and return Redis client instance"""
    return RedisClient()

