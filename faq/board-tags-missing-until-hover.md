# Why do board card tags / assignees only appear after you hover the card?

Symptom: on the **board (kanban) view**, some cards are missing their tag chips
(and/or assignee avatars) on load. Moving the cursor over an affected card makes
them pop in. The data is correct — it's a render-timing bug, and only the board is
affected (the list/table `TagsCell` doesn't use this path). Reported by Andreas
Batsis (batsis@gmail.com); fixed in PR #668.

**Root cause:** `BoardColumnVirtual.vue` caches each card's resolved entities
(tags, assignees) in `resolvedCardCache` — a `WeakMap` keyed by the **record
object**. `resolveCardEntities` builds that list by mapping `record.tagIds →
tagsStore.getById(id)` and **dropping any id the store can't resolve yet**. The
tag/user stores load asynchronously, so a card that renders *before* its store
populates caches an **empty** entity list. The cache only invalidated on (a) the
record object being replaced or (b) a board field-template change — **not** when
the store finished loading. Worse, a cache *hit* skips the store read, so the
render effect stops even tracking the store. Hovering a card fires the prefetch
(`upsertRecord(record, { background: true })`), which replaces the record object →
WeakMap miss → re-resolve → tags finally appear. That's the "only on hover" tell.

Became visible after the Jun-2026 board perf work (markRaw records + non-blocking
persist + no-persist-on-prefetch) cut the incidental record-replacement churn that
used to mask the stale cache.

**The fix / how to diagnose the next one:** the file already had the invalidation
machinery — `resolvedCardVersion` (a ref bumped by a `watch` to bust every cached
entry) — but it was wired only to `boardFieldTemplate`. The fix added the entity
stores it forgot:

```
watch(
  [() => tagsStore.tagsById, () => usersStore.usersById, () => customFieldsStore.fieldsById],
  () => resolvedCardVersion.value++,
)
```

Rule of thumb: **any per-card cache keyed on record identity that embeds
async-loaded store entities must invalidate when those stores change, not just
when the record object changes.** Renames/recolors already surface via the cached
reactive store objects; the gap is *additions* (an id that resolved to `undefined`
and got filtered out), which the reactive passthrough can't recover. If a future
card-display value is "missing until hover," suspect this cache, not a CSS/opacity
or v-if gate.
