# serve-api Specification (delta)

## MODIFIED Requirements

### Requirement: Endpoints mirror the store

Reads SHALL return the same JSON shapes as MCP read tools:

- `GET /api/projects` -> `[]Project`
- `POST /api/projects` `{title, goal?, tags?, id?}` -> `Project`
  (`id` pins a client UUID: 400 on garbage, 409 on collision; numbers stay server-assigned)
- `GET /api/projects/{ref}` -> `ProjectData` (ref = number or UUID)
- `PATCH /api/projects/{ref}` `{title?, goal?, status?, tags?}` -> `Project`
  (`title` is validated like creation: empty string is `400`; absent fields
  keep their stored value, `tags: []` clears the list)
- `DELETE /api/projects/{ref}` -> trash, `200 {"trashed": name}`
- `GET /api/projects/{ref}/steps` -> `[]Step`
- `POST /api/projects/{ref}/steps` `{title}` -> `Step` (slug id,
  duplicate title yields `409`)
- `POST /api/projects/{ref}/steps/{step}/{start,review,done}` ->
  `Step` with domain lifecycle validation (`422` on illegal transition)
- `PATCH /api/projects/{ref}/steps/{step}` `{title}` -> `Step`
  (rename; the step id stays stable, empty title is `400`, a title whose slug
  collides with another step is `409`)
- `DELETE /api/projects/{ref}/steps/{step}` -> `200`
- `GET /api/projects/{ref}/blockers` -> `[]Blocker`
- `POST /api/projects/{ref}/steps/{step}/blockers` `{title, reason?}`
  -> `Blocker` (default status `waiting`)
- `PATCH /api/projects/{ref}/steps/{step}/blockers/{blk}`
  `{title?, reason?}` -> `Blocker`
  (text edit; the blocker id stays stable, an empty title is `400`, a title
  whose slug collides inside the step is `409`, no field at all is `400`)
- `POST .../blockers/{blk}/resolve` -> `Blocker`
- `DELETE .../blockers/{blk}` -> `200`
- `GET /api/projects/{ref}/decisions` -> `[]Decision`
- `POST /api/projects/{ref}/decisions` `{title, reason?}` -> `Decision`
- `GET /api/briefing?date=&project=` -> briefing JSON

Errors are JSON `{"error": msg}` with codes:
`400` bad body, `401` auth, `404` unknown ref, `409` duplicate,
`422` lifecycle violation, `500` store failure.

#### Scenario: lifecycle violation surfaces as 422

- **WHEN** the client sends `done` for a step that is not in review
- **THEN** the server answers `422` with a JSON error body

#### Scenario: PATCH renames without touching other fields

- **WHEN** the client sends `PATCH /api/projects/{ref}` with `{"title": "new"}`
- **THEN** the response is `200` with the updated `Project`, goal, tags and status are unchanged, and a following `GET` returns the new title

#### Scenario: PATCH refuses an empty title

- **WHEN** the client sends `PATCH /api/projects/{ref}` with `{"title": ""}`
- **THEN** the server answers `400` with a JSON error body and the stored project keeps its old title

#### Scenario: blocker text edit keeps the id

- **WHEN** the client sends `PATCH .../blockers/{blk}` with `{"title": "new", "reason": "also new"}`
- **THEN** the response is `200` with the updated `Blocker`, its id is unchanged, and a following `GET .../steps` shows the new text
