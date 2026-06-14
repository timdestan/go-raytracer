#!/usr/bin/bash
#
# Runs raytracer benchmarks on working copy vs committed state, showing
# statistics.
#
# Prereqs: 
# - go (obviously)
# - benchstat (go install golang.org/x/perf/cmd/benchstat@latest)


set -o errexit
set -o nounset
set -o pipefail


NUM_TRIALS=20
BASE_FILE=output/bench_baseline.txt
MODIFIED_FILE=output/bench_modified.txt
BENCH_PATTERN=.

# git stash returns a success code even if nothing was stashed.
# So compare the stash before and after to see if a change was made.

STASH_BEFORE=$(git rev-parse --verify refs/stash 2>/dev/null || echo "none")
git stash push -u -m "bench-script-temp"
STASH_AFTER=$(git rev-parse --verify refs/stash 2>/dev/null || echo "none")

if [[ "$STASH_BEFORE" == "$STASH_AFTER" ]]; then
    echo "Error: no local changes found to benchmark against baseline." >&2
    exit 1
fi

# Ensure we restore on exit, even if benchmarks fail
trap 'git stash pop' EXIT

echo "Running benchmarks on baseline" >&2

go test -bench="$BENCH_PATTERN" -run=^$ -count=$NUM_TRIALS > "$BASE_FILE"

git stash pop
trap - EXIT

echo "Running benchmarks on change" >&2

go test -bench="$BENCH_PATTERN" -run=^$ -count=$NUM_TRIALS > "$MODIFIED_FILE"

benchstat "$BASE_FILE" "$MODIFIED_FILE"
