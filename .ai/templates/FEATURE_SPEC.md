# [FEATURE_NAME]

**Status:** Not Started | In Progress | Complete
**Created:** [DATE]
**Last updated:** [DATE]

---

## 1. Overview

- 2-3 paragraphs: what changes, why, and impact
- Dependencies with existing features
- Highlight BREAKING changes

---

## 2. Current State

| Layer | Status |
|-------|--------|
| [Component] | ✅ Exists / ❌ Missing / ⚠️ Partial |

**Contract changes:**
- [List changes]

---

## 3. Data Model

For database changes:
- a) Migration names (e.g. `000005_feature_tables`)
- b) SQL CREATE TABLE with inline comments
- c) SQL for indexes and constraints
- d) SQL for data migration (if applicable)

For Go-only changes: skip to "Technical Requirements"

---

## 4. [Models/Configs/Hierarchies]

- New models (roles, types, categories): present as table or diagram
- Configs: explain each parameter and default
- Hierarchies: show level/weight/priority with numeric values

---

## 5. [Resource/Permission/Config] Catalog

- Table showing what each entity/role/type can access
- "Compatibility" column if there are breaking changes

---

## 6. Main Flows

For each material flow (login, creation, approval, etc):
- a) Flow name
- b) Step sequence (1, 2, 3...)
- c) Decisions (if X, then Y)
- d) Expected responses (JSON/structure)
- e) Error cases and HTTP codes

---

## 7. Technical Requirements

- Section per code file that changes
- For each file: Go types + interfaces + new/changed functions
- Implementation examples for complex logic
- New domain errors

---

## 8. Expected API

| Method | Route | Description | Auth/Validation |
|--------|-------|-------------|-----------------|
| | | | |

For each route: request (JSON) + response (JSON) + possible errors (3xx-5xx)

---

## 9. Impact Mapping

| Existing Endpoint | Change | New Behavior |
|-------------------|--------|--------------|
| | | |

---

## 10. Recommended Implementation Order

Split into phases (Phase 1, 2, 3...):
- Each phase can be done independently
- Phase N+1 depends on Phase N being complete
- Each step must reference specific file/function/table
