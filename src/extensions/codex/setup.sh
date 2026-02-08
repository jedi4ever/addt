#!/bin/bash
echo "Setup [codex]: Initializing OpenAI Codex environment"

printenv OPENAI_API_KEY | codex login --with-api-key
