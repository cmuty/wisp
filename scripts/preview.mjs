// Render actual --preview output to SVG. Node is needed only for this design tool.
import {execFileSync} from 'node:child_process';
import {mkdirSync,writeFileSync} from 'node:fs';
import {fileURLToPath} from 'node:url';
import path from 'node:path';
const root=path.resolve(path.dirname(fileURLToPath(import.meta.url)),'..');
const out=path.join(root,'docs','previews');mkdirSync(out,{recursive:true});
const binary=path.resolve(root,process.argv[2]||'dist/wisp.exe');
const xml=s=>s.replaceAll('&','&amp;').replaceAll('<','&lt;').replaceAll('>','&gt;').replaceAll('"','&quot;');
function frame(stage,theme){
 const raw=execFileSync(binary,['--preview',stage,'--preview-theme',theme],{encoding:'utf8'});
 writeFileSync(path.join(out,`${stage}-${theme}.svg`),terminalSvg(raw));
}
export function terminalSvg(raw){
 const allRows=raw.trimEnd().split('\n');const width=Math.max(900,Math.ceil(48+Math.max(...allRows.map(row=>[...row.replace(/\x1b\[[0-9;]*m/g,'')].length))*9.6));const height=Math.max(680,54+allRows.length*20);
 let svg=`<svg xmlns="http://www.w3.org/2000/svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}"><rect width="${width}" height="${height}" fill="#0C0F16"/><g font-family="Cascadia Mono,Consolas,monospace" font-size="16" xml:space="preserve">`;
 for(const [row,line] of raw.trimEnd().split('\n').entries()){
  let col=0,color='#E4EAF1',bold=false,last=0;const sgr=/\x1b\[([0-9;]*)m/g;
  function run(s){if(!s)return;const length=[...s].length;svg+=`<text x="${24+col*9.6}" y="${34+row*20}" fill="${color}" font-weight="${bold?600:400}" textLength="${length*9.6}" lengthAdjust="spacingAndGlyphs">${xml(s)}</text>`;col+=length;}
  for(const match of line.matchAll(sgr)){run(line.slice(last,match.index));const n=match[1].split(';').map(Number);for(let i=0;i<n.length;i++){if(n[i]===0){color='#E4EAF1';bold=false}else if(n[i]===1){bold=true}else if(n[i]===38&&n[i+1]===2){color=`rgb(${n[i+2]},${n[i+3]},${n[i+4]})`;i+=4}else if(n[i]===38&&n[i+1]===5){const v=n[i+2];const c=v>=232?8+(v-232)*10:180;color=`rgb(${c},${c},${c})`;i+=2}}last=match.index+match[0].length;}run(line.slice(last));
 }
 svg+='</g></svg>';return svg;
}
if(process.argv[1]&&path.resolve(process.argv[1])===fileURLToPath(import.meta.url)){
for(const theme of ['mist','ember','haze'])frame('welcome',theme);
for(const stage of ['username','theme','sources','ready','disconnected','credential','completion','settings','playlists','lyrics','butterfly-flight'])frame(stage,'mist');
console.log('Generated terminal previews in docs/previews.');

}
