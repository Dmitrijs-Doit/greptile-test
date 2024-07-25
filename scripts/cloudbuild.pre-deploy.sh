#!/usr/bin/env bash

if [[ "$PROJECT_ID" == "me-doit-intl-com" ]]; then
  export SENTRY_PROJECT=scheduled-tasks
  export SENTRY_RELEASE=$SHORT_SHA
  export SENTRY_ENVIRONMENT=$PROJECT_ID
  export SENTRY_ORG=doitintl
  export SENTRY_AUTH_TOKEN
  SENTRY_AUTH_TOKEN=$(gcloud secrets versions access "latest" --secret "sentry_auth_token")

  curl -sL https://sentry.io/get-cli/ | bash
  sentry-cli releases new --project "$SENTRY_PROJECT" "$SENTRY_RELEASE"
  sentry-cli releases set-commits "$SENTRY_RELEASE" --auto
  sentry-cli releases finalize "$SENTRY_RELEASE"
  sentry-cli releases deploys "$SENTRY_RELEASE" new -e "$SENTRY_ENVIRONMENT"
fi
