"""Kafka consumer for consuming events"""

import json
from typing import List, Callable, Optional
from confluent_kafka import Consumer, KafkaError, KafkaException, Message

from ..config import settings
from ..logger import get_logger
from ..models.events import KafkaEvent
from ..metrics import get_metrics_collector

logger = get_logger(__name__)


class KafkaConsumer:
    """Kafka consumer wrapper"""

    def __init__(self, topic: str, group_id: str):
        self.topic = topic
        self.group_id = group_id

        consumer_config = {
            "bootstrap.servers": settings.kafka_bootstrap_servers,
            "group.id": group_id,
            "auto.offset.reset": settings.kafka_consumer_auto_offset_reset,
            "enable.auto.commit": str(settings.kafka_consumer_enable_auto_commit).lower(),
            "session.timeout.ms": str(settings.kafka_consumer_session_timeout_ms),
            "heartbeat.interval.ms": str(settings.kafka_consumer_heartbeat_interval_ms),
        }

        self.consumer = Consumer(consumer_config)
        self.consumer.subscribe([topic])
        self.metrics = get_metrics_collector()
        logger.info(
            "Kafka consumer initialized",
            topic=topic,
            group_id=group_id,
            bootstrap_servers=settings.kafka_bootstrap_servers,
        )

    def poll(self, timeout: float = 1.0) -> List[KafkaEvent]:
        """Poll for messages from Kafka"""
        events = []
        try:
            msg = self.consumer.poll(timeout=timeout)
            if msg is None:
                return events

            if msg.error():
                if msg.error().code() == KafkaError._PARTITION_EOF:
                    logger.debug(
                        "Reached end of partition",
                        topic=msg.topic(),
                        partition=msg.partition(),
                    )
                else:
                    logger.error(
                        "Consumer error",
                        error=str(msg.error()),
                        topic=msg.topic(),
                    )
                    self.metrics.error_count.labels(
                        error_type="kafka_consumer_error",
                        service=settings.service_name,
                    ).inc()
                return events

            try:
                event_dict = json.loads(msg.value().decode("utf-8"))
                event = KafkaEvent(**event_dict)
                events.append(event)

                logger.debug(
                    "Event consumed",
                    event_id=event.event_id,
                    event_type=event.event_type,
                    topic=msg.topic(),
                    partition=msg.partition(),
                    offset=msg.offset(),
                )

                # Update lag metrics
                metadata = msg.headers()
                # Note: actual lag requires additional metadata from broker
                # This is a simplified version

            except Exception as e:
                logger.error(
                    "Failed to parse event",
                    error=str(e),
                    topic=msg.topic(),
                    partition=msg.partition(),
                    offset=msg.offset(),
                )
                self.metrics.error_count.labels(
                    error_type="kafka_consumer_parse_error",
                    service=settings.service_name,
                ).inc()

        except KafkaException as e:
            logger.error("Kafka consumer exception", error=str(e))
            self.metrics.error_count.labels(
                error_type="kafka_consumer_exception", service=settings.service_name
            ).inc()

        return events

    def commit(self) -> None:
        """Commit current offsets"""
        try:
            self.consumer.commit(asynchronous=False)
        except Exception as e:
            logger.error("Failed to commit offsets", error=str(e))
            self.metrics.error_count.labels(
                error_type="kafka_consumer_commit_error",
                service=settings.service_name,
            ).inc()
            raise

    def close(self) -> None:
        """Close consumer"""
        self.consumer.close()
        logger.info("Kafka consumer closed", topic=self.topic, group_id=self.group_id)


def create_kafka_consumer(topic: str, group_id: str) -> KafkaConsumer:
    """Create and return Kafka consumer instance"""
    return KafkaConsumer(topic, group_id)

