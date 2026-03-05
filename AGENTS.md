<!-- CLAUDE-CONF-CODEX-MIGRATION START -->
## Claude -> Codex migrated rules
- Всегда отвечай пользователю на русском языке.
- Не выполняй 'git commit' и 'git push', если пользователь явно не попросил.
- Для задач используй 'bd' (Beads), а не markdown TODO.
- Для package manager используй 'bun' вместо 'npm', где возможно.
- Перед массовыми/рискованными изменениями сначала добавляй файлы в staging для быстрой точки отката.
- Не запускать генерацию/изменение API схем без явного запроса; после API-изменений валидировать схемы 'npx @redocly/cli lint'.
- Не оставляй комментарии в коде без необходимости.

## Claude -> Codex migrated skills
- 'test-api-requests'
- 'verify-jenkins-build'
- 'browser-testing'
- 'figma-integration'
- 'debug-tracing'

## Claude -> Codex migrated MCP
- Глобальные MCP перенесены в '/Users/falkomer/.codex/config.toml'.
- Project-specific MCP перенесены в '.codex/mcp.toml' каждого проекта.

Source of truth: '/Users/falkomer/claude-conf/.claude/' and '/Users/falkomer/.claude.json'
<!-- CLAUDE-CONF-CODEX-MIGRATION END -->
