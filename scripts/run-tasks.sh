#!/usr/bin/env bash
# Runs the numbered tasks of citypark-go, citypark-modern and citypark-admin one by one, each in a fresh headless
# Claude Code session on Sonnet. Picks the first unchecked task of each project whose "(needs <project> task NN)"
# dependencies are ticked, in the order backend → site → admin. Stops when a task is not ticked after its session
# (a blocker was written to PROGRESS.md) or when nothing is runnable.
#
# Usage: scripts/run-tasks.sh [max-tasks]        (default 50)
#        scripts/run-tasks.sh --dry-run         (show the next task and its prompt, run nothing)
# Env:   CLAUDE_FLAGS  extra flags for `claude`, e.g. "--dangerously-skip-permissions" inside a sandbox/VM
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
DRY=0
if [ "${1:-}" = --dry-run ]; then DRY=1; shift; fi
MAX=${1:-50}
LOG_DIR="$ROOT/.task-logs"
mkdir -p "$LOG_DIR"

PROJECTS=(citypark-go citypark-modern citypark-admin)

progress() { echo "$ROOT/$1/docs/tasks/PROGRESS.md"; }

is_ticked() { # project task
	grep -qE "^- \[(x|-)\] $2 " "$(progress "$1")"
}

# Prints the first unchecked task of a project, or nothing.
first_open() {
	grep -m1 -E '^- \[ \] ' "$(progress "$1")" || true
}

# True when every "(needs <project> task(s) A, B)" in the PROGRESS line is ticked.
deps_ok() { # line
	local line=$1 dep proj nums n
	while read -r dep; do
		[ -z "$dep" ] && continue
		proj=$(sed -E 's/needs (citypark-[a-z]+) tasks? .*/\1/' <<<"$dep")
		nums=$(sed -E 's/needs citypark-[a-z]+ tasks? //' <<<"$dep" | tr -d ' ')
		for n in ${nums//,/ }; do
			is_ticked "$proj" "$n" || return 1
		done
	done < <(grep -oE 'needs citypark-[a-z]+ tasks? [0-9a-z, ]+' <<<"$line" || true)
}

prompt_for() { # project task
	local project=$1 nn=$2
	if [ "$project" = citypark-go ]; then
		# The backend's own prompt template, between the ```text fences.
		sed -n '/^```text$/,/^```$/p' "$ROOT/citypark-go/docs/tasks/PROMPT.md" | sed '1d;$d' | sed "s/{{NN}}/$nn/g"
	else
		cat <<EOF
You are working in $project. Do exactly one task: task $nn.
Read CLAUDE.md, docs/tasks/CONVENTIONS.md, docs/tasks/$nn-*.md and only the files that task lists under "Read first".
Check in docs/tasks/PROGRESS.md that every task it depends on is ticked (also the citypark-go tasks it needs, in
../citypark-go/docs/tasks/PROGRESS.md). If one is not, stop.
Do only that task, run every command under "Done when", tick it in docs/tasks/PROGRESS.md and make one commit
"task $nn: <task title>". You cannot ask questions in this session: if something blocks you (a missing backend, a step
that needs the user, a contradiction), write one line under the task in PROGRESS.md saying what blocked you, do not
tick it, and stop. Finish with a short report: files changed, tests added, command results, open doubts.
EOF
	fi
}

ran=0
while [ "$ran" -lt "$MAX" ]; do
	picked=""
	for p in "${PROJECTS[@]}"; do
		line=$(first_open "$p")
		[ -z "$line" ] && continue
		if deps_ok "$line"; then picked="$p"; break; fi
	done
	if [ -z "$picked" ]; then
		echo "Nothing runnable. Open tasks:"; for p in "${PROJECTS[@]}"; do echo "  $p: $(first_open "$p")"; done
		exit 0
	fi

	nn=$(sed -E 's/^- \[ \] ([0-9a-z]+) .*/\1/' <<<"$line")
	if [ "$DRY" = 1 ]; then echo "next: $picked task $nn"; prompt_for "$picked" "$nn"; exit 0; fi
	log="$LOG_DIR/$picked-$nn.log"
	echo "=== $picked task $nn  (log: $log)"
	(cd "$ROOT/$picked" && prompt_for "$picked" "$nn" | claude -p --model sonnet ${CLAUDE_FLAGS:-} >"$log" 2>&1) || true
	tail -n 20 "$log"

	if ! is_ticked "$picked" "$nn"; then
		echo "!!! $picked task $nn was not ticked. See $log and $(progress "$picked")."
		exit 1
	fi
	ran=$((ran + 1))
done
echo "Stopped after $MAX tasks."
