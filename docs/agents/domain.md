# Domain docs

This repository uses a single-context domain documentation layout:

- `GLOSSARY.md` at the repository root.
- Architecture decision records in `docs/adr/`.

## Before exploring

Read the glossary and any ADRs relevant to the area being changed.

If these files do not exist, proceed silently. Do not flag their
absence or suggest creating them upfront. The domain-modeling skill,
also used by grill-with-docs and improve-codebase-architecture,
creates them lazily as terms and decisions are resolved.

## Use the glossary's vocabulary

Use defined domain terms in issue titles, proposals, hypotheses,
and test names. Avoid synonyms the glossary explicitly rejects.

If a needed concept is missing, reconsider whether it belongs in
the domain or note the gap for domain-modeling.

## Flag ADR conflicts

Explicitly identify any proposal that contradicts an existing ADR,
and explain why the decision should be reconsidered.
