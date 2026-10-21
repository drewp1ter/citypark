# Prompt template for running one task

Start a fresh session in the repository root for every task. Copy the block below, replace `{{NN}}` with the task number from `PROGRESS.md` (for example `07` or `25b`), and send it as the first message.

```text
You are working in the Go repository citypark-go. Do exactly one task: task {{NN}}.

Read, in this order:
1. docs/tasks/CONVENTIONS.md
2. docs/tasks/{{NN}}-*.md
3. Only the files that task lists under "Read first".
4. docs/API.md, only if the task adds or changes an HTTP endpoint.

Do not read other task files, docs/design/ (unless the task tells you to), or internal/iiko/testdata/.

Before writing code, check in docs/tasks/PROGRESS.md that every task named under "Depends on" is ticked. If one is not, stop and tell me.

How to work:
1. Create or change only the files the task lists. Keep every exported name, signature, SQL statement, route, status code and Russian message exactly as the task spells it. Other tasks depend on them.
2. Write the tests the task lists under "Tests to write", with the test names it gives.
3. Cover all logic you write with tests. Logic is any code with a branch, a loop, a calculation, a parsing or formatting step, an SQL statement, or an error path. If you write such code and the task's test list does not reach it, add a test for it. Each test must check a result (a returned value, a database row, a response status and body), not only that the code runs.
4. Run the tests often while you work. When a test fails, fix the code. Change a test only when the test itself contradicts the task file. Never delete a test, skip it, or loosen its check to make it pass.
5. Document every API endpoint you add or change, for the frontend developers. Open docs/API.md, follow the format shown there, and write or update one section per endpoint under its area heading: method and path, role, parameters and request fields with their rules, a request example, the success status with a response example, every error status with its exact message, and notes on behaviour the frontend must know. Take the examples from real output of your tests, so the field names and values match what the server returns. If the task has no endpoints, do not touch docs/API.md.
6. When you are done, run these and fix everything they report:
   gofmt -l .
   go build ./...
   go vet ./...
   go test -p 1 ./...
   go test -p 1 -cover ./internal/<each package you changed>/...
   Statement coverage of each internal package you changed must be at least 80%. If it is lower, run
   go test -p 1 -coverprofile=/tmp/cover.out ./internal/<package>/... && go tool cover -func=/tmp/cover.out
   find the functions with low coverage and add tests for them. The cmd/ packages are exempt.
   Also run any extra command the task lists under "Done when".
7. Tick the task in docs/tasks/PROGRESS.md and make one commit named "task {{NN}}: <task title>".

Database tests need PostgreSQL on port 5433. Start it with: make db-test-up
If it cannot be started, stop and tell me. Do not skip the database tests and do not replace the database with a mock.

Stop and ask me instead of guessing when:
- the task contradicts CONVENTIONS.md or code from an earlier task;
- a signature from an earlier task is different from what this task expects;
- a "Done when" command still fails after you have tried to fix it three times;
- you think a file outside the task's list has to change.
In those cases write one line under the task in PROGRESS.md that says what blocked you, and do not tick it.

Finish with a short report:
- files created and changed;
- the tests you added beyond the task's list, and what each covers;
- the endpoints you documented in docs/API.md;
- the final output of go test -p 1 -cover for the packages you changed;
- anything you were unsure about.
```

## Notes for the person running the tasks

- One task per session keeps the context small. Do not ask for two tasks in one session.
- If a session ends with a blocker, fix the task file (or the earlier code) yourself, then rerun the same prompt in a new session.
- After the task is committed, review the diff before starting the next one: later tasks build on these signatures.
