import {object,array,id,str,enumeration} from '../schema.js';
import type {GraphData} from '../types.js';
export const stateSchema=object({nodes:array(object({id:id(),label:str(),kind:enumeration('start','state','end')},['id','label']),500,1),edges:array(object({id:id(),from:id(),to:id(),label:str()},['id','from','to']),1000)});
export const stateExample:GraphData={nodes:[{id:'draft',label:'Draft',kind:'start'},{id:'review',label:'In review',kind:'state'},{id:'published',label:'Published',kind:'end'}],edges:[{id:'submit',from:'draft',to:'review',label:'Submit'},{id:'approve',from:'review',to:'published',label:'Approve'},{id:'revise',from:'review',to:'draft',label:'Request changes'}]};
