"""Prometheus metrics collector"""

from prometheus_client import (
    Counter,
    Histogram,
    Gauge,
    start_http_server,
    CollectorRegistry,
    REGISTRY,
)
from typing import Optional

from ..config import settings


class MetricsCollector:
    """Collector for Prometheus metrics"""

    def __init__(self):
        self.registry = REGISTRY

        # Business metrics
        self.events_processed_total = Counter(
            "dialogue_events_processed_total",
            "Total number of processed events",
            ["event_type", "status"],
            registry=self.registry,
        )

        self.dialogue_state_updates_total = Counter(
            "dialogue_state_updates_total",
            "Total number of dialogue state updates",
            ["chat_id"],
            registry=self.registry,
        )

        self.dialogue_active_count = Gauge(
            "dialogue_active_count",
            "Number of active dialogues",
            registry=self.registry,
        )

        self.dialogue_messages_total = Counter(
            "dialogue_messages_total",
            "Total number of messages in dialogue",
            ["chat_id"],
            registry=self.registry,
        )

        # Technical metrics
        self.event_processing_duration = Histogram(
            "event_processing_duration_seconds",
            "Event processing duration in seconds",
            ["event_type"],
            registry=self.registry,
        )

        self.redis_operation_duration = Histogram(
            "redis_operation_duration_seconds",
            "Redis operation duration in seconds",
            ["operation"],
            registry=self.registry,
        )

        self.kafka_producer_duration = Histogram(
            "kafka_producer_duration_seconds",
            "Kafka producer duration in seconds",
            registry=self.registry,
        )

        self.kafka_consumer_lag = Gauge(
            "kafka_consumer_lag",
            "Kafka consumer lag",
            ["topic", "partition"],
            registry=self.registry,
        )

        self.redis_connection_pool_size = Gauge(
            "redis_connection_pool_size",
            "Redis connection pool size",
            registry=self.registry,
        )

        self.error_count = Counter(
            "error_count",
            "Error count by type",
            ["error_type", "service"],
            registry=self.registry,
        )

    def start_metrics_server(self) -> None:
        """Start HTTP server for Prometheus metrics"""
        if settings.enable_prometheus:
            start_http_server(settings.metrics_port, registry=self.registry)


# Global metrics collector instance
_metrics_collector: Optional[MetricsCollector] = None


def get_metrics_collector() -> MetricsCollector:
    """Get or create metrics collector instance"""
    global _metrics_collector
    if _metrics_collector is None:
        _metrics_collector = MetricsCollector()
    return _metrics_collector

