import {exportToSvg,exportToBlob,convertToExcalidrawElements,getCommonBounds} from '@excalidraw/excalidraw';
import '@excalidraw/excalidraw/index.css';
(window as any).EXCALIDRAW_ASSET_PATH='/';
const MAX_DIMENSION=4096,MAX_BYTES=20*1024*1024;
(globalThis as any).renderScene=async({scene,format,assets}:any)=>{
 const elements=scene.elements.filter((e:any)=>!e.isDeleted);
 const files=Object.fromEntries(Object.entries(assets).map(([id,a]:[string,any])=>[id,{id,mimeType:a.mimeType,dataURL:`data:${a.mimeType};base64,${a.data}`,created:scene.files[id].created}]));
 const appState={...scene.appState,exportBackground:true,viewBackgroundColor:scene.appState.viewBackgroundColor??'#ffffff',exportWithDarkMode:false,exportEmbedScene:false};
 const options={elements,appState,files,exportPadding:10};
 const svg=await exportToSvg({...options,exportWithDarkMode:false,exportEmbedScene:false,skipInliningFonts:false});
 const width=Number(svg.getAttribute('width')),height=Number(svg.getAttribute('height'));
 if(width>MAX_DIMENSION||height>MAX_DIMENSION)throw new Error('RENDER_LIMIT: Export exceeds 4096 pixels.');
 // The official font loader and inline font export run before rasterization.
 await document.fonts.ready;
 let blob:Blob;
 if(format==='svg')blob=new Blob([svg.outerHTML],{type:'image/svg+xml'});
 else blob=await exportToBlob({...options,mimeType:'image/png',getDimensions:(w,h)=>({width:w,height:h,scale:1})});
 if(blob.size>MAX_BYTES)throw new Error('RENDER_LIMIT: Artifact exceeds 20 MiB.');
 const data=await new Promise<string>((resolve,reject)=>{const reader=new FileReader();reader.onerror=reject;reader.onload=()=>resolve(String(reader.result).split(',')[1]);reader.readAsDataURL(blob);});
 return {mimeType:format==='svg'?'image/svg+xml':'image/png',data};
};

(globalThis as any).normalizeNative=async(diagram:any)=>{
 await exportToSvg({elements:diagram.elements,appState:{},files:{},skipInliningFonts:true});
 const elements=convertToExcalidrawElements(diagram.elements,{regenerateIds:false});
 for(const element of elements){
  const original=diagram.elements.find((e:any)=>e.id===element.id);
  Object.assign(element,{version:original.version,versionNonce:original.versionNonce,updated:original.updated,seed:original.seed});
  if(element.type==='text'){
   // The skeleton API interprets x/y as alignment anchors, whereas complete
   // native text stores its top-left box. Retain the original box center.
   const scale=Math.min(1,original.width/Math.max(element.width,1),original.height/Math.max(element.height,1));
   element.fontSize*=scale;element.width*=scale;element.height*=scale;
   element.x=original.x+(original.width-element.width)/2;
   element.y=original.y+(original.height-element.height)/2;
  }
 }
 if(elements.length!==diagram.elements.length||elements.some(e=>e.type==='image')||elements.some(e=>!diagram.elements.some((original:any)=>original.id===e.id)))throw new Error('Native conversion mismatch.');
 const [x,y,x2,y2]=getCommonBounds(elements);
 return {...diagram,elements,bounds:{x,y,width:x2-x,height:y2-y}};
};
