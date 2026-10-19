import {listDiagramTypes,getDiagramTemplate} from '../catalog.js';
import type {DiagramInput} from '../types.js';
/** Fixtures are generated from the same public examples agents discover. */
export const canonicalFixtures=():DiagramInput[]=>listDiagramTypes().map(p=>({preset:p.id,instanceId:`canonical-${p.id}`,schemaVersion:1,theme:'pastel',direction:'LR',data:getDiagramTemplate(p.id).example}));
