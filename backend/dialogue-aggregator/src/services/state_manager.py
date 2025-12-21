"""Dialogue state management"""

from datetime import datetime
from typing import Optional, Dict, Any

from ..models.dialogue import DialogueState, DialogueStatus, DialogueFlags
from ..models.messages import Message
from ..redis_client import RedisClient
from ..logger import get_logger
from ..metrics import get_metrics_collector

logger = get_logger(__name__)


class StateManager:
    """Manages dialogue state lifecycle"""

    def __init__(self, redis_client: RedisClient):
        self.redis = redis_client
        self.metrics = get_metrics_collector()

    def initialize_dialogue(
        self,
        chat_id: str,
        user_id: str,
        session_id: str,
        dialogue_type: str,
        started_at: datetime,
    ) -> DialogueState:
        """Initialize new dialogue state"""
        logger.info(
            "Initializing dialogue",
            chat_id=chat_id,
            user_id=user_id,
            dialogue_type=dialogue_type,
        )

        # Check if dialogue already exists
        existing_state = self.get_dialogue_state(chat_id)
        if existing_state:
            logger.warning(
                "Dialogue already exists, returning existing state",
                chat_id=chat_id,
            )
            return existing_state

        # Create new state
        state = DialogueState(
            chat_id=chat_id,
            user_id=user_id,
            session_id=session_id,
            dialogue_type=dialogue_type,
            status=DialogueStatus.ACTIVE,
            started_at=started_at,
            last_activity=started_at,
            messages_count=0,
            state_version=0,
            flags=DialogueFlags(),
            metadata={},
        )

        # Save to Redis
        self.redis.save_dialogue_state(chat_id, state)

        # Add to active index
        self.redis.add_to_active_index(chat_id)

        # Update metrics
        active_count = len(self.redis.get_active_dialogues())
        self.metrics.dialogue_active_count.set(active_count)

        logger.info("Dialogue initialized", chat_id=chat_id)
        return state

    def get_dialogue_state(self, chat_id: str) -> Optional[DialogueState]:
        """Get dialogue state"""
        return self.redis.get_dialogue_state(chat_id)

    def update_dialogue_state(
        self, chat_id: str, updates: Dict[str, Any]
    ) -> DialogueState:
        """Update dialogue state with optimistic locking"""
        state = self.get_dialogue_state(chat_id)
        if not state:
            raise ValueError(f"Dialogue not found: {chat_id}")

        # Store original version for optimistic locking
        original_version = state.state_version

        # Apply updates
        for key, value in updates.items():
            if hasattr(state, key):
                setattr(state, key, value)

        # Increment version
        state.state_version += 1
        state.last_activity = datetime.utcnow()

        # Save to Redis
        self.redis.save_dialogue_state(chat_id, state)

        logger.debug(
            "Dialogue state updated",
            chat_id=chat_id,
            version=state.state_version,
            updates=list(updates.keys()),
        )

        return state

    def add_message_to_dialogue(self, chat_id: str, message: Message) -> None:
        """Add message to dialogue"""
        # Add to Redis messages list
        self.redis.add_message(chat_id, message)

        # Update state
        state = self.get_dialogue_state(chat_id)
        if state:
            self.update_dialogue_state(
                chat_id,
                {
                    "messages_count": state.messages_count + 1,
                },
            )

    def finish_dialogue(self, chat_id: str) -> None:
        """Finish dialogue"""
        state = self.get_dialogue_state(chat_id)
        if not state:
            logger.warning("Dialogue not found for finishing", chat_id=chat_id)
            return

        self.update_dialogue_state(
            chat_id,
            {
                "status": DialogueStatus.FINISHED,
                "flags": DialogueFlags(is_finished=True),
            },
        )

        # Remove from active index
        self.redis.remove_from_active_index(chat_id)

        # Update metrics
        active_count = len(self.redis.get_active_dialogues())
        self.metrics.dialogue_active_count.set(active_count)

        logger.info("Dialogue finished", chat_id=chat_id)

