"""
FraudModel: XGBoost-based fraud detection model with SHAP explainability.

Feature engineering + inference + SHAP explanation pipeline.
"""
from __future__ import annotations

import time
import math
import hashlib
from pathlib import Path
from typing import Optional
import numpy as np
import pandas as pd
import joblib
import shap
from xgboost import XGBClassifier
from sklearn.preprocessing import StandardScaler, LabelEncoder
from sklearn.pipeline import Pipeline

from app.core.logging import logger
from app.schemas.inference import InferenceRequest, InferenceResponse, RiskFactor


# Feature columns expected by the model (order matters)
FEATURE_COLUMNS = [
    "amount_log",
    "account_age_days",
    "prior_claims_count",
    "claim_type_enc",
    "amount_per_day",
    "is_new_account",
    "high_amount_flag",
    "repeat_claimant_flag",
    "days_since_incident",
    "policy_hash_mod",
]

CLAIM_TYPE_MAP = {
    "HEALTH": 0, "CAR": 1, "PROPERTY": 2,
    "LIFE": 3, "TRAVEL": 4, "OTHER": 5
}

FEATURE_DESCRIPTIONS = {
    "amount_log": "Claim amount magnitude",
    "account_age_days": "Account age",
    "prior_claims_count": "Number of prior claims",
    "claim_type_enc": "Insurance category",
    "amount_per_day": "Daily claim rate",
    "is_new_account": "New account indicator",
    "high_amount_flag": "High-value claim flag",
    "repeat_claimant_flag": "Repeat claimant pattern",
    "days_since_incident": "Incident recency",
    "policy_hash_mod": "Policy number pattern",
}


def _engineer_features(req: InferenceRequest) -> pd.DataFrame:
    """Transform raw claim request into model feature vector."""
    amount_log = math.log1p(req.amount)
    days = max(req.account_age_days, 1)
    amount_per_day = req.amount / days
    is_new_account = 1 if req.account_age_days < 90 else 0
    high_amount_flag = 1 if req.amount > 10_000 else 0
    repeat_claimant_flag = 1 if req.prior_claims_count >= 3 else 0
    claim_type_enc = CLAIM_TYPE_MAP.get(req.claim_type.upper(), 5)

    days_since_incident = 0
    if req.incident_date:
        try:
            import datetime
            incident = datetime.date.fromisoformat(req.incident_date)
            delta = (datetime.date.today() - incident).days
            days_since_incident = max(delta, 0)
        except ValueError:
            days_since_incident = 0

    # Deterministic hash of policy number pattern (detect suspicious serials)
    policy_hash_mod = int(hashlib.md5(req.policy_number.encode()).hexdigest(), 16) % 1000

    return pd.DataFrame([{
        "amount_log": amount_log,
        "account_age_days": req.account_age_days,
        "prior_claims_count": req.prior_claims_count,
        "claim_type_enc": claim_type_enc,
        "amount_per_day": amount_per_day,
        "is_new_account": is_new_account,
        "high_amount_flag": high_amount_flag,
        "repeat_claimant_flag": repeat_claimant_flag,
        "days_since_incident": days_since_incident,
        "policy_hash_mod": policy_hash_mod,
    }], columns=FEATURE_COLUMNS)


def _risk_level(score: float) -> str:
    if score >= 0.90:
        return "CRITICAL"
    if score >= 0.75:
        return "HIGH"
    if score >= 0.50:
        return "MEDIUM"
    return "LOW"


def _build_reason(shap_vals: list[RiskFactor], score: float) -> str:
    """Generate a human-readable fraud reason from SHAP values."""
    top = [f for f in shap_vals if f.direction == "increases_risk"][:3]
    if not top:
        return "Claim pattern appears normal based on historical data."
    parts = []
    for f in top:
        desc = FEATURE_DESCRIPTIONS.get(f.feature, f.feature)
        parts.append(desc)
    level = _risk_level(score)
    return f"{level} fraud risk. Primary indicators: {', '.join(parts)}."


class FraudModel:
    """Wraps the trained XGBoost model with SHAP TreeExplainer."""

    def __init__(self, model_path: str, preprocessor_path: Optional[str] = None):
        self.model_path = Path(model_path)
        self.preprocessor_path = Path(preprocessor_path) if preprocessor_path else None
        self._model: Optional[XGBClassifier] = None
        self._scaler: Optional[StandardScaler] = None
        self._explainer: Optional[shap.TreeExplainer] = None
        self._version = "v1"
        self._loaded = False

    def load(self) -> None:
        """Load model from disk. Falls back to a synthetic model if file not found."""
        if self.model_path.exists():
            self._model = joblib.load(str(self.model_path))
            logger.info("Model loaded from disk", path=str(self.model_path))
        else:
            logger.warning(
                "Model file not found — using synthetic model",
                path=str(self.model_path),
            )
            self._model = self._build_synthetic_model()

        if self.preprocessor_path and self.preprocessor_path.exists():
            self._scaler = joblib.load(str(self.preprocessor_path))

        self._explainer = shap.TreeExplainer(self._model)
        self._loaded = True
        logger.info("FraudModel ready", version=self._version)

    def is_loaded(self) -> bool:
        return self._loaded

    def predict(self, req: InferenceRequest) -> InferenceResponse:
        """Run inference and return fraud score with SHAP explanation."""
        if not self._loaded:
            raise RuntimeError("Model not loaded")

        t0 = time.monotonic()
        features = _engineer_features(req)
        X = features.values.astype(float)

        # Scale if preprocessor available
        if self._scaler is not None:
            X = self._scaler.transform(X)

        # Fraud probability
        proba = self._model.predict_proba(X)[0]
        fraud_score = float(proba[1])
        confidence = float(max(proba))

        # SHAP values for explainability
        shap_values = self._explainer.shap_values(X)
        # For binary XGB, shap_values is shape (1, n_features)
        sv = shap_values[0] if hasattr(shap_values, "__len__") else shap_values

        risk_factors_raw = []
        for i, col in enumerate(FEATURE_COLUMNS):
            sv_val = float(sv[i]) if hasattr(sv, "__getitem__") else 0.0
            risk_factors_raw.append(RiskFactor(
                feature=col,
                importance=abs(sv_val),
                direction="increases_risk" if sv_val > 0 else "decreases_risk",
                value=float(features[col].iloc[0]),
            ))

        # Sort by absolute SHAP importance desc
        risk_factors_raw.sort(key=lambda x: x.importance, reverse=True)
        top_risk_names = [
            f.feature for f in risk_factors_raw
            if f.direction == "increases_risk"
        ][:5]

        reason = _build_reason(risk_factors_raw, fraud_score)
        elapsed_ms = (time.monotonic() - t0) * 1000

        return InferenceResponse(
            claim_id=req.claim_id,
            fraud_score=round(fraud_score, 4),
            risk_level=_risk_level(fraud_score),
            reason=reason,
            risk_factors=top_risk_names,
            shap_values=risk_factors_raw[:10],
            model_version=self._version,
            confidence=round(confidence, 4),
            processing_time_ms=round(elapsed_ms, 2),
        )

    # ── private ────────────────────────────────────────────────────────────────

    @staticmethod
    def _build_synthetic_model() -> XGBClassifier:
        """
        Build and train a synthetic XGBoost model on programmatically generated data.
        This ensures the service works out-of-the-box before real training data is available.
        """
        np.random.seed(42)
        n = 5000

        # Simulate legitimate claims
        legit = pd.DataFrame({
            "amount_log": np.random.normal(6, 1.5, n),
            "account_age_days": np.random.randint(180, 3650, n),
            "prior_claims_count": np.random.poisson(0.5, n),
            "claim_type_enc": np.random.randint(0, 6, n),
            "amount_per_day": np.random.uniform(0.1, 50, n),
            "is_new_account": np.zeros(n, dtype=int),
            "high_amount_flag": np.random.randint(0, 2, n),
            "repeat_claimant_flag": np.zeros(n, dtype=int),
            "days_since_incident": np.random.randint(1, 365, n),
            "policy_hash_mod": np.random.randint(0, 1000, n),
        })
        legit["label"] = 0

        # Simulate fraudulent claims (fewer, with distinct patterns)
        n_fraud = 500
        fraud = pd.DataFrame({
            "amount_log": np.random.normal(9, 1.0, n_fraud),
            "account_age_days": np.random.randint(1, 90, n_fraud),
            "prior_claims_count": np.random.randint(3, 10, n_fraud),
            "claim_type_enc": np.random.randint(0, 6, n_fraud),
            "amount_per_day": np.random.uniform(200, 2000, n_fraud),
            "is_new_account": np.ones(n_fraud, dtype=int),
            "high_amount_flag": np.ones(n_fraud, dtype=int),
            "repeat_claimant_flag": np.ones(n_fraud, dtype=int),
            "days_since_incident": np.random.randint(0, 7, n_fraud),
            "policy_hash_mod": np.random.randint(0, 1000, n_fraud),
        })
        fraud["label"] = 1

        df = pd.concat([legit, fraud], ignore_index=True).sample(frac=1, random_state=42)
        X = df[FEATURE_COLUMNS].values
        y = df["label"].values

        model = XGBClassifier(
            n_estimators=200,
            max_depth=6,
            learning_rate=0.05,
            subsample=0.8,
            colsample_bytree=0.8,
            scale_pos_weight=n / n_fraud,
            use_label_encoder=False,
            eval_metric="logloss",
            random_state=42,
        )
        model.fit(X, y, verbose=False)
        logger.info("Synthetic XGBoost model trained", n_samples=len(df))
        return model
