"""Event routing service"""

from typing import Dict, Callable, Optional

from ..models.events import KafkaEvent, EventType
from ..logger import get_logger
from ..metrics import get_metrics_collector

logger = get_logger(__name__)


class EventRouter:
    """Routes events to appropriate handlers"""

    def __init__(self):
        self.handlers: Dict[EventType, Callable[[KafkaEvent], None]] = {}
        self.metrics = get_metrics_collector()

    def register_handler(
        self, event_type: EventType, handler: Callable[[KafkaEvent], None]
    ) -> None:
        """Register handler for event type"""
        self.handlers[event_type] = handler
        logger.info("Handler registered", event_type=event_type.value)

    def route_event(self, event: KafkaEvent) -> None:
        """Route event to appropriate handler"""
        logger.debug(
            "Routing event",
            event_id=event.event_id,
            event_type=event.event_type,
        )

        try:
            # Get handler
            handler = self.handlers.get(event.event_type)

            if handler is None:
                logger.warning(
                    "No handler registered for event type",
                    event_id=event.event_id,
                    event_type=event.event_type,
                )
                self.metrics.events_processed_total.labels(
                    event_type=event.event_type.value, status="no_handler"
                ).inc()
                return

            # Execute handler
            handler(event)

            self.metrics.events_processed_total.labels(
                event_type=event.event_type.value, status="success"
            ).inc()

            logger.info(
                "Event processed successfully",
                event_id=event.event_id,
                event_type=event.event_type,
            )

        except Exception as e:
            logger.error(
                "Failed to process event",
                event_id=event.event_id,
                event_type=event.event_type,
                error=str(e),
                exc_info=True,
            )
            self.metrics.events_processed_total.labels(
                event_type=event.event_type.value, status="error"
            ).inc()
            self.metrics.error_count.labels(
                error_type="event_processing_error",
                service="dialogue-aggregator",
            ).inc()
            raise

