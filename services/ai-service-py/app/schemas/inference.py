"""Pydantic schemas for inference request and response."""
from __future__ import annotations
from pydantic import BaseModel, Field, field_validator
from typing import Optional


class InferenceRequest(BaseModel):
    """Input features for fraud scoring."""
    claim_id: str = Field(..., description="UUID of the claim")
    amount: float = Field(..., gt=0, description="Claim amount in USD")
    claim_type: str = Field(..., description="HEALTH|CAR|PROPERTY|LIFE|TRAVEL|OTHER")
    description: str = Field(default="", max_length=2000)
    policy_number: str = Field(..., min_length=3, max_length=50)
    user_id: str = Field(..., description="UUID of the claimant")
    company_id: str = Field(..., description="UUID of the tenant company")
    account_age_days: int = Field(default=0, ge=0, description="Days since account creation")
    prior_claims_count: int = Field(default=0, ge=0, description="Number of prior claims")
    incident_date: Optional[str] = Field(default=None, description="ISO date YYYY-MM-DD")
    doc_url: Optional[str] = Field(default=None, description="URL of the uploaded document")

    @field_validator("claim_type")
    @classmethod
    def validate_claim_type(cls, v: str) -> str:
        valid = {"HEALTH", "CAR", "PROPERTY", "LIFE", "TRAVEL", "OTHER"}
        if v.upper() not in valid:
            raise ValueError(f"claim_type must be one of {valid}")
        return v.upper()


class RiskFactor(BaseModel):
    feature: str
    importance: float
    direction: str   # "increases_risk" | "decreases_risk"
    value: float


class InferenceResponse(BaseModel):
    """Full fraud analysis result with SHAP explanation."""
    claim_id: str
    fraud_score: float = Field(..., ge=0.0, le=1.0)
    risk_level: str              # LOW | MEDIUM | HIGH | CRITICAL
    reason: str                  # Human-readable summary
    risk_factors: list[str]      # Top N factor names
    shap_values: list[RiskFactor]
    model_version: str
    confidence: float = Field(..., ge=0.0, le=1.0)
    processing_time_ms: float


class HealthResponse(BaseModel):
    status: str
    service: str
    version: str
    model_loaded: bool
    model_version: str
