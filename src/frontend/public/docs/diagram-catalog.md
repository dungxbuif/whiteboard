# Editable diagram catalog

Collaboration v2 includes a local diagram compiler. It creates editable shapes, labels and connectors; no AI provider key is needed. Runtime/MCP integration is delivered separately from this package.

Choose a preset with `listDiagramTypes()`, fetch its data schema and example with `getDiagramTemplate(id)`, validate with `validateDiagram(input)`, then call `await compileDiagram(input)`.

```json
{"preset":"flowchart","instanceId":"checkout-v1","schemaVersion":1,"theme":"pastel","direction":"LR","data":{"nodes":[{"id":"cart","label":"Review cart"},{"id":"pay","label":"Pay","kind":"process"}],"edges":[{"id":"next","from":"cart","to":"pay","label":"Continue"}]}}
```

Use a new `instanceId` for a second diagram. Reuse it when retrying the same generation; IDs remain stable. Themes are `sketch`, `clean` and `pastel`; direction is `LR` or `TB`. Compilation returns `{elements,files,bounds,warnings}` and does not save the result. A caller must submit authorized scene commands separately.

Sequence diagrams keep participants across the top and messages in order below. Journeys keep stages across the top and actions in actor rows; row spacing includes wrapped actor names and experience scores. These two ordered layouts retain that orientation; direction controls graph, timeline and swimlane layouts.

Presets: flowchart, sequence, architecture, deployment, infrastructure, c4-context, c4-container, c4-component, erd, class, state, mindmap, timeline, user-journey, swimlane, org-chart, network, data-flow, agent-workflow. Each template describes its accepted data fields. Groups are editable boundaries, tree links are parent relationships, ER fields and class compartments remain text inside editable native boxes.

| Family | Required data | Useful optional fields |
| --- | --- | --- |
| Graph / C4 / architecture / network / agent | `nodes[{id,label}]`, `edges[{id,from,to}]` | node `kind`, `description`, `technology`, `group`; edge `label`, `kind`; `groups[{id,label}]` |
| Sequence | `participants[{id,label}]`, `messages[{id,from,to,label}]` | participant kind `actor`; message kind `return` or `async` |
| ERD | `entities[{id,label,fields:[{name,type}]}]`, `relations[{id,from,to}]` | field key `PK`, `FK`, `UK`; relation `cardinality` |
| Class | `classes[{id,label,members:[],methods:[]}]`, `relations[{id,from,to}]` | relation kind `inheritance`, `composition`, `dependency`, `association` |
| State | `nodes[{id,label}]`, `edges[{id,from,to}]` | node kind `start`, `state`, `end`; transition `label` |
| Tree / organization chart | `nodes[{id,label}]` | `parent`, `role` |
| Timeline | `events[{id,label,date}]` | `description`; input order is chronological order |
| Journey | `actors[{id,label}]`, `stages[{id,label,steps:[{id,actor,label}]}]` | experience `score` from 1 to 5 |
| Swimlane | `lanes[{id,label}]`, `steps[{id,lane,label}]`, `edges[{id,from,to}]` | step kind `decision`; edge `label` |

Use the discovered schema for the exact allowed values. Unknown properties are rejected, which catches misspelled fields early. Original native motifs include service boxes, person ellipses, decision diamonds and technology/role labels. No remote artwork or plugins are loaded.

Validation errors include a code, JSON path and corrective message. Missing endpoints and duplicate semantic IDs are errors. Graph cycles generate a warning and remain drawable; parent cycles in tree presets are rejected. Long labels wrap and produce a warning. Unicode is retained. Limits are 500 nodes, 1,000 edges, 50,000 JSON characters and 2,000 characters per label.

Mermaid support is intentionally a small native subset: simple flowchart node declarations/connections and sequence participants/messages. Unsupported directives, HTML, styling, diagrams or grammar produce `UNSUPPORTED_NATIVE_DIAGRAM` with a typed preset alternative. There is no image fallback. Full exports and browser rendering are separate D4 work.

Agent-readable source: this Markdown file is mirrored at `/docs/diagram-catalog.md`. Planned MCP discovery: `list_diagram_types`, `get_diagram_template`, `validate_diagram`; resources `whiteboard://diagrams/catalog` and `whiteboard://diagrams/{id}/schema`. Do not assume those tools are live merely because this compiler package is installed.

Verification: 40 compiler tests pass locally, including all 19 presets, three themes, both direction settings and long actor names across journey rows. Browser rendering, live scene persistence and exports require the D3/D4/E1 integration checks before release.
