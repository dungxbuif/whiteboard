import {LIMITS,type JSONSchema,type Diagnostic} from './types.js';
export const str=(max:number=LIMITS.label):JSONSchema=>({type:'string',minLength:1,maxLength:max});
export const id=()=>str(LIMITS.id);
export const enumeration=(...values:string[]):JSONSchema=>({type:'string',enum:values});
export const object=(properties:Record<string,JSONSchema>,required=Object.keys(properties)):JSONSchema=>({type:'object',properties,required,additionalProperties:false});
export const array=(items:JSONSchema,maxItems:number=LIMITS.nodes,minItems=0):JSONSchema=>({type:'array',items,maxItems,minItems});
export function checkSchema(value:unknown,schema:JSONSchema,path='$',errors:Diagnostic[]=[]):Diagnostic[]{
 const fail=(message:string)=>errors.push({code:'SCHEMA',path,message});
 if(schema.const!==undefined&&value!==schema.const)fail(`Expected ${String(schema.const)}.`);
 if(schema.enum&&!schema.enum.includes(value))fail(`Choose one of: ${schema.enum.join(', ')}.`);
 if(schema.type==='object'){
  if(value===null||typeof value!=='object'||Array.isArray(value)){fail('Expected an object.');return errors;}
  const record=value as Record<string,unknown>;
  for(const field of schema.required??[])if(!Object.hasOwn(record,field))errors.push({code:'SCHEMA',path:`${path}.${field}`,message:'Required field is missing.'});
  for(const [field,item] of Object.entries(record)){if(Object.hasOwn(schema.properties??{},field))checkSchema(item,schema.properties![field],`${path}.${field}`,errors);else if(schema.additionalProperties===false)errors.push({code:'SCHEMA',path:`${path}.${field}`,message:'Unknown field; use the discovered template schema.'});}
 }else if(schema.type==='array'){
  if(!Array.isArray(value)){fail('Expected an array.');return errors;}
  if(value.length<(schema.minItems??0)||value.length>(schema.maxItems??Infinity))fail(`Array must contain ${schema.minItems??0}–${schema.maxItems??'unlimited'} items.`);
  if(schema.items)for(let i=0;i<Math.min(value.length,(schema.maxItems??LIMITS.edges)+1);i++)checkSchema(value[i],schema.items,`${path}[${i}]`,errors);
 }else if(schema.type==='string'){
  if(typeof value!=='string'){fail('Expected plain text.');return errors;}
  if(value.length<(schema.minLength??0)||value.length>(schema.maxLength??Infinity))fail(`Text must contain ${schema.minLength??0}–${schema.maxLength??'unlimited'} characters.`);
 }else if(schema.type==='number'&&(typeof value!=='number'||!Number.isFinite(value)||value<(schema.minimum??-Infinity)||value>(schema.maximum??Infinity)))fail('Expected a finite number in the specified range.');
 return errors;
}
