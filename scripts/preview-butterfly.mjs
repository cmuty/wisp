// Render real wisp preview frames; the GIF uses the same text sprites and route as the app.
import {terminalSvg} from './preview.mjs';
import {execFileSync} from 'node:child_process';
import {mkdirSync,writeFileSync,existsSync} from 'node:fs';
import {createRequire} from 'node:module';
import path from 'node:path';
const require=createRequire(import.meta.url);
const {Resvg}=require(process.env.WISP_RESVG);
const binary=path.resolve(process.argv[2]||'dist/wisp-0.3.5.exe');
const frames=path.resolve('.preview-cache','butterfly-035');
mkdirSync(frames,{recursive:true});
for(let frame=0;frame<600;frame+=2){
 if(process.argv.includes('--reuse')&&existsSync(path.join(frames,`${String(frame).padStart(4,'0')}.png`)))continue;
 const raw=execFileSync(binary,['--preview','butterfly-search','--preview-frame',String(frame)],{encoding:'utf8'});
 const svg=terminalSvg(raw);
 const png=new Resvg(svg,{font:{loadSystemFonts:true}}).render().asPng();
 writeFileSync(path.join(frames,`${String(frame).padStart(4,'0')}.png`),png);
 if(frame===0)writeFileSync('docs/previews/butterfly-detail.png',png);
}
execFileSync('python',['-c',`
from PIL import Image,ImageDraw
from pathlib import Path
root=Path(__import__('sys').argv[1])
paths=sorted(root.glob('*.png'))
palette=Image.open(paths[0]).convert('RGB').quantize(colors=64)
frames=[Image.open(p).convert('RGB').quantize(palette=palette) for p in paths]
frames[0].save('docs/previews/butterfly-0.3.5.gif',save_all=True,append_images=frames[1:],duration=70,loop=0,optimize=True,disposal=1)
indices=[0,60,110,165,225,290]
canvas=Image.new('RGB',(1350,680),(12,15,22))
for i,n in enumerate(indices):
 image=frames[n].resize((450,340))
 canvas.paste(image,((i%3)*450,(i//3)*340))
canvas.save('docs/previews/butterfly-contact.png')
`,frames]);
console.log('Rendered butterfly animation and contact sheet.');
