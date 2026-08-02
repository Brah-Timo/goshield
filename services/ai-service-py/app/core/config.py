"""Central configuration loaded from environment variables."""
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_prefix="GOSHIELD_AI_",
        env_file=".env",
        env_file_encoding="utf-8",
        protected_namespaces=("settings_",),
    )

    # Service
    service_name: str = "ai-service-py"
    version: str = "1.0.0"
    environment: str = "development"
    port: int = 8090
    log_level: str = "INFO"

    # Model
    model_path: str = "/models/fraud_model.joblib"
    preprocessor_path: str = "/models/preprocessor.joblib"
    model_version: str = "v1"

    # Observability
    otlp_endpoint: str = "http://jaeger:4317"
    metrics_enabled: bool = True
    tracing_enabled: bool = True

    # CORS
    cors_origins: list[str] = ["*"]


settings = Settings()
