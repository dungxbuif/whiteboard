import katex from 'katex';
import {parseMermaidToExcalidraw} from '@excalidraw/mermaid-to-excalidraw';
import {convertToExcalidrawElements,exportToSvg} from '@excalidraw/excalidraw';
(globalThis as any).fixture={katex,parseMermaidToExcalidraw,convertToExcalidrawElements,exportToSvg};
