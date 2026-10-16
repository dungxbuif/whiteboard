import {object,array,id,str,enumeration} from '../schema.js';
import type {ClassData} from '../types.js';
export const classSchema=object({classes:array(object({id:id(),label:str(),members:array(str(),100),methods:array(str(),100)}),500,1),relations:array(object({id:id(),from:id(),to:id(),label:str(),kind:enumeration('association','inheritance','composition','dependency')},['id','from','to']),1000)});
export const classExample:ClassData={classes:[{id:'document',label:'Document',members:['+ id: UUID','+ title: string'],methods:['+ open(): Scene']},{id:'scene',label:'Scene',members:['+ elements: Element[]'],methods:['+ apply(command): Receipt']}],relations:[{id:'owns',from:'document',to:'scene',label:'owns',kind:'composition'}]};
