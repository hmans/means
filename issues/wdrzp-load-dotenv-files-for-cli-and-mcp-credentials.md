---
created_at: "2026-10-28T08:28:12Z"
data:
  priority: normal
  status: done
  type: feature
id: wdrzp
title: Load dotenv files for CLI and MCP credentials
updated_at: "2026-10-28T08:28:12Z"
---
Load collection --path/.env first, then working-directory .env. Preserve existing environment variables, including empty values. Support CLI and MCP startup; keep help/version independent. Document precedence and test errors without exposing secrets.

Implemented optional dotenv loading for CLI and MCP startup. Precedence: process environment (including empty values), collection .env, then working-directory .env. Missing files are ignored; errors omit file contents. Added guide and MCP help text, release note, and regression coverage for precedence, parsing, error handling, help, and MCP startup. Validation: mise test passes.