import {object,array,id,str} from '../schema';
import type {TreeData} from '../types';
export const treeSchema=object({nodes:array(object({id:id(),label:str(),parent:id(),role:str()},['id','label']),500,1)});
export const mindmapExample:TreeData={nodes:[{id:'idea',label:'Product idea'},{id:'people',label:'People',parent:'idea'},{id:'design',label:'Design',parent:'idea'},{id:'research',label:'Research',parent:'design'}]};
export const orgExample:TreeData={nodes:[{id:'lead',label:'Alex Chen',role:'Team lead'},{id:'design',label:'Sam Nguyen',role:'Design',parent:'lead'},{id:'engineering',label:'Taylor Park',role:'Engineering',parent:'lead'}]};
