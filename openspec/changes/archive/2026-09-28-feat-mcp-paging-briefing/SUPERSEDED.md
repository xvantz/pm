# Closed as superseded — 2026-09-29

Kept for history. Nothing here was implemented as written, and no delta was
ever applied to the baseline.

## Why it closed

The change bundled three unrelated things behind one proposal:

1. string IDs in JSON-RPC
2. bounded reads for large projects
3. a `briefing` capability spec

On review each turned out to be a different kind of work with a different
verdict, and keeping them together meant one ambiguous answer for three
questions.

**1. String IDs — split out** into `fix-mcp-string-id`. Small, transport-level,
and separable. The rest of this change had nothing to do with it.

**2. Bounded reads — reframed, then shipped** as `feat-mcp-bounded-reads`.
The original proposal asked for "pagination / truncate / cursors". Measured,
that was the wrong frame: `get_project` cost 2 195 B and there is nothing to
page through at that size. The real problem was that a caller asking "where
does this stand" paid for every field of every step. Solved as three read
sizes instead — summary, brief list, single step — which no amount of paging
would have produced.

**3. Briefing capability — shipped** as `chore-briefing-spec`. It found a bug
while being written: on an unparsable `date` the briefing fell back to today
for its arithmetic but still returned the original string in `date`, so the
response claimed a day it had not computed.

## Why archived rather than deleted

The proposal records the reasoning that was wrong, which is worth keeping: it
is the clearest example in the repo of a task that looked urgent for two weeks
and was three smaller decisions wearing one trenchcoat.