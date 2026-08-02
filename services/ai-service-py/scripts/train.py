#!/usr/bin/env python3
"""
Train the GoShield fraud detection XGBoost model.

Usage:
    python scripts/train.py --data /path/to/claims.csv --output /models/

The CSV must have columns:
    amount, claim_type, account_age_days, prior_claims_count,
    incident_date, policy_number, label (0=legit, 1=fraud)

If no --data is provided, synthetic data is generated for bootstrapping.
"""
import argparse
import sys
from pathlib import Path

import numpy as np
import pandas as pd
import joblib
import shap
from xgboost import XGBClassifier
from sklearn.model_selection import train_test_split, StratifiedKFold, cross_val_score
from sklearn.metrics import (
    classification_report, roc_auc_score,
    precision_recall_curve, average_precision_score,
)
from sklearn.preprocessing import StandardScaler
from imblearn.over_sampling import SMOTE
import warnings
warnings.filterwarnings("ignore")

# Keep in sync with fraud_model.py
FEATURE_COLUMNS = [
    "amount_log", "account_age_days", "prior_claims_count",
    "claim_type_enc", "amount_per_day", "is_new_account",
    "high_amount_flag", "repeat_claimant_flag", "days_since_incident",
    "policy_hash_mod",
]

CLAIM_TYPE_MAP = {
    "HEALTH": 0, "CAR": 1, "PROPERTY": 2,
    "LIFE": 3, "TRAVEL": 4, "OTHER": 5,
}


def engineer_features(df: pd.DataFrame) -> pd.DataFrame:
    import math, hashlib, datetime
    out = pd.DataFrame()
    out["amount_log"] = df["amount"].apply(lambda x: math.log1p(max(x, 0)))
    out["account_age_days"] = df.get("account_age_days", pd.Series([365] * len(df)))
    out["prior_claims_count"] = df.get("prior_claims_count", pd.Series([0] * len(df)))
    out["claim_type_enc"] = df.get("claim_type", "OTHER").map(
        lambda x: CLAIM_TYPE_MAP.get(str(x).upper(), 5)
    )
    days = out["account_age_days"].clip(lower=1)
    out["amount_per_day"] = df["amount"] / days
    out["is_new_account"] = (out["account_age_days"] < 90).astype(int)
    out["high_amount_flag"] = (df["amount"] > 10_000).astype(int)
    out["repeat_claimant_flag"] = (out["prior_claims_count"] >= 3).astype(int)

    def days_since(d):
        try:
            delta = (datetime.date.today() - datetime.date.fromisoformat(str(d))).days
            return max(delta, 0)
        except Exception:
            return 0

    if "incident_date" in df.columns:
        out["days_since_incident"] = df["incident_date"].apply(days_since)
    else:
        out["days_since_incident"] = 30

    import hashlib
    if "policy_number" in df.columns:
        out["policy_hash_mod"] = df["policy_number"].apply(
            lambda x: int(hashlib.md5(str(x).encode()).hexdigest(), 16) % 1000
        )
    else:
        out["policy_hash_mod"] = 0

    return out[FEATURE_COLUMNS]


def generate_synthetic_data(n_legit: int = 8000, n_fraud: int = 800) -> pd.DataFrame:
    np.random.seed(42)
    legit = pd.DataFrame({
        "amount": np.random.lognormal(7, 1.5, n_legit),
        "claim_type": np.random.choice(list(CLAIM_TYPE_MAP.keys()), n_legit),
        "account_age_days": np.random.randint(180, 3650, n_legit),
        "prior_claims_count": np.random.poisson(0.5, n_legit),
        "incident_date": pd.date_range("2023-01-01", periods=n_legit, freq="h").date,
        "policy_number": [f"POL{i:06d}" for i in range(n_legit)],
        "label": 0,
    })
    fraud = pd.DataFrame({
        "amount": np.random.lognormal(10, 1.0, n_fraud),
        "claim_type": np.random.choice(list(CLAIM_TYPE_MAP.keys()), n_fraud),
        "account_age_days": np.random.randint(1, 90, n_fraud),
        "prior_claims_count": np.random.randint(3, 10, n_fraud),
        "incident_date": pd.date_range("2024-01-01", periods=n_fraud, freq="h").date,
        "policy_number": [f"FRD{i:06d}" for i in range(n_fraud)],
        "label": 1,
    })
    return pd.concat([legit, fraud], ignore_index=True).sample(frac=1, random_state=42)


def train(data_path: str | None, output_dir: str) -> None:
    out = Path(output_dir)
    out.mkdir(parents=True, exist_ok=True)

    print("── Loading data ─────────────────────────────────────────")
    if data_path:
        df = pd.read_csv(data_path)
        print(f"   Loaded {len(df)} rows from {data_path}")
    else:
        print("   No data path — generating synthetic training data")
        df = generate_synthetic_data()
        print(f"   Generated {len(df)} synthetic samples")

    print(f"   Fraud rate: {df['label'].mean():.2%}")

    # Feature engineering
    X = engineer_features(df)
    y = df["label"].values

    # Train/test split
    X_train, X_test, y_train, y_test = train_test_split(
        X, y, test_size=0.2, stratify=y, random_state=42
    )

    # Scale
    scaler = StandardScaler()
    X_train_s = scaler.fit_transform(X_train)
    X_test_s = scaler.transform(X_test)

    # SMOTE oversampling for imbalanced dataset
    smote = SMOTE(random_state=42)
    X_res, y_res = smote.fit_resample(X_train_s, y_train)
    print(f"   After SMOTE: {len(X_res)} samples, fraud rate: {y_res.mean():.2%}")

    print("\n── Training XGBoost model ────────────────────────────────")
    model = XGBClassifier(
        n_estimators=300,
        max_depth=6,
        learning_rate=0.05,
        subsample=0.8,
        colsample_bytree=0.8,
        min_child_weight=5,
        gamma=0.1,
        reg_alpha=0.1,
        reg_lambda=1.0,
        use_label_encoder=False,
        eval_metric="aucpr",
        random_state=42,
        n_jobs=-1,
    )
    model.fit(
        X_res, y_res,
        eval_set=[(X_test_s, y_test)],
        verbose=50,
    )

    # Evaluation
    print("\n── Evaluation ────────────────────────────────────────────")
    y_pred = model.predict(X_test_s)
    y_proba = model.predict_proba(X_test_s)[:, 1]

    print(classification_report(y_test, y_pred, target_names=["Legit", "Fraud"]))
    roc = roc_auc_score(y_test, y_proba)
    pr = average_precision_score(y_test, y_proba)
    print(f"   ROC-AUC:  {roc:.4f}")
    print(f"   PR-AUC:   {pr:.4f}")

    # Cross-validation
    cv = StratifiedKFold(n_splits=5, shuffle=True, random_state=42)
    cv_scores = cross_val_score(model, scaler.transform(X), y, cv=cv, scoring="roc_auc")
    print(f"   CV ROC-AUC: {cv_scores.mean():.4f} ± {cv_scores.std():.4f}")

    # SHAP feature importance
    print("\n── SHAP feature importance ──────────────────────────────")
    explainer = shap.TreeExplainer(model)
    shap_vals = explainer.shap_values(X_test_s[:500])
    mean_abs = np.abs(shap_vals).mean(axis=0)
    for col, imp in sorted(zip(FEATURE_COLUMNS, mean_abs), key=lambda x: -x[1]):
        print(f"   {col:<30} {imp:.4f}")

    # Save model and preprocessor
    model_out = out / "fraud_model.joblib"
    preprocessor_out = out / "preprocessor.joblib"
    joblib.dump(model, str(model_out))
    joblib.dump(scaler, str(preprocessor_out))
    print(f"\n✓ Model saved:        {model_out}")
    print(f"✓ Preprocessor saved: {preprocessor_out}")
    print(f"✓ ROC-AUC: {roc:.4f}  |  PR-AUC: {pr:.4f}")


def main():
    parser = argparse.ArgumentParser(description="Train GoShield fraud detection model")
    parser.add_argument("--data", type=str, default=None, help="Path to CSV training data")
    parser.add_argument("--output", type=str, default="/models", help="Output directory for model files")
    args = parser.parse_args()
    train(args.data, args.output)


if __name__ == "__main__":
    main()
