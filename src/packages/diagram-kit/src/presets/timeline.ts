import {object,array,id,str} from '../schema';
import type {TimelineData} from '../types';
export const timelineSchema=object({events:array(object({id:id(),label:str(),date:str(100),description:str()},['id','label','date']),500,1)});
export const timelineExample:TimelineData={events:[{id:'discover',label:'Discover',date:'Week 1',description:'Research needs'},{id:'prototype',label:'Prototype',date:'Week 2',description:'Test the flow'},{id:'launch',label:'Launch',date:'Week 4',description:'Release and observe'}]};
