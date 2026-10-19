import {object,array,id,str,enumeration} from '../schema.js';
import type {SequenceData} from '../types.js';
export const sequenceSchema=object({participants:array(object({id:id(),label:str(),kind:enumeration('actor','participant')},['id','label']),500,1),messages:array(object({id:id(),from:id(),to:id(),label:str(),kind:enumeration('call','return','async')},['id','from','to','label']),1000)});
export const sequenceExample:SequenceData={participants:[{id:'user',label:'User',kind:'actor'},{id:'api',label:'API'},{id:'db',label:'Database'}],messages:[{id:'request',from:'user',to:'api',label:'Open scene'},{id:'query',from:'api',to:'db',label:'Load document'},{id:'result',from:'db',to:'api',label:'Document',kind:'return'},{id:'response',from:'api',to:'user',label:'Scene ready',kind:'return'}]};
