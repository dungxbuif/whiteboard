import {defineConfig} from 'vite';
import {cpSync,readFileSync,mkdirSync} from 'node:fs';
import {createHash} from 'node:crypto';
import {fileURLToPath} from 'node:url';
const root=fileURLToPath(new URL('./render',import.meta.url));
const outDir=fileURLToPath(new URL('./render-dist',import.meta.url));
export default defineConfig({root,publicDir:false,build:{outDir,emptyOutDir:true},plugins:[{name:'bundle-reviewed-native-fonts',closeBundle(){
 const manifest=JSON.parse(readFileSync(new URL('./render-fonts.json',import.meta.url),'utf8'));
 for(const asset of manifest.assets){
  const source=fileURLToPath(new URL('../node_modules/@excalidraw/excalidraw/dist/prod/fonts/'+asset.path,import.meta.url));
  if(createHash('sha256').update(readFileSync(source)).digest('hex')!==asset.sha256)throw new Error('Font hash changed: '+asset.path);
  const target=outDir+'/fonts/'+asset.path;mkdirSync(target.slice(0,target.lastIndexOf('/')),{recursive:true});cpSync(source,target);
 }
 cpSync(fileURLToPath(new URL('./licenses',import.meta.url)),outDir+'/licenses',{recursive:true});
 cpSync(fileURLToPath(new URL('./render-fonts.json',import.meta.url)),outDir+'/render-fonts.json');
}}]});
