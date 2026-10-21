# citypark-go

Go rewrite of a restaurant ordering backend (menu synced from the iiko POS, payments through MONETA.RU, staff notifications through Telegram). The work is split into small numbered tasks.

## How to work in this repository

1. Read `docs/tasks/CONVENTIONS.md`.
2. Open `docs/tasks/PROGRESS.md` and take the first unchecked task.
3. Read that task file in `docs/tasks/` and the files it lists under "Read first". Do not read other task files or `docs/design/` unless the task says so.
4. Do only that task. Run every command under "Done when".
5. Tick the task in `PROGRESS.md` and make one commit: `task NN: <title>`.

Endpoints are documented for the frontend in `docs/API.md`; a task that adds or changes an endpoint updates it.

The prompt to start a task session is in `docs/tasks/PROMPT.md`; the template for adding a task is `docs/tasks/TASK_TEMPLATE.md`.

Do one task per session. If a task cannot be completed as written, stop and write the reason under the task's line in `PROGRESS.md` instead of improvising.

Never open the files in `internal/iiko/testdata/`: they are large JSON fixtures.
