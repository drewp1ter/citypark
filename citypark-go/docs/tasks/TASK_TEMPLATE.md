# Template for writing a new task file

Copy this file to `docs/tasks/NN-short-name.md`, fill every section, and add a line to `PROGRESS.md`. A task must be doable in one session by a model that sees only `CONVENTIONS.md`, this file, and the files under "Read first".

Sizing: one package or one vertical slice, about 300 lines of Go including tests, at most about five files, and a task file of about 150 lines.

---

# Task NN: <title>

## Goal
One or two sentences: what exists after this task that did not exist before.

## Depends on
Task numbers that must be ticked first, or "Nothing".

## Read first
`docs/tasks/CONVENTIONS.md` and at most three existing files, by exact path.

## Files to create
- exact paths

## Files to change
- exact paths, each with what changes (leave the section out when there are none)

## Spec
Everything the code must do, written so that nothing has to be guessed:
- exported types and function signatures in a Go code block;
- SQL statements in full, with `$1` placeholders;
- routes as a table: method, path, role, behaviour, success status;
- request and response JSON examples;
- every user-facing message, verbatim, in Russian;
- rules for edge cases (empty input, missing row, duplicate, nil slice).

## Tests to write
Required for every task that contains logic. Logic is any code with a branch, a loop, a calculation, a parsing or formatting step, an SQL statement, or an error path. Only tasks that are pure configuration or documentation may leave this section out, and then they say why.

List each test by name with its input and its expected result:
- `TestXxx`: <setup> → <expected value, row, status or body>.

Cover, for each function or route in the Spec:
- the normal case;
- each error path the Spec names (one test per message or status);
- the edge cases the Spec names;
- for routes: no token (401), wrong role (403), invalid body (400), missing row (404), where they apply;
- for SQL: what is written to the database, checked by reading it back.

Use table tests for pure functions. Say which helpers to use (`testutil.NewDB`, `testutil.DoJSON`, `clock.Fixed`, fakes).

## API documentation
Only for tasks with HTTP endpoints: list the endpoints whose sections the executor must add or update in `docs/API.md`, and any behaviour note the frontend needs (for example "posting twice returns the same id").

## Done when
The commands in CONVENTIONS "Done means" pass, coverage of the changed packages is at least 80%, `docs/API.md` covers the task's endpoints, plus any extra command for this task.

## Out of scope
What the executor must not do or touch, and which later task does it.
